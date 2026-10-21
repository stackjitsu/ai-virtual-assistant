// Opt-in local Chromium interaction test: node web/tests/rating.browser.mjs
// Serves only local fixtures; no backend, provider or credentials are used.
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { mkdtemp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import http from 'node:http';
import { spawn } from 'node:child_process';
import { once } from 'node:events';

const directory = await mkdtemp(join(tmpdir(), 'rating-browser-'));
const servers = [];
let browser, socket;
const requests = [];
const responses = [];
const pending = new Map();
let sequence = 0;
let session;
const contexts = new Map();
const wait = async (fn, label) => {
  for (let i = 0; i < 200; i++) {
    const value = await fn();
    if (value) return value;
    await new Promise((resolve) => setTimeout(resolve, 25));
  }
  throw new Error(`Timed out: ${label}`);
};
async function listen(handler) {
  const server = http.createServer(handler);
  servers.push(server);
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  return `http://127.0.0.1:${server.address().port}`;
}
function cdp(method, params = {}, target = session) {
  return new Promise((resolve, reject) => {
    const id = ++sequence;
    pending.set(id, { resolve, reject });
    socket.send(JSON.stringify({ id, method, params, ...(target ? { sessionId: target } : {}) }));
  });
}
async function evaluate(expression, contextId) {
  const reply = await cdp('Runtime.evaluate', { expression, ...(contextId ? { contextId } : {}), returnByValue: true, awaitPromise: true });
  if (reply.exceptionDetails) throw new Error(reply.exceptionDetails.text);
  return reply.result.value;
}
async function key(key, code, virtualKey) {
  await cdp('Input.dispatchKeyEvent', { type: 'keyDown', key, code, windowsVirtualKeyCode: virtualKey, text: '\r', unmodifiedText: '\r' });
  await cdp('Input.dispatchKeyEvent', { type: 'keyUp', key, code, windowsVirtualKeyCode: virtualKey });
}
try {
  let assistant;
  assistant = await listen(async (req, res) => {
    if (req.url.startsWith('/api/v1/')) {
      let body = '';
      for await (const chunk of req) body += chunk;
      requests.push({ path: req.url, body: body ? JSON.parse(body) : null, bearer: req.headers.authorization });
      res.setHeader('Content-Type', 'application/json');
      res.setHeader('Cache-Control', 'no-store');
      const path = req.url;
      let value;
      if (path === '/api/v1/conversations') {
        res.statusCode = 201;
        value = { conversation_id: 'a'.repeat(32), capability: 'b'.repeat(64), created_at: '2026-10-20T12:00:00Z' };
      } else if (path.endsWith('/messages?limit=20')) value = { messages: [], next_before: null };
      else if (path.endsWith('/messages')) {
        const turn = requests.filter((request) => request.path.endsWith('/messages')).length;
        const request_id = JSON.parse(body).request_id;
        value = { request_id, messages: [
          { seq: turn * 2 - 1, request_id, role: 'user', text: JSON.parse(body).text, status: 'submitted', sources: [], created_at: '2026-10-20T12:00:01Z' },
          { seq: turn * 2, request_id, role: 'assistant', text: 'Please clarify your question.', status: 'needs_clarification', sources: [], created_at: '2026-10-20T12:00:02Z' },
        ] };
      } else if (path.endsWith('/rating')) {
        const rating = JSON.parse(body);
        value = { ...rating, rated_at: `2026-10-20T12:00:0${requests.filter((request) => request.path.endsWith('/rating')).length}Z` };
      } else { res.statusCode = 404; value = { error: { code: 'not_found', message: 'Not found.' } }; }
      responses.push({ path, status: res.statusCode });
      res.end(JSON.stringify(value));
      return;
    }
    const asset = req.url?.replace(/^\/widget\//u, '');
    if (!['loader.js', 'loader.css', 'widget.html', 'widget.css', 'widget.js', 'widget-core.mjs', 'protocol.mjs'].includes(asset)) { res.statusCode = 404; res.end(); return; }
    res.setHeader('Content-Type', asset.endsWith('.html') ? 'text/html' : asset.endsWith('.css') ? 'text/css' : 'text/javascript');
    if (asset === 'widget.html') res.setHeader('Content-Security-Policy', `default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors ${host}`);
    res.end(await readFile(new URL(`../assets/${asset}`, import.meta.url)));
  });
  const host = await listen((_req, res) => {
    res.setHeader('Content-Type', 'text/html');
    res.end(`<!doctype html><title>Local rating fixture</title><script src="${assistant}/widget/loader.js" data-assistant-origin="${assistant}" defer></script>`);
  });
  browser = spawn(process.env.CHROMIUM || '/usr/bin/chromium', [
    '--headless', '--no-sandbox', '--disable-gpu', '--no-first-run', '--disable-background-networking',
    '--proxy-server=http://127.0.0.1:9', '--proxy-bypass-list=127.0.0.1;localhost',
    '--remote-debugging-address=127.0.0.1', '--remote-debugging-port=0', `--user-data-dir=${directory}`, 'about:blank',
  ], { stdio: 'ignore' });
  const port = await wait(async () => {
    try { return (await readFile(join(directory, 'DevToolsActivePort'), 'utf8')).split('\n')[0]; } catch { return null; }
  }, 'Chromium startup');
  const version = await (await fetch(`http://127.0.0.1:${port}/json/version`)).json();
  socket = new WebSocket(version.webSocketDebuggerUrl);
  await once(socket, 'open');
  socket.addEventListener('message', (event) => {
    const message = JSON.parse(event.data);
    if (message.method === 'Runtime.executionContextCreated') contexts.set(message.params.context.id, message.params.context);
    if (message.method === 'Runtime.executionContextDestroyed') contexts.delete(message.params.executionContextId);
    if (message.id) {
      const callback = pending.get(message.id);
      pending.delete(message.id);
      if (message.error) callback.reject(new Error(JSON.stringify(message.error)));
      else callback.resolve(message.result);
    }
  });
  const target = await cdp('Target.createTarget', { url: host }, null);
  session = (await cdp('Target.attachToTarget', { targetId: target.targetId, flatten: true }, null)).sessionId;
  await cdp('Runtime.enable');
  const contextFor = (origin) => [...contexts].find(([, context]) => context.auxData?.isDefault && context.origin === origin)?.[0];
  const hostContext = await wait(() => contextFor(host), 'host context');
  await wait(() => evaluate("!!document.querySelector('.assistant-launcher')", hostContext), 'launcher');
  assert.equal(await evaluate("document.querySelector('.assistant-launcher').getAttribute('aria-expanded')", hostContext), 'false');
  await evaluate("document.querySelector('.assistant-launcher').click()", hostContext);
  const frameContext = await wait(() => contextFor(assistant), 'iframe context');
  await wait(() => evaluate("!!document.querySelector('#question:not(:disabled)')", frameContext), 'iframe ready');
  const inFrame = (expression) => evaluate(`(()=>{const d=document;return (${expression});})()`, frameContext);
  assert.equal(await evaluate("document.querySelector('iframe').getAttribute('sandbox')", hostContext), 'allow-scripts allow-same-origin');
  assert.equal(await inFrame("d.querySelector('#rating-section').hidden"), true);
  assert.equal(await inFrame("d.querySelector('#transmission').textContent.includes('bounded approved excerpts may go to the configured AI provider')"), true);
  assert.equal(await inFrame("d.querySelector('details summary').textContent"), 'More information');
  await inFrame("(d.querySelector('#question').value='Please help',d.querySelector('#send').click())");
  await wait(() => inFrame("!d.querySelector('#rating-section').hidden"), 'completed clarification');
  assert.equal(await inFrame("d.querySelector('#messages').textContent.includes('Clarification')"), true);
  assert.equal(await inFrame("d.querySelector('#message-form').getBoundingClientRect().bottom <= d.documentElement.clientHeight"), true);
  await inFrame("(d.querySelector('#rating-choice-5').checked=true,d.querySelector('#rating-comment').value='  Helpful  ',d.querySelector('#rating-send').click())");
  await wait(() => inFrame("d.querySelector('#rating-outcome').textContent.includes('recorded')"), 'recorded rating');
  await inFrame("(d.querySelector('#rating-choice-2').checked=true,d.querySelector('#rating-comment').value='Revised',d.querySelector('#rating-send').click())");
  await wait(() => inFrame("d.querySelector('#rating-outcome').textContent.includes('updated')"), 'revised rating');
  assert.deepEqual(requests.filter(({ path }) => path.endsWith('/rating')).map(({ body }) => body), [
    { rating: 5, comment: 'Helpful' }, { rating: 2, comment: 'Revised' },
  ]);
  assert.equal(requests.filter(({ path }) => path.endsWith('/rating')).every(({ bearer }) => bearer === `Bearer ${'b'.repeat(64)}`), true);
  assert.equal(responses.filter(({ path }) => path.endsWith('/rating')).every(({ status }) => status === 200), true);
  // Keyboard focus reaches the native disclosure and radio controls.
  await inFrame("d.querySelector('details summary').scrollIntoView({block:'center'})");
  const frameOffset = await evaluate("(()=>{const r=document.querySelector('iframe').getBoundingClientRect();return {x:r.x,y:r.y};})()", hostContext);
  const summaryPoint = await inFrame("(()=>{const r=d.querySelector('details summary').getBoundingClientRect();return {x:r.x+r.width/2,y:r.y+r.height/2};})()");
  const point = { x: frameOffset.x + summaryPoint.x, y: frameOffset.y + summaryPoint.y };
  await cdp('Input.dispatchMouseEvent', { type: 'mousePressed', ...point, button: 'left', clickCount: 1 });
  await cdp('Input.dispatchMouseEvent', { type: 'mouseReleased', ...point, button: 'left', clickCount: 1 });
  assert.equal(await inFrame("d.activeElement === d.querySelector('details summary')"), true);
  await inFrame("d.querySelector('details').open=false");
  await key('Enter', 'Enter', 13);
  assert.equal(await inFrame("d.querySelector('details').open"), true);
  console.log('Chromium rating interaction: PASS (clarification, 1–5 selection, comment, revision, disclosure, sandbox)');
} finally {
  socket?.close();
  if (browser && browser.exitCode === null) {
    const exited = once(browser, 'exit');
    browser.kill('SIGTERM');
    await exited;
  }
  for (const server of servers) await new Promise((resolve) => server.close(resolve));
  await rm(directory, { recursive: true, force: true });
}
