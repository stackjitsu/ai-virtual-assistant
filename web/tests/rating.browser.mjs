// Opt-in actual-backend regression; never launched by the Node unit glob.
// Usage: node web/tests/rating.browser.mjs /absolute/assistant /absolute/fresh-evidence-dir
// Only loopback Go/SQLite and Chromium are used. No mock API or runtime asset edits.
import assert from 'node:assert/strict';
import fs from 'node:fs';
import http from 'node:http';
import path from 'node:path';
import { spawn } from 'node:child_process';
import { once } from 'node:events';

const [binary, directory] = process.argv.slice(2);
assert.ok(binary && directory && path.isAbsolute(binary) && path.isAbsolute(directory),
  'Supply absolute assistant binary and fresh evidence directory paths');
fs.mkdirSync(directory, { recursive: true });
assert.ok(!fs.existsSync(`${directory}/assistant.sqlite`), 'Use a fresh evidence directory');
const children = [], servers = [], contexts = new Map(), pending = new Map();
// Credentials exist only in runner memory for inspection, never in evidence or host state.
const guests = [], held = [];
let holdRating = false;
const report = { assertions: [], requests: [], responses: [], failures: [], trustedInputs: [], screenshots: [], security: [] };
let socket, session, sequence = 0, backend, origin, backendPort, host;
const pause = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
async function wait(predicate, label, ms = 8000) {
  const deadline = Date.now() + ms;
  do {
    const value = await predicate();
    if (value) return value;
    await pause(25);
  } while (Date.now() < deadline);
  throw new Error(`Timeout: ${label}`);
}
function check(name, condition, detail) {
  report.assertions.push({ name, pass: !!condition, ...(detail === undefined ? {} : { detail }) });
  assert.ok(condition, name);
}
function start(command, args, log, env = { PATH: process.env.PATH, HOME: directory }) {
  const fd = fs.openSync(`${directory}/${log}`, 'a');
  const child = spawn(command, args, { cwd: directory, env, stdio: ['ignore', fd, fd] });
  fs.closeSync(fd);
  children.push(child);
  child.on('error', (error) => { report.processError = error.message; });
  return child;
}
async function stop(child) {
  if (child.exitCode !== null || child.signalCode !== null) return;
  const exited = once(child, 'exit');
  child.kill('SIGTERM');
  await Promise.race([exited, pause(2000)]);
  if (child.exitCode === null && child.signalCode === null) {
    child.kill('SIGKILL');
    await exited;
  }
}
async function startBackend() {
  backend = start(binary, ['serve', '--seed-demo'], 'backend.log', {
    PATH: process.env.PATH, HOME: directory, STORE_MODE: 'sqlite', SQLITE_PATH: `${directory}/assistant.sqlite`,
    OPENAI_API_KEY: '', TURSO_DATABASE_URL: '', TURSO_AUTH_TOKEN: '',
    ASSISTANT_ORIGIN: origin, LISTEN_ADDR: `127.0.0.1:${backendPort}`, WIDGET_PARENT_ORIGINS: host,
  });
  await wait(async () => {
    try { return (await fetch(`${origin}/widget/widget.html`)).ok; } catch { return false; }
  }, 'Go startup');
  return backend.pid;
}
async function listen(handler) {
  const server = http.createServer(handler);
  servers.push(server);
  await new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', resolve);
  });
  return server;
}
function cdp(method, params = {}, target = session) {
  return new Promise((resolve, reject) => {
    const id = ++sequence;
    const timer = setTimeout(() => { pending.delete(id); reject(new Error(`CDP timeout: ${method}`)); }, 10000);
    pending.set(id, {
      resolve: (value) => { clearTimeout(timer); resolve(value); },
      reject: (error) => { clearTimeout(timer); reject(error); },
    });
    socket.send(JSON.stringify({ id, method, params, ...(target ? { sessionId: target } : {}) }));
  });
}
async function evaluate(contextId, expression) {
  const result = await cdp('Runtime.evaluate', { expression, contextId, returnByValue: true, awaitPromise: true });
  if (result.exceptionDetails) throw new Error(result.exceptionDetails.text);
  return result.result.value;
}
async function click(context, selector, parentContext) {
  await evaluate(context, `document.querySelector(${JSON.stringify(selector)}).scrollIntoView({block:'center'})`);
  const point = await evaluate(context, `(()=>{const r=document.querySelector(${JSON.stringify(selector)}).getBoundingClientRect();return {x:r.x+r.width/2,y:r.y+r.height/2};})()`);
  if (parentContext) {
    const offset = await evaluate(parentContext, `(()=>{const f=document.querySelector('iframe'),r=f.getBoundingClientRect();return {x:r.x+f.clientLeft,y:r.y+f.clientTop};})()`);
    point.x += offset.x;
    point.y += offset.y;
  }
  await cdp('Input.dispatchMouseEvent', { type: 'mousePressed', ...point, button: 'left', clickCount: 1 });
  await cdp('Input.dispatchMouseEvent', { type: 'mouseReleased', ...point, button: 'left', clickCount: 1 });
}
async function key(key, code, virtualKey, modifiers = 0, text) {
  await cdp('Input.dispatchKeyEvent', { type: 'keyDown', key, code, windowsVirtualKeyCode: virtualKey, modifiers,
    ...(text ? { text, unmodifiedText: text } : {}) });
  await cdp('Input.dispatchKeyEvent', { type: 'keyUp', key, code, windowsVirtualKeyCode: virtualKey, modifiers });
}
async function screenshot(name) {
  const { data } = await cdp('Page.captureScreenshot', { format: 'png' });
  fs.writeFileSync(`${directory}/${name}.png`, Buffer.from(data, 'base64'));
  report.screenshots.push(`${name}.png`);
}
const ratingPosts = () => report.requests.filter((r) => r.method === 'POST' && r.url.endsWith('/rating'));
const ratingResponses = () => report.responses.filter((r) => r.source === 'browser' && r.url.endsWith('/rating') && r.status === 200 && r.body);
async function keyboardSubmit(f, h) {
  // A disabled native button loses focus; navigate back rather than assume focus survives a save.
  await click(f, '#rating-comment', h);
  await key('Tab', 'Tab', 9);
  check('Keyboard submission focuses Save via Tab', await evaluate(f, `document.activeElement.id==='rating-send'`));
  await key('Enter', 'Enter', 13, 0, '\r');
}
async function captureResponse(params) {
  try {
    const { body, base64Encoded } = await cdp('Fetch.getResponseBody', { requestId: params.requestId });
    const value = JSON.parse(base64Encoded ? Buffer.from(body, 'base64').toString() : body);
    const entry = { source: 'browser', url: params.request.url, status: params.responseStatusCode };
    if (entry.url.endsWith('/conversations') && entry.status === 201) {
      guests.push(value);
      entry.body = { conversation_id: value.conversation_id, created_at: value.created_at, capability: '[not retained in evidence]' };
    } else entry.body = value;
    report.responses.push(entry);
  } catch {
    report.bodyCaptureError = 'Could not inspect an API response';
  } finally {
    // Continue the original response without modifying its body, status or headers.
    await cdp('Fetch.continueRequest', { requestId: params.requestId });
  }
}
async function contextFor(expectedOrigin, selector) {
  return wait(async () => {
    for (const [id, context] of contexts) {
      if (!context.auxData?.isDefault) continue;
      try {
        if (await evaluate(id, `location.origin===${JSON.stringify(expectedOrigin)}&&!!document.querySelector(${JSON.stringify(selector)})`)) return id;
      } catch { /* A previous navigation context may have just disappeared. */ }
    }
    return false;
  }, `context ${selector}`);
}
async function openGuest(reload = false) {
  if (reload) await cdp('Page.reload', { ignoreCache: true });
  else await cdp('Page.navigate', { url: host });
  const h = await contextFor(host, '.assistant-launcher');
  await click(h, '.assistant-launcher');
  const f = await contextFor(origin, '#question:not(:disabled)');
  await wait(() => guests.length >= (reload ? 2 : 1), 'observed creation response');
  check('Restricted iframe sandbox retained', await evaluate(h,
    `document.querySelector('iframe').getAttribute('sandbox')==='allow-scripts allow-same-origin'`));
  return { h, f };
}
async function captureInputs(f, scenario) {
  report.trustedInputs.push({ scenario, events: await evaluate(f, '__ratingInputs.splice(0)') });
}
async function saved(f, expectedCount, wording) {
  await wait(() => ratingResponses().length === expectedCount, 'actual Go rating response');
  await wait(() => evaluate(f, `!document.querySelector('#rating-send').disabled&&document.querySelector('#rating-outcome').textContent.includes(${JSON.stringify(wording)})`), 'honest rating outcome');
  check(`${wording}: visible success with no error or review/delivery claim`, await evaluate(f, `(()=>{
    const o=document.querySelector('#rating-outcome'),e=document.querySelector('#rating-error');
    return !o.hidden&&e.hidden&&!/reviewed|delivered|sent/i.test(o.textContent);
  })()`));
  return ratingResponses().at(-1).body;
}
async function privateState(h, f) {
  const credentials = guests.map((guest) => guest.capability);
  for (const [name, context] of [['host', h], ['iframe', f]]) {
    const inspected = await evaluate(context, `({local:localStorage.length,session:sessionStorage.length,
      dom:document.documentElement.outerHTML,url:location.href})`);
    check(`${name}: no local/session storage or capability in DOM/URL`, inspected.local === 0 && inspected.session === 0
      && credentials.every((capability) => !inspected.dom.includes(capability) && !inspected.url.includes(capability)));
  }
  // No credential is injected into the host: messages are inspected in runner memory only.
  const messages = await evaluate(h, '__hostMessages');
  check('Parent messages contain no capability, transcript or rating', messages.every((message) => {
    const text = JSON.stringify(message);
    return !credentials.some((capability) => text.includes(capability))
      && !/Helpful|Revised|conversation_id|rated_at|comment/u.test(text);
  }));
}
async function apiRating(id, capability, body, label) {
  const response = await fetch(`${origin}/api/v1/conversations/${id}/rating`, {
    method: 'POST', headers: { Authorization: `Bearer ${capability}`, 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  const value = await response.json();
  report.responses.push({ source: 'runner HTTP, not UI', scenario: label, url: `${origin}/api/v1/conversations/${id}/rating`, status: response.status, body: value });
  return { status: response.status, body: value };
}

try {
  const reservation = await listen((_req, res) => res.end());
  backendPort = reservation.address().port;
  await new Promise((resolve) => reservation.close(resolve));
  origin = `http://127.0.0.1:${backendPort}`;
  const hostServer = await listen((_req, res) => {
    res.setHeader('Content-Type', 'text/html; charset=utf-8');
    res.end(`<!doctype html><title>Trusted rating regression</title><h1>Local Go / SQLite rating</h1><script src="${origin}/widget/loader.js" data-page-path="/help" data-locale="en" defer></script>`);
  });
  host = `http://127.0.0.1:${hostServer.address().port}`;
  report.initialBackendPID = await startBackend();
  const asset = await fetch(`${origin}/widget/widget.html`);
  const csp = asset.headers.get('content-security-policy');
  check('Actual backend CSP retained', csp?.includes("form-action 'none'") && csp.includes("script-src 'self'")
    && csp.includes("connect-src 'self'") && csp.includes(`frame-ancestors ${host}`), csp);
  start(process.env.CHROMIUM || '/usr/bin/chromium', [
    '--headless', '--disable-gpu', '--no-first-run', '--no-default-browser-check', '--disable-background-networking',
    '--disable-component-update', '--disable-sync', '--proxy-server=http://127.0.0.1:9',
    '--proxy-bypass-list=127.0.0.1;localhost', '--remote-debugging-address=127.0.0.1', '--remote-debugging-port=0',
    `--user-data-dir=${directory}/chromium-profile`, '--window-size=1280,1000', 'about:blank',
  ], 'chromium.log');
  const debuggerPort = await wait(() => {
    try { return fs.readFileSync(`${directory}/chromium-profile/DevToolsActivePort`, 'utf8').split('\n')[0]; } catch { return false; }
  }, 'Chromium startup');
  const browser = await (await fetch(`http://127.0.0.1:${debuggerPort}/json/version`)).json();
  report.browser = browser.Browser;
  socket = new WebSocket(browser.webSocketDebuggerUrl);
  await once(socket, 'open');
  socket.addEventListener('message', (event) => {
    const message = JSON.parse(event.data);
    if (message.id) {
      const callback = pending.get(message.id);
      pending.delete(message.id);
      if (!callback) return;
      if (message.error) callback.reject(new Error(JSON.stringify(message.error)));
      else callback.resolve(message.result);
      return;
    }
    const { method, params } = message;
    if (method === 'Runtime.executionContextCreated') contexts.set(params.context.id, params.context);
    if (method === 'Runtime.executionContextDestroyed') contexts.delete(params.executionContextId);
    if (method === 'Runtime.executionContextsCleared') contexts.clear();
    if (method === 'Fetch.requestPaused') {
      if (params.responseStatusCode !== undefined) {
        void captureResponse(params).catch(() => { report.bodyCaptureError = 'Could not continue an API response'; });
      } else if (holdRating) held.push(params.requestId);
      else void cdp('Fetch.continueRequest', { requestId: params.requestId }).catch(() => { report.bodyCaptureError = 'Could not continue a rating request'; });
    }
    if (method === 'Network.requestWillBeSent' && params.request.url.includes('/api/v1/')) {
      const { url, method, postData } = params.request;
      report.requests.push({ url, method, ...(postData ? { body: JSON.parse(postData) } : {}) });
    }
    if (method === 'Network.loadingFailed') report.failures.push({ error: params.errorText, canceled: !!params.canceled });
    if (method === 'Log.entryAdded' && params.entry.source === 'security') report.security.push(params.entry.text);
    if (method === 'Runtime.exceptionThrown') report.exception = params.exceptionDetails.text;
  });
  const target = await cdp('Target.createTarget', { url: 'about:blank' }, null);
  session = (await cdp('Target.attachToTarget', { targetId: target.targetId, flatten: true }, null)).sessionId;
  for (const domain of ['Page', 'Runtime', 'Network', 'Log']) await cdp(`${domain}.enable`);
  // Response-stage inspection avoids Chromium's canceled-transfer body eviction.
  // No response is fulfilled/replaced; all original Go bytes are continued.
  await cdp('Fetch.enable', { patterns: [
    { urlPattern: `${origin}/api/v1/*`, requestStage: 'Response' },
    { urlPattern: `${origin}/api/v1/conversations/*/rating`, requestStage: 'Request' },
  ] });
  // Observation only: no application state, event handlers, fetches or values are replaced.
  await cdp('Page.addScriptToEvaluateOnNewDocument', { source: `
    window.__ratingInputs=[];window.__hostMessages=[];window.__closeMessages=[];
    for(const type of ['click','keydown','input','change'])addEventListener(type,e=>{
      if(['question','send','chat-close','escalate-email','escalate-note','escalate-consent'].includes(e.target.id)||e.target.id.startsWith('rating-'))
        __ratingInputs.push({type,id:e.target.id,key:e.key,trusted:e.isTrusted});
    },true);
    addEventListener('message',e=>{if(window===parent){__hostMessages.push(e.data);if(e.data?.type==='assistant:close')__closeMessages.push({data:e.data,origin:e.origin,fromFrame:e.source===document.querySelector('iframe')?.contentWindow});}});
  ` });
  let { h, f } = await openGuest();
  const a = guests[0];
  check('Before first turn rating is hidden and disabled', await evaluate(f,
    `document.querySelector('#rating-section').hidden&&document.querySelector('#rating-send').disabled`));
  await click(f, 'details summary', h);
  check('Pre-send sharing and guest-reload disclosures are user-accessible', await evaluate(f, `
    document.querySelector('details').open&&document.querySelector('#transmission').textContent.includes('bounded approved excerpts')
    &&document.querySelector('details').textContent.includes('Reloading loses guest access')`));
  await screenshot('before-turn-disclosure');
  await click(f, 'details summary', h);
  const premature = await apiRating(a.conversation_id, a.capability, { rating: 5 }, 'zero-turn prerequisite');
  check('Backend rejects authorized zero-turn rating', premature.status === 409 && premature.body.error.code === 'rating_requires_turn');
  await click(f, '#question', h);
  await cdp('Input.insertText', { text: 'Can you explain quantum entanglement?' });
  await click(f, '#send', h);
  await wait(() => evaluate(f, `!document.querySelector('#rating-section').hidden&&!document.querySelector('#send').disabled`), 'clarification unlock');
  check('Actual completed clarification unlocks conversation rating', await evaluate(f, `
    document.querySelector('#messages').children.length===2&&document.querySelector('#status').textContent.includes('Clarification received')`));
  check('Exactly one trusted chat send', report.requests.filter((r) => r.method === 'POST' && r.url.endsWith('/messages')).length === 1);
  check('Native radios have accessible group and label associations', await evaluate(f, `(()=>{
    const radios=[...document.querySelectorAll('input[name="rating"]')];
    return radios.length===5&&radios.every((r,i)=>r.type==='radio'&&r.value===String(i+1)&&r.labels.length===1&&r.labels[0].textContent.trim()===String(i+1))
      &&document.querySelector('#rating-section [role="group"]').getAttribute('aria-label')==='Conversation rating, 1 to 5';
  })()`));
  await click(f, '#rating-choice-1', h);
  for (let choice = 1; choice <= 5; choice++) {
    if (choice > 1) await key('ArrowRight', 'ArrowRight', 39);
    check(`Native radio keyboard navigation selects/focuses ${choice}`, await evaluate(f, `
      document.activeElement.id==='rating-choice-${choice}'&&document.querySelector('#rating-choice-${choice}').checked
      &&document.querySelectorAll('input[name="rating"]:checked').length===1`));
  }
  await key('ArrowLeft', 'ArrowLeft', 37);
  check('Native radio ArrowLeft selects previous choice', await evaluate(f, `document.activeElement.id==='rating-choice-4'&&document.activeElement.checked`));
  await key('ArrowRight', 'ArrowRight', 39);
  await key('Tab', 'Tab', 9);
  check('Tab leaves radio group for labelled comment', await evaluate(f, `document.activeElement.id==='rating-comment'&&document.activeElement.labels[0].textContent==='Optional comment'`));
  await cdp('Input.insertText', { text: '  Helpful — local browser  ' });
  await key('Tab', 'Tab', 9);
  check('Tab reaches Save rating', await evaluate(f, `document.activeElement.id==='rating-send'`));
  holdRating = true;
  const before = ratingPosts().length;
  await click(f, '#rating-send', h);
  await wait(() => held.length === 1, 'real request held before backend response');
  check('In-flight actual request disables submission and announces saving', await evaluate(f, `
    document.querySelector('#rating-send').disabled&&document.querySelector('#rating-section').getAttribute('aria-busy')==='true'
    &&document.querySelector('#rating-outcome').textContent==='Saving your rating…'`));
  await click(f, '#rating-send', h);
  await key('Enter', 'Enter', 13, 0, '\r');
  await pause(150);
  check('Repeated trusted activation while busy emits exactly one request', held.length === 1 && ratingPosts().length === before + 1);
  await screenshot('saving-duplicate-blocked');
  holdRating = false;
  await cdp('Fetch.continueRequest', { requestId: held.shift() });
  const first = await saved(f, 1, 'recorded');
  check('First saved response matches canonical trusted input', first.rating === 5 && first.comment === 'Helpful — local browser' && !!first.rated_at);
  await captureInputs(f, 'clarification, native 1–5 navigation, comment and duplicate exclusion');
  await screenshot('first-rating-recorded');

  await click(f, '#rating-choice-2', h);
  check('Trusted pointer selects revision score', await evaluate(f, `document.querySelector('#rating-choice-2').checked`));
  await key('Tab', 'Tab', 9);
  check('Revision Tab reaches comment', await evaluate(f, `document.activeElement.id==='rating-comment'`));
  await key('a', 'KeyA', 65, 2);
  await cdp('Input.insertText', { text: 'Revised by trusted input' });
  await key('Tab', 'Tab', 9);
  await key('Enter', 'Enter', 13, 0, '\r');
  const revised = await saved(f, 2, 'updated');
  check('Revision replaces both fields and changes stored timestamp', revised.rating === 2 && revised.comment === 'Revised by trusted input' && revised.rated_at !== first.rated_at);
  // Combined-tree regression: close a rated conversation with unsaved rating,
  // question and support drafts. Observe only; drive every edit/close via CDP input.
  await click(f, '#rating-choice-3', h);
  await click(f, '#rating-comment', h);
  await key('a', 'KeyA', 65, 2);
  await cdp('Input.insertText', { text: 'Unsent rating draft' });
  await click(f, '#question', h);
  await cdp('Input.insertText', { text: 'Unsent question draft' });
  await click(f, '#escalate-toggle', h);
  await click(f, '#handoff-email', h);
  check('Approved support labels retain available-email and unavailable-WhatsApp semantics', await evaluate(f, `(()=>{
    const e=document.querySelector('#handoff-email'),w=document.querySelector('#handoff-whatsapp');
    return e.textContent==='Request via email'&&!e.disabled&&e.getAttribute('aria-pressed')==='true'
      &&w.textContent==='Request via WhatsApp'&&w.disabled&&w.getAttribute('aria-pressed')==='false'
      &&document.querySelector('#whatsapp-reason').textContent.includes('not enabled WhatsApp');
  })()`));
  await click(f, '#escalate-email', h);
  await cdp('Input.insertText', { text: 'draft@example.org' });
  await click(f, '#escalate-note', h);
  await cdp('Input.insertText', { text: 'Unsent support note' });
  await click(f, '#escalate-consent', h);
  const draftState = `(()=>{const q=s=>document.querySelector(s);return {
    transcript:q('#messages').textContent,question:q('#question').value,comment:q('#rating-comment').value,
    score:q('input[name="rating"]:checked').value,outcome:q('#rating-outcome').textContent,
    email:q('#escalate-email').value,note:q('#escalate-note').value,consent:q('#escalate-consent').checked};})()`;
  const beforeCloseDrafts = await evaluate(f, draftState);
  const beforeClosePosts = report.requests.filter((r) => r.method === 'POST').length;
  const beforeCloseGuests = guests.length;
  for (const action of ['pointer close', 'email-form Escape']) {
    const closes = await evaluate(h, '__closeMessages.length');
    if (action === 'pointer close') await click(f, '#chat-close', h);
    else {
      await click(f, '#escalate-email', h);
      await key('Escape', 'Escape', 27);
    }
    await wait(() => evaluate(h, `document.querySelector('iframe').hidden&&document.activeElement===document.querySelector('.assistant-launcher')`), `${action} returns launcher focus`);
    check(`${action}: exact initialized close protocol from the mounted iframe`, await evaluate(h, `(()=>{
      const m=__closeMessages.at(-1);return __closeMessages.length===${closes + 1}
        &&JSON.stringify(m.data)==='{"type":"assistant:close","version":1}'&&m.origin===${JSON.stringify(origin)}&&m.fromFrame;
    })()`));
    await click(h, '.assistant-launcher');
    check(`${action}: same rated iframe preserves transcript and all unsent drafts`,
      JSON.stringify(await evaluate(f, draftState)) === JSON.stringify(beforeCloseDrafts) && guests.length === beforeCloseGuests);
    check(`${action}: disclosures reset but rating stays eligible and editable`, await evaluate(f, `
      document.querySelector('#handoff-menu').hidden&&document.querySelector('#escalation-form').hidden
      &&!document.querySelector('#rating-section').hidden&&!document.querySelector('#rating-send').disabled`));
    check(`${action}: no creation, message, rating or escalation POST`, report.requests.filter((r) => r.method === 'POST').length === beforeClosePosts);
    await click(f, '#escalate-toggle', h);
    await click(f, '#handoff-email', h);
  }
  await screenshot('rated-reopened-drafts');
  // Restore the saved feedback with trusted controls; the unchanged replay below
  // independently confirms neither close nor draft edits changed the durable row.
  await click(f, '#rating-choice-2', h);
  await click(f, '#rating-comment', h);
  await key('a', 'KeyA', 65, 2);
  await cdp('Input.insertText', { text: 'Revised by trusted input' });
  await keyboardSubmit(f, h);
  const replay = await saved(f, 3, 'recorded');
  check('Trusted identical replay preserves complete stored envelope', JSON.stringify(replay) === JSON.stringify(revised));
  check('Save/revise/replay each issued exactly one browser POST', ratingPosts().length === 3);
  await captureInputs(f, 'trusted pointer revision, keyboard save and identical replay');
  await privateState(h, f);
  await screenshot('revised-and-replayed');

  await stop(backend);
  check('Original Go OS process exited before restart', backend.exitCode !== null || backend.signalCode !== null);
  await keyboardSubmit(f, h);
  await wait(() => evaluate(f, `!document.querySelector('#rating-error').hidden&&document.querySelector('#rating-error').textContent.includes('not confirmed')`), 'truthful stopped-backend outcome');
  check('Unavailable backend never shows successful recording', await evaluate(f, `document.querySelector('#rating-outcome').hidden&&!document.querySelector('#rating-send').disabled`));
  await screenshot('backend-down-unconfirmed');
  report.restartedBackendPID = await startBackend();
  check('New Go OS process uses same SQLite file', report.initialBackendPID !== report.restartedBackendPID && fs.existsSync(`${directory}/assistant.sqlite`));
  await keyboardSubmit(f, h);
  const restartedReplay = await saved(f, 4, 'recorded');
  check('Process restart retains complete rating and identical timestamp', JSON.stringify(restartedReplay) === JSON.stringify(revised));
  check('Restart/failure/retry count: five attempts, four successful UI responses', ratingPosts().length === 5 && ratingResponses().length === 4);
  const history = await fetch(`${origin}/api/v1/conversations/${a.conversation_id}/messages?limit=20`, { headers: { Authorization: `Bearer ${a.capability}` } });
  const storedHistory = await history.json();
  check('Completed clarification persists across process restart', history.status === 200 && storedHistory.messages.length === 2 && storedHistory.messages[1].status === 'needs_clarification');
  await captureInputs(f, 'backend OS stop, truthful failure, OS restart and trusted retry');
  await screenshot('process-restart-replay');

  ({ h, f } = await openGuest(true));
  const b = guests[1];
  check('Browser reload loses prior guest access, creates distinct empty conversation', a.conversation_id !== b.conversation_id && a.capability !== b.capability && await evaluate(f, `
    document.querySelector('#rating-section').hidden&&document.querySelector('#messages').children.length===0&&document.querySelector('#rating-outcome').hidden`));
  const wrongA = await apiRating(a.conversation_id, b.capability, { rating: 1, comment: 'wrong guest' }, 'B cannot revise A');
  const wrongB = await apiRating(b.conversation_id, a.capability, { rating: 1, comment: 'wrong guest' }, 'A cannot revise B');
  check('Cross-guest rating authorization fails uniformly both directions', wrongA.status === 401 && wrongB.status === 401 && JSON.stringify(wrongA.body) === JSON.stringify(wrongB.body));
  const emptyB = await apiRating(b.conversation_id, b.capability, { rating: 1 }, 'new guest still requires own turn');
  check('Reloaded guest cannot inherit old turn eligibility', emptyB.status === 409 && emptyB.body.error.code === 'rating_requires_turn');
  const unchanged = await apiRating(a.conversation_id, a.capability, { rating: 2, comment: 'Revised by trusted input' }, 'runner-only prior credential confirms no unauthorized mutation');
  check('Isolation attempts leave original complete persisted row unchanged', unchanged.status === 200 && JSON.stringify(unchanged.body) === JSON.stringify(revised));
  await privateState(h, f);
  await screenshot('reload-new-guest');
  const events = report.trustedInputs.flatMap((input) => input.events);
  check('All captured control events are browser-trusted', events.length > 0 && events.every((event) => event.trusted));
  check('Trusted rating input includes native radio changes, comment input and pointer/keyboard submission',
    events.some((e) => e.type === 'change' && e.id === 'rating-choice-5')
    && events.some((e) => e.type === 'input' && e.id === 'rating-comment')
    && events.some((e) => e.type === 'click' && e.id === 'rating-send')
    && events.some((e) => e.type === 'keydown' && e.id === 'rating-send' && e.key === 'Enter'));
  check('No browser exceptions, response inspection failures or CSP security errors', !report.exception && !report.bodyCaptureError && !report.processError && report.security.length === 0);
  check('Backend logs contain no raw guest capability', guests.every((g) => !fs.readFileSync(`${directory}/backend.log`, 'utf8').includes(g.capability)));
} catch (error) {
  if (socket && session) {
    for (const [id, context] of contexts) {
      if (!context.auxData?.isDefault) continue;
      try {
        const state = await evaluate(id, `(()=>{const q=s=>document.querySelector(s);return q('#rating-section')?{
          outcome:q('#rating-outcome').textContent,error:q('#rating-error').textContent,
          disabled:q('#rating-send').disabled,focus:document.activeElement.id}:null})()`);
        if (state) report.failureState = state;
      } catch { /* A destroyed context is not failure evidence. */ }
    }
    try { await screenshot('failure'); } catch { /* Keep the original failure. */ }
  }
  report.fatal = String(error.stack || error);
  console.error(report.fatal);
  process.exitCode = 1;
} finally {
  socket?.close();
  for (const child of children.reverse()) await stop(child);
  for (const server of servers) if (server.listening) await new Promise((resolve) => server.close(resolve));
  report.summary = { passed: report.assertions.filter((entry) => entry.pass).length, failed: report.assertions.filter((entry) => !entry.pass).length };
  fs.writeFileSync(`${directory}/results.json`, `${JSON.stringify(report, null, 2)}\n`);
  console.log(JSON.stringify(report.summary));
  if (report.summary.failed || report.fatal) process.exitCode = 1;
}
