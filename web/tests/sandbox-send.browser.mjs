// Separate opt-in browser regression, never launched by the Node unit glob.
// Usage: node web/tests/sandbox-send.browser.mjs /absolute/assistant /absolute/evidence-dir
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
const report = { assertions: [], requests: [], responses: [], security: [], trustedInputs: [], screenshots: [] };
let socket, session, sequence = 0;
const pause = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
async function wait(predicate, label, ms = 5000) {
  const deadline = Date.now() + ms;
  do {
    const value = await predicate();
    if (value) return value;
    await pause(25);
  } while (Date.now() < deadline);
  throw new Error(`Timeout: ${label}`);
}
function check(name, condition, detail) {
  report.assertions.push({ name, pass: !!condition, detail });
  if (!condition) console.error(`FAIL: ${name}`, detail ?? '');
}
function child(command, args, log, env = { PATH: process.env.PATH, HOME: directory }) {
  const fd = fs.openSync(`${directory}/${log}`, 'a');
  const process = spawn(command, args, { cwd: directory, env, stdio: ['ignore', fd, fd] });
  fs.closeSync(fd);
  children.push(process);
  return process;
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
function cdp(method, params = {}, targetSession = session) {
  return new Promise((resolve, reject) => {
    const id = ++sequence;
    pending.set(id, { resolve, reject });
    socket.send(JSON.stringify({ id, method, params, ...(targetSession ? { sessionId: targetSession } : {}) }));
  });
}
async function evaluate(contextId, expression) {
  const result = await cdp('Runtime.evaluate', { expression, contextId, returnByValue: true, awaitPromise: true });
  if (result.exceptionDetails) throw new Error(result.exceptionDetails.text);
  return result.result.value;
}
const state = (context) => evaluate(context, `({status:document.querySelector('#status').textContent,
  error:document.querySelector('#error').textContent, count:document.querySelector('#messages').children.length,
  text:document.querySelector('#messages').textContent, disabled:document.querySelector('#send').disabled})`);
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
  fs.writeFileSync(`${directory}/${name}`, Buffer.from(data, 'base64'));
  report.screenshots.push(name);
}
const posts = () => report.requests.filter((request) => request.method === 'POST' && request.url.endsWith('/messages'));

try {
  const reservation = await listen((_request, response) => response.end());
  const port = reservation.address().port;
  await new Promise((resolve) => reservation.close(resolve));
  const origin = `http://127.0.0.1:${port}`;
  const hostServer = await listen((_request, response) => {
    response.setHeader('Content-Type', 'text/html');
    response.end(`<!doctype html><title>Sandbox send regression</title><h1>Local Go / SQLite, no AI key</h1><script src="${origin}/widget/loader.js" data-page-path="/help" data-locale="en" defer></script>`);
  });
  const host = `http://127.0.0.1:${hostServer.address().port}`;
  child(binary, ['serve', '--seed-demo'], 'backend.log', {
    PATH: process.env.PATH, HOME: directory, STORE_MODE: 'sqlite', SQLITE_PATH: `${directory}/assistant.sqlite`,
    OPENAI_API_KEY: '', TURSO_DATABASE_URL: '', TURSO_AUTH_TOKEN: '',
    ASSISTANT_ORIGIN: origin, LISTEN_ADDR: `127.0.0.1:${port}`, WIDGET_PARENT_ORIGINS: host,
  });
  await wait(async () => { try { return (await fetch(`${origin}/widget/widget.html`)).ok; } catch { return false; } }, 'Go startup');
  const csp = (await fetch(`${origin}/widget/widget.html`)).headers.get('content-security-policy');
  check('Restricted CSP is present', csp.includes("form-action 'none'") && csp.includes("script-src 'self'") && csp.includes(`frame-ancestors ${host}`), csp);
  child(process.env.CHROMIUM || '/usr/bin/chromium', [
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
      const result = pending.get(message.id);
      pending.delete(message.id);
      if (message.error) result.reject(new Error(JSON.stringify(message.error)));
      else result.resolve(message.result);
      return;
    }
    const { method, params } = message;
    if (method === 'Runtime.executionContextCreated') contexts.set(params.context.id, params.context);
    if (method === 'Runtime.executionContextDestroyed') contexts.delete(params.executionContextId);
    if (method === 'Runtime.executionContextsCleared') contexts.clear();
    if (method === 'Network.requestWillBeSent' && params.request.url.includes('/api/v1')) {
      const { url, method, postData } = params.request;
      // Deliberately never save headers/capabilities or response bodies.
      report.requests.push({ url, method, ...(postData ? { body: JSON.parse(postData) } : {}) });
    }
    if (method === 'Network.responseReceived' && params.response.url.includes('/api/v1')) {
      report.responses.push({ url: params.response.url, status: params.response.status });
    }
    if (method === 'Log.entryAdded' && params.entry.source === 'security') report.security.push(params.entry.text);
    if (method === 'Runtime.exceptionThrown') report.exception = params.exceptionDetails.text;
  });
  const target = await cdp('Target.createTarget', { url: 'about:blank' }, null);
  session = (await cdp('Target.attachToTarget', { targetId: target.targetId, flatten: true }, null)).sessionId;
  for (const domain of ['Page', 'Runtime', 'Network', 'Log']) await cdp(`${domain}.enable`);
  await cdp('Page.addScriptToEvaluateOnNewDocument', { source: `window.__inputs=[];window.__closeMessages=[];addEventListener('message',e=>{if(e.data?.type==='assistant:close')__closeMessages.push({data:e.data,origin:e.origin,fromFrame:e.source===document.querySelector('iframe')?.contentWindow});});for(const type of ['click','keydown'])addEventListener(type,e=>{if(['send','question','retry','chat-close'].includes(e.target.id))__inputs.push({type,id:e.target.id,key:e.key,trusted:e.isTrusted});});` });

  async function open() {
    await cdp('Page.navigate', { url: host });
    const h = await wait(async () => {
      for (const [id, context] of contexts) {
        if (!context.auxData?.isDefault) continue;
        try { if (await evaluate(id, `location.origin===${JSON.stringify(host)}&&!!document.querySelector('.assistant-launcher')`)) return id; } catch { /* Old context. */ }
      }
      return false;
    }, 'host launcher');
    await click(h, '.assistant-launcher');
    const f = await wait(async () => {
      for (const [id, context] of contexts) {
        if (!context.auxData?.isDefault) continue;
        try { if (await evaluate(id, `location.origin===${JSON.stringify(origin)}&&!!document.querySelector('#send')&&!document.querySelector('#send').disabled`)) return id; } catch { /* Loading context. */ }
      }
      return false;
    }, 'ready iframe');
    check('Exact restricted sandbox retained', await evaluate(h, `document.querySelector('iframe').getAttribute('sandbox')==='allow-scripts allow-same-origin'`));
    return { h, f };
  }
  for (const action of ['mouse', 'button-enter', 'button-space', 'ctrl-enter', 'meta-enter']) {
    const { h, f } = await open();
    await click(f, '#question', h);
    const question = action === 'mouse' ? 'How do I start the demo?' : 'Can you explain quantum entanglement?';
    await cdp('Input.insertText', { text: question });
    if (action === 'ctrl-enter') {
      const before = posts().length;
      await key('Enter', 'Enter', 13, 0, '\r');
      await pause(100);
      check('Plain textarea Enter inserts newline without sending', posts().length === before && await evaluate(f, `document.querySelector('#question').value.endsWith('\\n')`));
    }
    const before = posts().length;
    if (action === 'mouse') await click(f, '#send', h);
    else if (action.startsWith('button-')) {
      await key('Tab', 'Tab', 9);
      check(`${action}: Tab reaches Send`, await evaluate(f, `document.activeElement.id==='send'`));
      if (action === 'button-enter') await key('Enter', 'Enter', 13, 0, '\r');
      else await key(' ', 'Space', 32, 0, ' ');
    } else await key('Enter', 'Enter', 13, action === 'ctrl-enter' ? 2 : 4);
    await pause(400);
    const sent = posts().length - before;
    check(`${action}: exactly one POST from real input`, sent === 1, { sent, state: await state(f) });
    if (sent === 1) {
      await wait(async () => !(await state(f)).status.includes('Sending') && !(await state(f)).status.includes('Checking'), `${action} outcome`);
      const outcome = await state(f);
      if (action === 'mouse') {
        check('Empty key gives 503 + visible unavailable + history, no automatic resend',
          report.responses.some((response) => response.url.endsWith('/messages') && response.status === 503)
          && outcome.error.includes('AI service is unavailable') && outcome.count === 1
          && report.requests.at(-1).method === 'GET' && posts().length === before + 1, outcome);
        await screenshot(`${action}.png`);
        const original = posts().at(-1).body;
        await click(f, '#retry', h);
        await wait(async () => posts().length === before + 2 && (await state(f)).status.startsWith('No saved reply'), 'same-ID explicit retry');
        check('Real retry click preserves ID/text and reconciles history', JSON.stringify(posts().at(-1).body) === JSON.stringify(original) && report.requests.at(-1).method === 'GET');
      } else check(`${action}: persisted clarification pair`, outcome.count === 2 && outcome.status.startsWith('Clarification received') && outcome.text.includes('Clarification · limited knowledge'), outcome);
    }
    report.trustedInputs.push({ action, events: await evaluate(f, '__inputs') });
    await screenshot(`${action}${sent ? '-complete' : '-blocked'}.png`);
  }
  // Real keyboard events inside the iframe must close the host window in one
  // press, even when focus is in the expanded email form. Hiding must not
  // reload the iframe or discard the conversation and unsent draft.
  const { h, f } = await open();
  const priorPosts = posts().length;
  const creations = () => report.requests.filter((request) => request.method === 'POST' && request.url.endsWith('/conversations')).length;
  const initialCreations = creations();
  await click(f, '#question', h);
  await cdp('Input.insertText', { text: 'Can you explain quantum entanglement?' });
  await click(f, '#send', h);
  await wait(() => evaluate(f, `document.querySelector('#messages').children.length===2&&!document.querySelector('#send').disabled`), 'completed turn before close');
  const transcript = await evaluate(f, `document.querySelector('#messages').textContent`);
  await click(f, '#question', h);
  await cdp('Input.insertText', { text: 'Unsent draft stays here' });
  await key('q', 'KeyQ', 81, 0, 'q');
  check('Other iframe key does not close window', await evaluate(h, `!document.querySelector('iframe').hidden`));
  await key('Escape', 'Escape', 27);
  await wait(() => evaluate(h, `document.querySelector('iframe').hidden`), 'iframe Escape closes window');
  check('Iframe Escape returns focus and closed launcher state', await evaluate(h, `document.activeElement===document.querySelector('.assistant-launcher')&&document.querySelector('.assistant-launcher').getAttribute('aria-expanded')==='false'&&document.querySelector('.assistant-launcher').getAttribute('aria-label')==='Open support assistant'`));
  await click(h, '.assistant-launcher');
  check('Reopen preserves iframe and unsent draft without new creation', await evaluate(f, `document.querySelector('#question').value.startsWith('Unsent draft stays here')`) && creations() === initialCreations);
  await click(f, '#escalate-toggle', h);
  await click(f, '#handoff-email', h);
  await click(f, '#escalate-email', h);
  await cdp('Input.insertText', { text: 'visitor@example.org' });
  check('Email form open before Escape', await evaluate(f, `!document.querySelector('#handoff-menu').hidden&&!document.querySelector('#escalation-form').hidden`));
  await key('Escape', 'Escape', 27);
  await wait(() => evaluate(h, `document.querySelector('iframe').hidden`), 'email form Escape closes window');
  check('Email form Escape focuses launcher', await evaluate(h, `document.activeElement===document.querySelector('.assistant-launcher')`));
  await click(h, '.assistant-launcher');
  check('Email draft and conversation survive reopening', await evaluate(f, `document.querySelector('#escalate-email').value==='visitor@example.org'&&document.querySelector('#question').value.startsWith('Unsent draft stays here')`) && creations() === initialCreations);
  await click(f, '#escalate-toggle', h);
  await click(f, '#handoff-email', h);
  await click(f, '#escalate-note', h);
  await cdp('Input.insertText', { text: 'Unsent support note' });
  await click(f, '#escalate-consent', h);
  for (const action of ['mouse', 'enter', 'space']) {
    const beforeClose = await evaluate(h, '__closeMessages.length');
    if (action === 'mouse') await click(f, '#chat-close', h);
    else {
      // Reach the header button by actual keyboard traversal, not DOM focus/click.
      await click(f, '#question', h);
      for (let n = 0; n < 20 && !await evaluate(f, `document.activeElement.id==='chat-close'`); n++) await key('Tab', 'Tab', 9, 8);
      check(`${action}: keyboard reaches Close chat`, await evaluate(f, `document.activeElement.id==='chat-close'`));
      check(`${action}: visible unobscured focus ring`, await evaluate(f, `(()=>{const b=document.querySelector('#chat-close'),s=getComputedStyle(b),r=b.getBoundingClientRect();return b.matches(':focus-visible')&&s.outlineStyle==='solid'&&parseFloat(s.outlineWidth)>=3&&r.top>=6&&r.right+6<=innerWidth})()`));
      await pause(180);
      await screenshot(`close-${action}-focus.png`);
      await key(action === 'enter' ? 'Enter' : ' ', action === 'enter' ? 'Enter' : 'Space', action === 'enter' ? 13 : 32, 0, action === 'enter' ? '\r' : ' ');
    }
    await wait(() => evaluate(h, `document.querySelector('iframe').hidden`), `${action} closes widget`);
    check(`${action}: one exact close message from initialized iframe`, await evaluate(h, `(()=>{const m=__closeMessages.at(-1);return __closeMessages.length===${beforeClose + 1}&&JSON.stringify(m.data)==='{"type":"assistant:close","version":1}'&&m.origin===${JSON.stringify(origin)}&&m.fromFrame})()`));
    check(`${action}: host restores launcher focus`, await evaluate(h, `document.activeElement===document.querySelector('.assistant-launcher')&&document.querySelector('.assistant-launcher').getAttribute('aria-expanded')==='false'`));
    await click(h, '.assistant-launcher');
    check(`${action}: same iframe preserves completed chat and question draft`, await evaluate(f, `document.querySelector('#messages').textContent===${JSON.stringify(transcript)}&&document.querySelector('#question').value.startsWith('Unsent draft stays here')`) && creations() === initialCreations);
    check(`${action}: close collapses disclosures without discarding support drafts`, await evaluate(f, `document.querySelector('#handoff-menu').hidden&&document.querySelector('#escalation-form').hidden&&document.querySelector('#escalate-email').value==='visitor@example.org'&&document.querySelector('#escalate-note').value==='Unsent support note'&&document.querySelector('#escalate-consent').checked`));
    await click(f, '#escalate-toggle', h);
    await click(f, '#handoff-email', h);
  }
  const closeInputs = await evaluate(f, `__inputs.filter(e=>e.id==='chat-close')`);
  report.trustedInputs.push({ action: 'header-close-pointer-enter-space', events: closeInputs });
  check('Close button received three browser-trusted clicks', closeInputs.filter((event) => event.type === 'click' && event.trusted).length === 3);
  await click(h, '.assistant-launcher');
  // Launcher has focus after the real pointer click; reopen then Escape there.
  await click(h, '.assistant-launcher');
  await key('Escape', 'Escape', 27);
  check('Trusted host launcher Escape closes and keeps focus', await evaluate(h, `document.querySelector('iframe').hidden&&document.activeElement===document.querySelector('.assistant-launcher')&&document.querySelector('.assistant-launcher').getAttribute('aria-expanded')==='false'`));
  check('Close and Escape do not submit questions or escalation', posts().length === priorPosts + 1 && !report.requests.some((request) => request.url.endsWith('/escalation')));
  check('All observed control inputs are browser-trusted', report.trustedInputs.every((input) => input.events.length && input.events.every((event) => event.trusted)));
  check('No browser JavaScript exceptions', !report.exception);
} catch (error) {
  report.fatal = String(error.stack || error);
  console.error(report.fatal);
  process.exitCode = 1;
} finally {
  if (socket) socket.close();
  for (const process of children.reverse()) {
    if (process.exitCode !== null) continue;
    const exited = once(process, 'exit');
    process.kill('SIGTERM');
    await Promise.race([exited, pause(2000)]);
    if (process.exitCode === null) { process.kill('SIGKILL'); await exited; }
  }
  for (const server of servers) if (server.listening) await new Promise((resolve) => server.close(resolve));
  report.summary = { passed: report.assertions.filter((entry) => entry.pass).length, failed: report.assertions.filter((entry) => !entry.pass).length };
  fs.writeFileSync(`${directory}/results.json`, `${JSON.stringify(report, null, 2)}\n`);
  console.log(JSON.stringify(report.summary));
  if (report.summary.failed) process.exitCode = 1;
}
