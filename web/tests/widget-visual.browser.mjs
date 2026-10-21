// Opt-in visual/interaction evidence. Usage: node web/tests/widget-visual.browser.mjs /absolute/assistant /absolute/output-dir
import assert from 'node:assert/strict';
import fs from 'node:fs';
import http from 'node:http';
import path from 'node:path';
import { spawn } from 'node:child_process';
import { once } from 'node:events';

const [binary, directory] = process.argv.slice(2);
assert.ok(binary && directory && path.isAbsolute(binary) && path.isAbsolute(directory));
fs.mkdirSync(directory, { recursive: true });
assert.ok(!fs.existsSync(`${directory}/assistant.sqlite`), 'Use a fresh output directory');
const processes = [], servers = [], contexts = new Map(), pending = new Map();
const results = [];
let socket, session, sequence = 0;
const pause = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
async function wait(predicate, label) {
  for (let n = 0; n < 240; n++) {
    const value = await predicate();
    if (value) return value;
    await pause(50);
  }
  throw new Error(`Timed out: ${label}`);
}
function check(label, value) {
  results.push({ label, pass: !!value });
  assert.ok(value, label);
}
function start(command, args, env, log) {
  const fd = fs.openSync(`${directory}/${log}`, 'a');
  const child = spawn(command, args, { cwd: directory, env, stdio: ['ignore', fd, fd] });
  fs.closeSync(fd);
  processes.push(child);
}
function cdp(method, params = {}, target = session) {
  return new Promise((resolve, reject) => {
    const id = ++sequence;
    pending.set(id, { resolve, reject });
    socket.send(JSON.stringify({ id, method, params, ...(target ? { sessionId: target } : {}) }));
  });
}
async function evaluate(id, expression) {
  const result = await cdp('Runtime.evaluate', { expression, contextId: id, returnByValue: true });
  if (result.exceptionDetails) throw new Error(result.exceptionDetails.text);
  return result.result.value;
}
async function click(id, selector, hostId) {
  await evaluate(id, `document.querySelector(${JSON.stringify(selector)}).scrollIntoView({block:'center'})`);
  const point = await evaluate(id, `(()=>{const r=document.querySelector(${JSON.stringify(selector)}).getBoundingClientRect();return {x:r.x+r.width/2,y:r.y+r.height/2}})()`);
  if (hostId) {
    const offset = await evaluate(hostId, `(()=>{const f=document.querySelector('iframe'),r=f.getBoundingClientRect();return {x:r.x+f.clientLeft,y:r.y+f.clientTop}})()`);
    point.x += offset.x;
    point.y += offset.y;
  }
  await cdp('Input.dispatchMouseEvent', { type: 'mousePressed', ...point, button: 'left', clickCount: 1 });
  await cdp('Input.dispatchMouseEvent', { type: 'mouseReleased', ...point, button: 'left', clickCount: 1 });
}
async function screenshot(name) {
  const { data } = await cdp('Page.captureScreenshot', { format: 'png' });
  fs.writeFileSync(`${directory}/${name}.png`, Buffer.from(data, 'base64'));
}
try {
  let port;
  const server = http.createServer((_request, response) => {
    response.setHeader('Content-Type', 'text/html; charset=utf-8');
    // Host variables deliberately conflict with the iframe palette.
    response.end(`<!doctype html><meta name="viewport" content="width=device-width,initial-scale=1"><title>Local embed fixture</title><style>:root{--accent:hotpink;--border:magenta;--bg:yellow;--space-3:100px}body{margin:0;background:#fff;font:16px system-ui}main{padding:32px;max-width:700px}h1{font-size:2rem}</style><main><h1>Sample support host</h1><p>Local-only embed fixture.</p></main><script src="http://127.0.0.1:${port}/widget/loader.js" data-page-path="/help" defer></script>`);
  });
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  servers.push(server);
  const host = `http://127.0.0.1:${server.address().port}`;
  const probe = http.createServer();
  await new Promise((resolve) => probe.listen(0, '127.0.0.1', resolve));
  port = probe.address().port;
  await new Promise((resolve) => probe.close(resolve));
  const origin = `http://127.0.0.1:${port}`;
  start(binary, ['serve', '--seed-demo'], {
    PATH: process.env.PATH, HOME: directory, STORE_MODE: 'sqlite', SQLITE_PATH: `${directory}/assistant.sqlite`,
    OPENAI_API_KEY: '', TURSO_DATABASE_URL: '', TURSO_AUTH_TOKEN: '',
    ASSISTANT_ORIGIN: origin, LISTEN_ADDR: `127.0.0.1:${port}`, WIDGET_PARENT_ORIGINS: host,
  }, 'backend.log');
  await wait(async () => { try { return (await fetch(`${origin}/widget/widget.html`)).ok; } catch { return false; } }, 'offline server');
  start(process.env.CHROMIUM || '/usr/bin/chromium', [
    '--headless', '--disable-gpu', '--no-first-run', '--no-default-browser-check', '--disable-background-networking',
    '--disable-component-update', '--disable-sync', '--proxy-server=http://127.0.0.1:9',
    '--proxy-bypass-list=127.0.0.1;localhost', '--remote-debugging-address=127.0.0.1', '--remote-debugging-port=0',
    `--user-data-dir=${directory}/chromium-profile`, '--window-size=1280,900', 'about:blank',
  ], { PATH: process.env.PATH, HOME: directory }, 'chromium.log');
  const debuggerPort = await wait(() => {
    try { return fs.readFileSync(`${directory}/chromium-profile/DevToolsActivePort`, 'utf8').split('\n')[0]; } catch { return false; }
  }, 'Chromium');
  const browser = await (await fetch(`http://127.0.0.1:${debuggerPort}/json/version`)).json();
  socket = new WebSocket(browser.webSocketDebuggerUrl);
  await once(socket, 'open');
  socket.addEventListener('message', (event) => {
    const message = JSON.parse(event.data);
    if (message.id) {
      const request = pending.get(message.id);
      pending.delete(message.id);
      if (message.error) request.reject(new Error(JSON.stringify(message.error)));
      else request.resolve(message.result);
    } else if (message.method === 'Runtime.executionContextCreated') contexts.set(message.params.context.id, message.params.context);
    else if (message.method === 'Runtime.executionContextDestroyed') contexts.delete(message.params.executionContextId);
    else if (message.method === 'Runtime.executionContextsCleared') contexts.clear();
  });
  const target = await cdp('Target.createTarget', { url: 'about:blank' }, null);
  session = (await cdp('Target.attachToTarget', { targetId: target.targetId, flatten: true }, null)).sessionId;
  for (const domain of ['Page', 'Runtime']) await cdp(`${domain}.enable`);
  for (const [name, width, height, dark] of [['mobile-dark', 360, 640, true], ['desktop', 1280, 900, false], ['mobile', 360, 640, false], ['zoom-200', 180, 320, false]]) {
    await cdp('Emulation.setDeviceMetricsOverride', { width, height, deviceScaleFactor: 1, mobile: name !== 'desktop' });
    await cdp('Emulation.setEmulatedMedia', { features: [{ name: 'prefers-color-scheme', value: dark ? 'dark' : 'light' }] });
    await cdp('Page.navigate', { url: host });
    const h = await wait(async () => {
      for (const [id, context] of contexts) {
        if (!context.auxData?.isDefault) continue;
        try { if (await evaluate(id, `location.origin===${JSON.stringify(host)}&&!!document.querySelector('.assistant-launcher')`)) return id; } catch { /* navigation */ }
      }
      return false;
    }, 'launcher');
    check(`${name}: browser preference emulated`, await evaluate(h, `matchMedia('(prefers-color-scheme: dark)').matches===${dark}`));
    check(`${name}: host stays white`, await evaluate(h, `getComputedStyle(document.body).backgroundColor==='rgb(255, 255, 255)'`));
    check(`${name}: independent launcher color`, await wait(() => evaluate(h, `getComputedStyle(document.querySelector('.assistant-launcher')).backgroundColor==='rgb(26, 115, 232)'`), 'launcher CSS'));
    check(`${name}: launcher text is readable`, await evaluate(h, `parseFloat(getComputedStyle(document.querySelector('.assistant-launcher')).fontSize)>=15`));
    await click(h, '.assistant-launcher');
    const f = await wait(async () => {
      for (const [id, context] of contexts) {
        if (!context.auxData?.isDefault) continue;
        try { if (await evaluate(id, `location.origin===${JSON.stringify(origin)}&&!document.querySelector('#send').disabled`)) return id; } catch { /* navigation */ }
      }
      return false;
    }, 'ready iframe');
    check(`${name}: frame fits`, await evaluate(h, `(()=>{const r=document.querySelector('iframe').getBoundingClientRect();return r.left>=0&&r.right<=innerWidth&&r.top>=0&&r.bottom<=innerHeight})()`));
    const geometry = `(()=>{const pane=document.querySelector('.chat-content'),q=document.querySelector('#question').getBoundingClientRect(),s=document.querySelector('#send').getBoundingClientRect();return q.width>0&&q.height>0&&s.width>0&&s.height>0&&q.top>=0&&s.bottom<=innerHeight&&q.right<=innerWidth&&s.right<=innerWidth&&document.documentElement.scrollHeight<=innerHeight+2&&pane.clientHeight>0})()`;
    check(`${name}: ready composer visible without scrolling`, await evaluate(f, geometry));
    check(`${name}: base and essential text sizes`, await evaluate(f, `(()=>{
      const size=(selector)=>parseFloat(getComputedStyle(document.querySelector(selector)).fontSize);
      return size('html')>=16 && size('#question')>=16 && size('#send')>=16
        && size('.disclosure')>=13 && size('#status')>=13
        && size('.eyebrow')>=12 && size('.chat-header p:last-child')>=13
        && size('#compose-hint')>=12.5 && size('#escalate-hint')>=13
        && size('.consent')>=13 && size('#escalate-error')>=13
        && size('#escalate-outcome')>=13;
    })()`));
    // White must remain white with a dark-preferring OS/browser, not merely in light mode.
    check(`${name}: browser preference reaches iframe`, await evaluate(f, `matchMedia('(prefers-color-scheme: dark)').matches===${dark}`));
    check(`${name}: white window surfaces and light form controls`, await evaluate(f, `(()=>{
       const color=(selector)=>getComputedStyle(document.querySelector(selector)).backgroundColor;
       const regions=['.chat-header','.chat-content','#message-form','.disclosure','#question'];
       return regions.every((selector)=>color(selector)==='rgb(255, 255, 255)')
         && getComputedStyle(document.documentElement).colorScheme==='light'
         && getComputedStyle(document.querySelector('#question')).colorScheme==='light';
     })()`));
    check(`${name}: loader frame stays white`, await evaluate(h, `getComputedStyle(document.querySelector('iframe')).backgroundColor==='rgb(255, 255, 255)'`));
      check(`${name}: concise disclosure present before send without origin or port`, await evaluate(f, `(()=>{const d=document.querySelector('.disclosure'),t=d.textContent;return t.includes('AI may be wrong')&&t.includes('avoid sensitive data')&&t.includes("Sample content isn't official guidance")&&t.includes('Questions and recent chat go to the app server')&&t.includes('approved excerpts may go to the configured AI provider')&&t.includes('Reloading loses guest access')&&t.includes('server keeps its history')&&document.querySelector('.disclosure .more-info summary').textContent.includes('More information')&&!t.includes(location.origin)&&!/localhost|127\\.0\\.0\\.1|https?:\\/\\/|:\\d{2,5}\\b|OpenAI/i.test(t)&&getComputedStyle(d).display!=='none'})()`));
    check(`${name}: restricted sandbox`, await evaluate(h, `document.querySelector('iframe').getAttribute('sandbox')==='allow-scripts allow-same-origin'`));
    await screenshot(`${name}-ready`);
    // Only local, synthetic non-visitor text appears in long-history screenshots.
    await evaluate(f, `(()=>{const list=document.querySelector('#messages');for(let i=0;i<16;i++){const li=document.createElement('li'),p=document.createElement('p');li.className='message message-assistant';p.textContent=i===15?'W'.repeat(480):'Synthetic layout fixture '+i+' — illustrative conversation spacing only.';li.append(p);list.append(li)}document.querySelector('.chat-content').scrollTop=99999})()`);
    check(`${name}: message metadata remains legible`, await evaluate(f, `(()=>{const li=document.querySelector('.message'),label=document.createElement('span'),note=document.createElement('span');label.className='message-label';note.className='source-note';li.append(label,note);return parseFloat(getComputedStyle(label).fontSize)>=12.5&&parseFloat(getComputedStyle(note).fontSize)>=13})()`));
    check(`${name}: long conversation scrolls independently`, await evaluate(f, `(()=>{const p=document.querySelector('.chat-content');return p.scrollHeight>p.clientHeight&&p.scrollTop>0&&p.scrollWidth<=p.clientWidth+2})()`) && await evaluate(f, geometry));
    await screenshot(`${name}-long`);
    await evaluate(f, `document.querySelector('#messages').replaceChildren()`);
    await click(f, '#escalate-toggle', h);
    check(`${name}: handoff menu opens with aria-expanded`, await evaluate(f, `!document.querySelector('#handoff-menu').hidden&&document.querySelector('#escalate-toggle').getAttribute('aria-expanded')==='true'`));
    check(`${name}: WhatsApp is unavailable and disabled, never an operational channel`, await evaluate(f, `(()=>{const w=document.querySelector('#handoff-whatsapp');return w.disabled&&!document.querySelector('input[type="tel"]')&&document.querySelector('#whatsapp-reason').textContent.includes('not enabled WhatsApp')})()`));
    check(`${name}: no channel is selected before the visitor chooses one`, await evaluate(f, `(()=>{const w=document.querySelector('#handoff-whatsapp'),e=document.querySelector('#handoff-email');return w.getAttribute('aria-pressed')==='false'&&e.getAttribute('aria-pressed')==='false'&&document.querySelector('#escalation-form').hidden})()`));
    await screenshot(`${name}-handoff-menu`);
    // A real click on the unavailable channel changes nothing.
    await click(f, '#handoff-whatsapp', h);
    check(`${name}: clicking WhatsApp does nothing`, await evaluate(f, `(()=>{const w=document.querySelector('#handoff-whatsapp');return w.disabled&&document.querySelector('#handoff-menu').hidden===false&&document.querySelector('#escalation-form').hidden&&!document.querySelector('#escalate-error').textContent})()`));
    // Escape closes the disclosure and gives focus back to its trigger.
    await cdp('Input.dispatchKeyEvent', { type: 'rawKeyDown', windowsVirtualKeyCode: 27, nativeVirtualKeyCode: 27, code: 'Escape', key: 'Escape' });
    await cdp('Input.dispatchKeyEvent', { type: 'keyUp', windowsVirtualKeyCode: 27, nativeVirtualKeyCode: 27, code: 'Escape', key: 'Escape' });
    check(`${name}: Escape closes the handoff menu`, await evaluate(f, `document.querySelector('#handoff-menu').hidden&&document.querySelector('#escalate-toggle').getAttribute('aria-expanded')==='false'`));
    check(`${name}: Escape returns focus to the trigger`, await evaluate(f, `document.activeElement.id==='escalate-toggle'`));
    // A click outside the handoff region closes it as well.
    await click(f, '#escalate-toggle', h);
    check(`${name}: handoff menu reopens`, await evaluate(f, `!document.querySelector('#handoff-menu').hidden`));
    await click(f, '#status', h);
    check(`${name}: outside click closes the handoff menu`, await evaluate(f, `document.querySelector('#handoff-menu').hidden&&document.querySelector('#escalation-form').hidden`));
    await click(f, '#escalate-toggle', h);
    await click(f, '#handoff-email', h);
    check(`${name}: By email opens form and consent starts unchecked`, await evaluate(f, `!document.querySelector('#escalation-form').hidden&&!document.querySelector('#escalate-consent').checked&&document.querySelector('#handoff-email').getAttribute('aria-pressed')==='true'`));
    // The expanded handoff panel never pushes the pinned composer out of view.
    check(`${name}: composer stays visible with the handoff panel open`, await evaluate(f, geometry));
    await click(f, '#escalate-send', h);
    check(`${name}: consent gate`, await evaluate(f, `document.querySelector('#escalate-error').textContent.includes('consent')`));
    await evaluate(f, `document.querySelector('#escalate-send').scrollIntoView({block:'end'})`);
    check(`${name}: handoff button reachable by scrolling`, await wait(async () => {
      const view = await evaluate(f, `(()=>{const pane=document.querySelector('.chat-content'),r=document.querySelector('#escalate-send').getBoundingClientRect(),p=pane.getBoundingClientRect();return {top:r.top,bottom:r.bottom,left:r.left,right:r.right,paneTop:p.top,paneBottom:p.bottom,paneLeft:p.left,paneRight:p.right,scrollTop:pane.scrollTop,scrollHeight:pane.scrollHeight,clientHeight:pane.clientHeight}})()`);
      if (view.top >= view.paneTop && view.bottom <= view.paneBottom + 2 && view.left >= view.paneLeft && view.right <= view.paneRight + 2) return true;
      return false;
    }, 'handoff scroll'));
    await screenshot(`${name}-email-consent`);
    if (name === 'desktop') {
      await click(f, '#question', h);
      await cdp('Input.insertText', { text: 'Can you explain quantum entanglement?' });
      await click(f, '#send', h);
      check('real click: clarification turn', await wait(() => evaluate(f, `document.querySelector('#messages').children.length===2&&document.querySelector('#status').textContent.includes('Clarification received')`), 'clarification'));
      // The chat interaction closed the handoff disclosure; the visitor reopens it.
      await click(f, '#escalate-toggle', h);
      await click(f, '#handoff-email', h);
      check('handoff reopens after a chat interaction', await evaluate(f, `!document.querySelector('#handoff-menu').hidden&&!document.querySelector('#escalation-form').hidden`));
      await click(f, '#escalate-email', h);
      await cdp('Input.insertText', { text: 'visitor@example.org' });
      await click(f, '#escalate-consent', h);
      await click(f, '#escalate-send', h);
      check('real click: recorded, no email sent', await wait(() => evaluate(f, `document.querySelector('#escalate-outcome').textContent.includes('no email has been sent')`), 'recorded outcome'));
      // Closing and reopening the disclosure keeps what the visitor typed.
      await click(f, '#question', h);
      await click(f, '#escalate-toggle', h);
      await click(f, '#handoff-email', h);
      check('typed support details survive a close and reopen', await evaluate(f, `document.querySelector('#escalate-email').value==='visitor@example.org'&&document.querySelector('#escalate-outcome').textContent.includes('no email has been sent')`));
    }
    await click(h, '.assistant-launcher');
    check(`${name}: launcher closes`, await evaluate(h, `document.querySelector('iframe').hidden&&document.querySelector('.assistant-launcher').getAttribute('aria-expanded')==='false'`));
  }
  await cdp('Emulation.setDeviceMetricsOverride', { width: 1280, height: 900, deviceScaleFactor: 1, mobile: false });
  await cdp('Emulation.setEmulatedMedia', { features: [{ name: 'prefers-color-scheme', value: 'dark' }] });
  await cdp('Page.navigate', { url: `${origin}/widget/sample-host.html` });
  const sample = await wait(async () => {
    for (const [id, context] of contexts) {
      if (!context.auxData?.isDefault) continue;
      try { if (await evaluate(id, `location.pathname==='/widget/sample-host.html'&&document.querySelector('.sample-page')!==null`)) return id; } catch { /* navigation */ }
    }
    return false;
  }, 'sample host');
  check('dark preference: actual sample host stays white with light controls', await wait(() => evaluate(sample, `matchMedia('(prefers-color-scheme: dark)').matches&&getComputedStyle(document.body).backgroundColor==='rgb(255, 255, 255)'&&getComputedStyle(document.documentElement).colorScheme==='light'`), 'sample host stylesheet'));
  await screenshot('sample-host-dark');
} catch (error) {
  results.push({ label: 'fatal', pass: false, detail: String(error.stack || error) });
  process.exitCode = 1;
} finally {
  if (socket) socket.close();
  for (const child of processes.reverse()) {
    if (child.exitCode !== null) continue;
    const exit = once(child, 'exit');
    child.kill('SIGTERM');
    await Promise.race([exit, pause(2000)]);
    if (child.exitCode === null) { child.kill('SIGKILL'); await exit; }
  }
  for (const server of servers) if (server.listening) await new Promise((resolve) => server.close(resolve));
  fs.writeFileSync(`${directory}/results.json`, `${JSON.stringify(results, null, 2)}\n`);
  console.log(JSON.stringify({ passed: results.filter((result) => result.pass).length, failed: results.filter((result) => !result.pass).length }));
}
