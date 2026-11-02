import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { creation, emptyHistory, json, loaderFixture, settle, turn, walk, widgetFixture } from './harness.mjs';

test('iframe handshake rejects wrong source/origin/version and initializes only once', async () => {
  const fixture = widgetFixture();
  const good = { type: 'assistant:init', version: 1, context: { page_path: '/help', locale: 'en' } };
  fixture.init(good, 'https://host.example.attacker');
  fixture.init(good, 'https://host.example', {});
  await settle();
  assert.equal(fixture.requests.length, 0);
  fixture.init(good); fixture.init(good); await settle();
  assert.equal(fixture.requests.length, 2);
  assert.deepEqual(JSON.parse(fixture.requests[0].init.body), { context: good.context });
  fixture.init({ ...good, version: 2 }); await settle();
  assert.equal(fixture.requests.length, 2);
});

test('invalid init/context never fetches and shows a visible configuration error', async () => {
  for (const data of [null, {}, { type: 'assistant:init', version: 2, context: {} }, { type: 'assistant:init', version: 1, context: { raw_html: '<body>' } }, { type: 'assistant:init', version: 1, context: { page_path: '/help?token=secret' } }, { type: 'assistant:init', version: 1, context: {}, capability: 'forged' }]) {
    const fixture = widgetFixture(); fixture.init(data); await settle();
    assert.equal(fixture.requests.length, 0);
    assert.equal(fixture.ui.error.hidden, false);
    assert.match(fixture.ui.error.textContent, /invalid context/u);
  }
});

test('bad or missing parent origin and handshake timeout are visible no-fetch failures', async () => {
  for (const hash of ['', '#parent_origin=null', '#parent_origin=https%3A%2F%2Fhost.example%2F', '#parent_origin=https%3A%2F%2Fhost.example&extra=1', '#parent_origin=https://host.example&parent_origin=https://evil.example']) {
    const fixture = widgetFixture([], { location: { hash } }); fixture.init(); await settle();
    assert.equal(fixture.requests.length, 0);
    assert.equal(fixture.ui.error.hidden, false);
    assert.equal(fixture.sent.length, 0);
  }
  const fixture = widgetFixture();
  [...fixture.timers.values()].find((timer) => timer.delay === 10000).fn();
  fixture.init(); await settle();
  assert.match(fixture.ui.error.textContent, /not verified/u);
  assert.equal(fixture.requests.length, 0);
});

test('loader verifies exact iframe source, assistant origin, schema and protocol version', async () => {
  const fixture = await loaderFixture({ 'data-page-path': '/help', 'data-locale': 'en' }); fixture.open();
  const frame = fixture.frame();
  const ready = { type: 'assistant:ready', version: 1 };
  fixture.message(ready, 'https://assistant.example.evil');
  fixture.message(ready, 'https://assistant.example', {});
  fixture.message({ ...ready, version: 2 });
  fixture.message({ ...ready, transcript: 'bad' });
  assert.equal(frame.contentWindow.sent.length, 0);
  fixture.message(ready); fixture.message(ready);
  assert.equal(frame.contentWindow.sent.length, 1);
  const sent = JSON.parse(JSON.stringify(frame.contentWindow.sent[0]));
  assert.deepEqual(sent, { data: { type: 'assistant:init', version: 1, context: { page_path: '/help', locale: 'en' } }, origin: 'https://assistant.example' });
  assert.equal(frame.src, 'https://assistant.example/widget/widget.html#parent_origin=https%3A%2F%2Fhost.example');
});

test('resize accepts only exact source/origin/schema and clamps 300–720', async () => {
  const fixture = await loaderFixture(); fixture.open();
  const frame = fixture.frame();
  const resize = { type: 'assistant:resize', version: 1, height: 9000 };
  fixture.message(resize); assert.equal(frame.height, '600');
  fixture.message({ type: 'assistant:ready', version: 1 });
  fixture.message(resize, 'https://evil.example'); assert.equal(frame.height, '600');
  fixture.message({ ...resize, height: '9000' }); assert.equal(frame.height, '600');
  fixture.message({ ...resize, height: Infinity }); assert.equal(frame.height, '600');
  fixture.message({ ...resize, transcript: 'bad' }); assert.equal(frame.height, '600');
  fixture.message(resize); assert.equal(frame.height, '720');
  fixture.message({ ...resize, height: -500 }); assert.equal(frame.height, '300');
});

test('embed origin validation permits HTTPS/loopback only and never falls back to parent URL', async () => {
  for (const origin of ['https://assistant.example/path', 'https://a.example/', 'https://user:pass@a.example', 'http://remote.example', 'null', '*']) {
    const fixture = await loaderFixture({ 'data-assistant-origin': origin });
    assert.equal(fixture.launcher, undefined);
    assert.match(fixture.document.body.textContent, /could not start/u);
  }
  const fixture = await loaderFixture({ 'data-assistant-origin': 'http://127.0.0.1:8080' }); fixture.open();
  assert.match(fixture.frame().src, /^http:\/\/127\.0\.0\.1:8080\/widget\/widget.html/u);
});

test('capability is only a bearer in iframe-owned API calls, never URL/DOM/parent messages', async () => {
  const fixture = widgetFixture([json(creation, 201), emptyHistory(), json(turn())]);
  fixture.init(); await settle(); fixture.submit(); await settle();
  assert.equal(fixture.requests[0].url, 'https://assistant.example/api/v1/conversations');
  assert.equal(fixture.requests[1].url, `https://assistant.example/api/v1/conversations/${creation.conversation_id}/messages?limit=20`);
  for (const [index, request] of fixture.requests.entries()) {
    assert.equal(new URL(request.url).origin, fixture.platform.location.origin);
    assert.equal(request.init.credentials, 'omit');
    assert.equal(request.init.mode, 'same-origin');
    assert.equal(request.init.redirect, 'error');
    assert.equal(request.init.cache, 'no-store');
    assert.doesNotMatch(request.url, new RegExp(creation.capability));
    assert.equal(request.init.headers.Authorization, index === 0 ? undefined : `Bearer ${creation.capability}`);
  }
  const publicData = JSON.stringify(fixture.sent) + fixture.document.body.textContent;
  assert.doesNotMatch(publicData, new RegExp(creation.capability));
  assert.doesNotMatch(JSON.stringify(fixture.sent), /How do I start|fictional demo guide|conversation_id|capability/u);
  for (const message of fixture.sent) {
    assert.equal(message.origin, 'https://host.example');
    assert.ok(['assistant:ready', 'assistant:resize'].includes(message.data.type));
  }
});

test('close control uses exact versioned shape, source and origin; invalid messages cannot close', async () => {
  const fixture = await loaderFixture(); fixture.open();
  const frame = fixture.frame();
  const close = { type: 'assistant:close', version: 1 };
  fixture.message(close); // No close before handshake.
  assert.equal(frame.hidden, false);
  fixture.message({ type: 'assistant:ready', version: 1 });
  for (const [data, origin, source] of [
    [close, 'https://assistant.example.evil', frame.contentWindow],
    [close, 'https://assistant.example', {}],
    [{ ...close, version: 2 }, 'https://assistant.example', frame.contentWindow],
    [{ ...close, capability: 'bad' }, 'https://assistant.example', frame.contentWindow],
    [{ ...close, height: 300 }, 'https://assistant.example', frame.contentWindow],
    [{ type: 'assistant:resize', version: 1 }, 'https://assistant.example', frame.contentWindow],
  ]) {
    fixture.message(data, origin, source);
    assert.equal(frame.hidden, false);
  }
  fixture.message(close);
  assert.equal(frame.hidden, true);
  assert.equal(fixture.launcher.focused, true);
  assert.equal(fixture.launcher.getAttribute('aria-expanded'), 'false');
  assert.equal(fixture.launcher.getAttribute('aria-label'), 'Open support assistant');
  fixture.open();
  assert.equal(fixture.frame(), frame);
  assert.equal(frame.hidden, false);
  assert.equal(frame.contentWindow.sent.length, 1);
});

test('disclosure keeps the safety warning visible and names app server and configurable provider without displaying origin before send', async () => {
  const fixture = widgetFixture();
  const html = await readFile(new URL('../assets/widget.html', import.meta.url), 'utf8');
  assert.match(fixture.ui.transmission.textContent, /Questions and recent chat go to the app server/u);
  assert.match(fixture.ui.transmission.textContent, /app server/u);
  assert.match(fixture.ui.transmission.textContent, /approved excerpts may go to the configured AI provider/u);
  assert.doesNotMatch(fixture.ui.transmission.textContent, /https?:\/\/|localhost|127\.0\.0\.1|:\d{2,5}\b|OpenAI/iu);
  assert.ok(!fixture.ui.transmission.textContent.includes(fixture.platform.location.origin));
  // Immediate safety warning stays visible; sharing and guest facts sit in More information.
  assert.match(html, /AI may be wrong/u);
  assert.match(html, /avoid sensitive data/u);
  assert.match(html, /Sample content isn't official guidance/u);
  assert.match(html, /<details class="more-info">[\s\S]*?<summary>More information<\/summary>[\s\S]*?<p id="transmission"><\/p>[\s\S]*?Reloading loses guest access to this chat, but the server keeps its history\.<\/p>[\s\S]*?<\/details>/u);
  assert.equal(fixture.ui.send.disabled, true);
});

test('malformed creation/history/turn envelopes never unlock unsafe success', async () => {
  const badCreation = widgetFixture([json({ ...creation, capability: 'bad' }, 201)]);
  badCreation.init(); await settle(); assert.equal(badCreation.ui.send.disabled, true);
  const badHistory = widgetFixture([json(creation, 201), json({ messages: [turn().messages[1]], next_before: 999 })]);
  badHistory.init(); await settle(); assert.equal(badHistory.ui.send.disabled, true);
  const badTurn = turn(); badTurn.messages[0].text = 'Unexpected question';
  const fixture = widgetFixture([json(creation, 201), emptyHistory(), json(badTurn), emptyHistory()]);
  fixture.init(); await settle(); fixture.submit(); await settle();
  assert.equal(fixture.ui.messages.children.length, 1);
  assert.equal(fixture.ui.send.disabled, true);
  assert.equal(fixture.requests.at(-1).init.method, 'GET');
});

test('response byte, media type and UTF-8 boundaries fail closed', async () => {
  for (const response of [
    new Response('<html>secret</html>', { status: 201, headers: { 'Content-Type': 'text/html' } }),
    new Response('x'.repeat(1048577), { status: 201, headers: { 'Content-Type': 'application/json' } }),
    new Response(new Uint8Array([0xff]), { status: 201, headers: { 'Content-Type': 'application/json' } }),
  ]) {
    const fixture = widgetFixture([response]); fixture.init(); await settle();
    assert.equal(fixture.ui.send.disabled, true);
    assert.equal(fixture.requests.length, 1);
    assert.equal(fixture.ui.error.hidden, false);
    assert.doesNotMatch(fixture.document.body.textContent, /secret/u);
  }
});

test('a legitimate maximum-text history page over 256 KiB still loads', async () => {
  const messages = [];
  for (let number = 1; number <= 10; number++) {
    const pair = turn(`large_${String(number).padStart(4, '0')}`, '😀'.repeat(2000), number);
    pair.messages[1].text = '😀'.repeat(4000);
    pair.messages[1].sources = [1, 2, 3].map((source) => ({ id: `demo-${source}`, title: '😀'.repeat(160), url: `https://example.org/${'a'.repeat(2000)}` }));
    messages.push(...pair.messages);
  }
  const page = { messages, next_before: null };
  assert.ok(new TextEncoder().encode(JSON.stringify(page)).length > 262144);
  const fixture = widgetFixture([json(creation, 201), json(page)]);
  fixture.init(); await settle();
  assert.equal(fixture.ui.messages.children.length, 20);
  assert.equal(fixture.ui.send.disabled, false);
});

test('published assets use local external files, semantic form and actual sample loader', async () => {
  const html = await readFile(new URL('../assets/widget.html', import.meta.url), 'utf8');
  const sample = await readFile(new URL('../assets/sample-host.html', import.meta.url), 'utf8');
  assert.match(html, /<script type="module" src="widget.js"><\/script>/u);
  assert.match(html, /<label for="question">Your question<\/label>/u);
  assert.match(html, /<form id="message-form" aria-label="Ask the support assistant">/u);
  assert.match(html, /<button id="send" type="button" disabled>Send question<\/button>/u);
  assert.doesNotMatch(html, /type="submit"/u);
  assert.match(html, /aria-describedby="compose-hint" aria-keyshortcuts="Control\+Enter Meta\+Enter"/u);
  assert.match(html, /id="compose-hint">[^<]*Enter adds a line/u);
  assert.match(html, /AI may be wrong/u);
  assert.match(html, /avoid sensitive data/u);
  assert.match(html, /Sample content isn't official guidance/u);
  assert.match(html, /<summary>More information<\/summary>/u);
  assert.doesNotMatch(html, /DEMO \/ AI SUPPORT|fictional demo/u);
  assert.doesNotMatch(html, /<style| style=| on(?:click|load|submit)=/u);
  assert.match(sample, /<script src="loader.js" data-page-path="\/demo\/getting-started" data-locale="en" defer><\/script>/u);
  assert.match(sample, /Preview only: the knowledge is fictional, not official guidance or a staffed support service/u);
  assert.match(sample, /no email is sent/u);
  assert.doesNotMatch(sample, /assistant demo|FICTIONAL DEMO|demo onboarding/u);
  assert.doesNotMatch(sample, /<iframe|message-form|id="messages"/u);
  // Real mount already exercises control listeners and text-only DOM; these are
  // packaging checks, not a replacement for behavioral or browser tests.
  const fixture = await loaderFixture(); fixture.open();
  assert.equal(walk(fixture.document.body).filter((node) => node.tagName === 'IFRAME').length, 1);
});

test('handoff menu is additive markup: Request via email available, Request via WhatsApp disabled, no phone field', async () => {
  const html = await readFile(new URL('../assets/widget.html', import.meta.url), 'utf8');
  // The menu and email form are in the scrollable pane, never nested inside the pinned chat form.
  assert.match(html, /<section class="escalation" aria-label="Request human support">[\s\S]*?<\/section>[\s\S]*?<\/div>\s*<form id="message-form"/u);
  assert.match(html, /<button id="escalate-toggle" type="button" disabled aria-expanded="false" aria-controls="handoff-menu">Request human support<\/button>/u);
  assert.match(html, /<div id="handoff-menu" class="handoff-menu" role="group" aria-label="Ways to request human support" hidden>/u);
  assert.match(html, /<button id="handoff-email" type="button" aria-expanded="false" aria-controls="escalation-form" aria-pressed="false">Request via email<\/button>/u);
  assert.match(html, /<button id="handoff-whatsapp" type="button" disabled aria-describedby="whatsapp-reason" aria-pressed="false">Request via WhatsApp<\/button>/u);
  assert.match(html, /<p id="whatsapp-reason">Unavailable: this support service has not enabled WhatsApp\.<\/p>/u);
  assert.match(html, /<form id="escalation-form" aria-label="Request human support by email" hidden>/u);
  assert.match(html, /<label for="escalate-email">Your email<\/label>/u);
  assert.match(html, /<input id="escalate-email" name="email" type="text" inputmode="email" autocomplete="email" placeholder="you@example\.org" aria-describedby="escalate-hint" required disabled>/u);
  assert.match(html, /<label for="escalate-note">Optional note<\/label>/u);
  assert.match(html, /<textarea id="escalate-note" name="question_note" rows="2"/u);
  // Consent is a standalone, never pre-checked checkbox with its own label.
  assert.match(html, /<input id="escalate-consent" name="consent" type="checkbox">/u);
  assert.match(html, /<label class="consent"><input id="escalate-consent" name="consent" type="checkbox"><span>I agree to store this request and note for human support by email\. No email is sent while delivery is not configured\.<\/span><\/label>/u);
  assert.match(html, /<p id="escalate-error" role="alert" hidden><\/p>/u);
  assert.match(html, /<p id="escalate-outcome" role="status" aria-live="polite" hidden><\/p>/u);
  assert.match(html, /<button id="escalate-send" type="button" disabled>Send request<\/button>/u);
  assert.match(html, /id="escalate-hint">Consent is required before a request is recorded\. Email ≤254 bytes; note ≤1,000 characters \(2,000 bytes\)\.<\/span>/u);
  // WhatsApp never appears to be an operational channel and no phone input exists.
  assert.doesNotMatch(html, /type="tel"|inputmode="tel"|name="phone"|name="whatsapp"|wa\.me|api\.whatsapp/iu);
  assert.doesNotMatch(html, /checked(?!=)/u);
  assert.doesNotMatch(html, /type="submit"/u);
  assert.doesNotMatch(html, /<style| style=| on(?:click|load|submit)=/u);
  // The sample host still demonstrates only the loader; escalation stays in the iframe.
  const sample = await readFile(new URL('../assets/sample-host.html', import.meta.url), 'utf8');
  assert.doesNotMatch(sample, /escalat/iu);
  // Widget core actually wires the additive controls; handler coverage lives in the unit tests.
  const core = await readFile(new URL('../assets/widget-core.mjs', import.meta.url), 'utf8');
  assert.match(core, /ui\['escalate-toggle'\]\.addEventListener\('click', /u);
  assert.match(core, /ui\['handoff-email'\]\.addEventListener\('click', /u);
  assert.match(core, /ui\['escalate-send'\]\.addEventListener\('click', submitEscalation\)/u);
  assert.match(core, /ui\['escalation-form'\]\.addEventListener\('submit', \(event\) => \{/u);
  assert.doesNotMatch(core, /innerHTML/u);
});

test('only the visitor opens the handoff menu, and the composer stays pinned outside it', async () => {
  const core = await readFile(new URL('../assets/widget-core.mjs', import.meta.url), 'utf8');
  // No server, provider or history event can open the menu: the only write is
  // the visitor toggle, and the unavailable channel is never selected.
  assert.match(core, /handoffOpen = !handoffOpen;/u);
  assert.doesNotMatch(core, /handoffOpen\s*=\s*true/u);
  assert.match(core, /ui\['handoff-whatsapp'\]\.setAttribute\('aria-pressed', 'false'\)/u);
  // The header button and Escape share the initialized/non-stopped close path.
  assert.match(core, /ui\['chat-close'\]\.addEventListener\('click', closeChat\)/u);
  assert.match(core, /function closeChat\(\) \{\s*if \(stopped \|\| !initialized\) return;/u);
  // Escape is the only key requesting host close, including with this disclosure open.
  assert.match(core, /if \(event\.key !== 'Escape' \|\| stopped \|\| !initialized\) return;/u);
  assert.match(core, /platform\.parent\.postMessage\(\{ type: 'assistant:close', version: 1 \}, parent\)/u);
  assert.doesNotMatch(core, /event\.key === 'q'/u);
  // The handoff panel lives in the scrollable pane; the composer stays outside
  // it, so the question and Send controls remain visible on a small screen.
  const html = await readFile(new URL('../assets/widget.html', import.meta.url), 'utf8');
  assert.match(html, /<div class="chat-content"[^>]*>[\s\S]*?<section class="escalation"[\s\S]*?<\/div>\s*<form id="message-form"/u);
  assert.match(html, /<header class="chat-header">[\s\S]*?<button id="chat-close" type="button" class="chat-close" aria-label="Close chat">×<\/button>\s*<\/header>/u);
});
