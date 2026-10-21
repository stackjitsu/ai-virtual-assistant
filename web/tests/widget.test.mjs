import test from 'node:test';
import assert from 'node:assert/strict';
import { validContext, validHistory, validMessage, validOrigin, validSource, validText, retrySeconds } from '../assets/protocol.mjs';
import { creation, emptyHistory, fail, id, json, loaderFixture, settle, turn, walk, widgetFixture } from './harness.mjs';

test('origin and optional context boundaries accept only bounded declared values', () => {
  for (const origin of ['https://assistant.example', 'http://localhost:8080', 'http://127.0.0.1:8080', 'http://[::1]:8080']) assert.equal(validOrigin(origin), true);
  for (const origin of ['null', '*', 'http://example.org', 'https://a.example/', 'https://a.example/path', 'https://user:pass@a.example', 'https://a.example?token=x']) assert.equal(validOrigin(origin), false);
  for (const context of [{}, { page_path: '/help', locale: 'en-US' }, { locale: 'fil' }]) assert.equal(validContext(context), true);
  for (const context of [null, [], { page_path: '//host/path' }, { page_path: '/a?token=secret' }, { page_path: '/a#x' }, { page_path: '/a\\b' }, { page_path: '/a\u0085' }, { page_path: `/${'é'.repeat(128)}` }, { locale: 'en_US' }, { account_id: 'secret' }]) assert.equal(validContext(context), false);
});

test('pre-send transmission text identifies server and possible provider without rendering an origin or port', () => {
  for (const origin of ['https://assistant.example', 'http://127.0.0.1:8765', 'http://localhost:8080']) {
    const fixture = widgetFixture([], { location: { origin } });
    const notice = fixture.ui.transmission.textContent;
    assert.match(notice, /Questions and recent chat go to the app server/u);
    assert.match(notice, /approved excerpts may go to the configured AI provider/u);
    assert.doesNotMatch(notice, /localhost|127\.0\.0\.1|https?:\/\/|:\d{2,5}\b/iu);
    assert.ok(!notice.includes(origin));
    assert.equal(fixture.ui.send.disabled, true);
  }
});

test('message, sources and text validation reject unsafe or incoherent envelopes', () => {
  const pair = turn();
  assert.equal(validMessage(pair.messages[1]), true);
  assert.equal(validText('é'.repeat(2000)), true);
  for (const text of ['', 'a'.repeat(2001), 'x\u0000', '\ud800', 'x\u0085']) assert.equal(validText(text), false);
  assert.equal(validText('line\nwith\ttabs'), true);
  for (const url of ['javascript:alert(1)', 'http://example.org', 'https://user:secret@example.org', 'https://example.org/\nsecret']) assert.equal(validSource({ id: 'demo', title: 'Guide', url }), false);
  assert.equal(validMessage({ ...pair.messages[1], sources: [] }), false);
  assert.equal(validMessage({ ...pair.messages[0], sources: pair.messages[1].sources }), false);
  assert.equal(validHistory({ messages: [...pair.messages].reverse(), next_before: null }), false);
  assert.equal(validHistory({ messages: [pair.messages[0], { ...pair.messages[1], request_id: 'other_id1' }], next_before: null }), false);
});

test('bounded retry-after parsing never produces an infinite or huge wait', () => {
  assert.equal(retrySeconds('60'), 60);
  assert.equal(retrySeconds('999999'), 120);
  assert.equal(retrySeconds('0'), 1);
  for (const value of [null, '-1', 'Infinity', 'Wed, 20 Oct', '9999999']) assert.equal(retrySeconds(value), 1);
});

test('loading disables form, then a validated creation and initial history enable sends', async () => {
  const fixture = widgetFixture();
  assert.equal(fixture.ui.send.disabled, true);
  assert.match(fixture.ui.status.textContent, /Waiting|^$/u);
  fixture.init();
  assert.equal(fixture.ui.send.disabled, true);
  assert.match(fixture.ui.status.textContent, /Connecting/u);
  await settle();
  assert.equal(fixture.ui.send.disabled, false);
  assert.match(fixture.ui.status.textContent, /Ready/u);
  assert.equal(fixture.requests.length, 2);
});

test('Send click validates and sends directly without a native form submit event', async () => {
  const fixture = widgetFixture([json(creation, 201), emptyHistory(), json(turn())]);
  fixture.init(); await settle();
  fixture.ui.question.value = '  ';
  fixture.ui.send.dispatch('click'); await settle();
  assert.equal(fixture.requests.length, 2);
  assert.match(fixture.ui.error.textContent, /Enter a question/u);
  fixture.ui.question.value = '  How do I start?  ';
  fixture.ui.send.dispatch('click');
  fixture.ui.send.dispatch('click');
  fixture.ui.question.dispatch('keydown', { key: 'Enter', ctrlKey: true });
  assert.equal(fixture.requests.length, 3);
  assert.equal(fixture.ui.send.disabled, true);
  await settle();
  assert.deepEqual(JSON.parse(fixture.requests[2].init.body), { request_id: id, text: 'How do I start?' });
  assert.equal(fixture.ui.messages.children.length, 2);
});

for (const modifier of ['ctrlKey', 'metaKey']) {
  test(`Textarea ${modifier}+Enter uses the shared validated send path exactly once`, async () => {
    const fixture = widgetFixture([json(creation, 201), emptyHistory(), json(turn())]);
    fixture.init(); await settle();
    let prevented = 0;
    const event = { key: 'Enter', [modifier]: true, preventDefault() { prevented++; } };
    fixture.ui.question.value = '\u0000';
    fixture.ui.question.dispatch('keydown', event); await settle();
    assert.equal(fixture.requests.length, 2);
    assert.match(fixture.ui.error.textContent, /Enter a question/u);
    fixture.ui.question.value = 'How do I start?';
    fixture.ui.question.dispatch('keydown', event);
    fixture.ui.question.dispatch('keydown', event);
    fixture.ui.send.dispatch('click');
    await settle();
    assert.equal(prevented, 3);
    assert.equal(fixture.requests.length, 3);
    assert.equal(fixture.ui.messages.children.length, 2);
  });
}

test('Textarea Enter preserves newlines and ignores composition, repeats and unrelated shortcuts', async () => {
  const fixture = widgetFixture();
  fixture.init(); await settle();
  fixture.ui.question.value = 'How do I start?';
  let prevented = 0;
  for (const event of [
    { key: 'Enter' }, { key: 'Enter', shiftKey: true }, { key: 'Escape', ctrlKey: true },
    { key: 'Enter', ctrlKey: true, isComposing: true },
    { key: 'Enter', metaKey: true, altKey: true }, { key: 'Enter', ctrlKey: true, shiftKey: true },
  ]) fixture.ui.question.dispatch('keydown', { ...event, preventDefault() { prevented++; } });
  assert.equal(prevented, 0);
  fixture.ui.question.dispatch('keydown', { key: 'Enter', ctrlKey: true, repeat: true, preventDefault() { prevented++; } });
  assert.equal(prevented, 1);
  assert.equal(fixture.requests.length, 2);
});

test('AI answer and dynamic source titles render as text, never HTML', async () => {
  const text = '<img src=x onerror=alert(1)>';
  const response = turn(id, text);
  response.messages[1].text = '<script>unsafe()</script>';
  response.messages[1].sources[0].title = '<b>Guide</b>';
  const fixture = widgetFixture([json(creation, 201), emptyHistory(), json(response)]);
  fixture.init(); await settle(); fixture.submit(text);
  assert.equal(fixture.ui.send.disabled, true);
  assert.match(fixture.ui.status.textContent, /Sending/u);
  await settle();
  assert.equal(fixture.ui.messages.children.length, 2);
  assert.match(fixture.ui.messages.textContent, /<script>unsafe\(\)<\/script>/u);
  const nodes = walk(fixture.ui.messages);
  assert.equal(nodes.some((node) => ['SCRIPT', 'IMG', 'B'].includes(node.tagName)), false);
  const link = nodes.find((node) => node.tagName === 'A');
  assert.equal(link.textContent, '<b>Guide</b>');
  assert.equal(link.href, 'https://example.org/help/start');
  assert.equal(link.rel, 'noopener noreferrer');
  assert.equal(link.target, undefined);
  assert.match(fixture.ui.messages.textContent, /Source links replace this frame/u);
  assert.equal(fixture.ui.send.disabled, false);
});

test('clarification is visibly not an AI answer and has no invented sources', async () => {
  const fixture = widgetFixture([json(creation, 201), emptyHistory(), json(turn(id, 'How do I start?', 1, true))]);
  fixture.init(); await settle(); fixture.submit(); await settle();
  assert.match(fixture.ui.messages.textContent, /Clarification · limited knowledge/u);
  assert.doesNotMatch(fixture.ui.messages.textContent, /AI answer/u);
  assert.equal(walk(fixture.ui.messages).some((node) => node.tagName === 'A'), false);
});

test('double submit is locked and a rejected text does not allocate a request', async () => {
  let finish;
  const deferred = new Promise((resolve) => { finish = resolve; });
  const fixture = widgetFixture([json(creation, 201), emptyHistory(), () => deferred]);
  fixture.init(); await settle();
  fixture.submit('x\u0000'); await settle();
  assert.equal(fixture.requests.length, 2);
  fixture.submit(); fixture.submit('A second question');
  assert.equal(fixture.requests.length, 3);
  finish(json(turn())); await settle();
  assert.equal(fixture.ui.messages.children.length, 2);
});

test('only a genuinely new submission allocates a new cryptographic UUID', async () => {
  const nextID = '87654321-4321-4321-8321-123456789abc';
  const fixture = widgetFixture([json(creation, 201), emptyHistory(), new Error('lost'), emptyHistory(), json(turn()), json(turn(nextID, 'Second question', 2))]);
  let allocated = 0;
  fixture.platform.crypto.randomUUID = () => { allocated++; return allocated === 1 ? id : nextID; };
  fixture.init(); await settle();
  fixture.submit(); await settle();
  fixture.ui.retry.dispatch('click'); await settle();
  assert.equal(allocated, 1);
  fixture.submit('Second question'); await settle();
  assert.equal(allocated, 2);
  assert.equal(JSON.parse(fixture.requests.at(-1).init.body).request_id, nextID);
  assert.equal(fixture.ui.messages.children.length, 4);
});

test('ambiguous transport first reads history and recovers a completed turn with no repost', async () => {
  const fixture = widgetFixture([json(creation, 201), emptyHistory(), new Error('lost response'), json({ messages: turn().messages, next_before: null })]);
  fixture.init(); await settle(); fixture.submit(); await settle();
  assert.deepEqual(fixture.requests.map(({ init }) => init.method), ['POST', 'GET', 'POST', 'GET']);
  assert.equal(fixture.ui.messages.children.length, 2);
  assert.match(fixture.ui.status.textContent, /recovered/u);
  assert.equal(fixture.ui.retry.hidden, true);
  assert.equal(fixture.ui.send.disabled, false);
});

test('uncommitted transport failure retries identical ID and trimmed text without duplicate bubbles', async () => {
  const fixture = widgetFixture([json(creation, 201), emptyHistory(), new Error('lost'), emptyHistory(), json(turn())]);
  fixture.init(); await settle(); fixture.submit('  How do I start?  '); await settle();
  assert.equal(fixture.ui.send.disabled, true);
  assert.equal(fixture.ui.messages.children.length, 1);
  assert.equal(fixture.requests.length, 4);
  fixture.ui.retry.dispatch('click'); await settle();
  assert.equal(fixture.requests[2].init.body, fixture.requests[4].init.body);
  assert.deepEqual(JSON.parse(fixture.requests[4].init.body), { request_id: id, text: 'How do I start?' });
  assert.equal(fixture.ui.messages.children.length, 2);
  assert.equal(fixture.ui.send.disabled, false);
});

test('uncertain delivery plus failed history blocks new sends until history is checked', async () => {
  const fixture = widgetFixture([json(creation, 201), emptyHistory(), new Error('lost'), new Error('history down'), emptyHistory(), json(turn())]);
  fixture.init(); await settle(); fixture.submit(); await settle();
  assert.match(fixture.ui.error.textContent, /uncertain/u);
  assert.equal(fixture.ui.send.disabled, true);
  fixture.submit('new question'); await settle();
  assert.equal(fixture.requests.length, 4);
  fixture.ui.retry.dispatch('click'); await settle();
  assert.equal(fixture.requests[4].init.method, 'GET');
  fixture.ui.retry.dispatch('click'); await settle();
  assert.equal(fixture.requests[5].init.body, fixture.requests[2].init.body);
});

test('provider 503 is unavailable, not fake success, and raw server message is ignored', async () => {
  const fixture = widgetFixture([json(creation, 201), emptyHistory(), fail('provider_unavailable', 503, '<script>secret key</script>'), emptyHistory()]);
  fixture.init(); await settle(); fixture.submit(); await settle();
  assert.match(fixture.ui.error.textContent, /AI service is unavailable/u);
  assert.doesNotMatch(fixture.document.body.textContent, /secret key/u);
  assert.equal(fixture.ui.messages.children.length, 1);
  assert.equal(fixture.ui.send.disabled, true);
  assert.equal(fixture.requests.filter(({ init }) => init.method === 'POST').length, 2);
});

test('busy response reconciles GET and enforces bounded cooldown with no automatic paid retry', async () => {
  const fixture = widgetFixture([json(creation, 201), emptyHistory(), fail('conversation_busy', 409, 'ignored', { 'Retry-After': '999999' }), emptyHistory()]);
  fixture.init(); await settle(); fixture.submit(); await settle();
  assert.match(fixture.ui.error.textContent, /busy/u);
  assert.equal(fixture.ui.retry.disabled, true);
  assert.equal([...fixture.timers.values()].some((timer) => timer.delay === 120000), true);
  fixture.ui.retry.dispatch('click'); await settle();
  assert.equal(fixture.requests.length, 4);
});

test('429 holds same question and does not automatically retry or fetch new history', async () => {
  const fixture = widgetFixture([json(creation, 201), emptyHistory(), fail('rate_limited', 429, 'ignored', { 'Retry-After': '60' })]);
  fixture.init(); await settle(); fixture.submit(); await settle();
  assert.match(fixture.ui.error.textContent, /Too many requests/u);
  assert.equal(fixture.ui.retry.disabled, true);
  assert.equal(fixture.requests.length, 3);
  assert.equal([...fixture.timers.values()].some((timer) => timer.delay === 60000), true);
});

test('invalid response source fails before any answer or link enters DOM', async () => {
  const response = turn(); response.messages[1].sources[0].url = 'javascript:alert(1)';
  const fixture = widgetFixture([json(creation, 201), emptyHistory(), json(response), emptyHistory()]);
  fixture.init(); await settle(); fixture.submit(); await settle();
  assert.equal(fixture.ui.messages.children.length, 1);
  assert.equal(walk(fixture.ui.messages).some((node) => node.tagName === 'A'), false);
  assert.equal(fixture.ui.send.disabled, true);
});

test('initial history failure blocks sending and provides a history-only retry', async () => {
  const fixture = widgetFixture([json(creation, 201), new Error('down'), emptyHistory()]);
  fixture.init(); await settle();
  assert.equal(fixture.ui.send.disabled, true);
  fixture.submit(); assert.equal(fixture.requests.length, 2);
  fixture.ui.retry.dispatch('click'); await settle();
  assert.equal(fixture.requests[2].init.method, 'GET');
  assert.equal(fixture.ui.send.disabled, false);
});

test('history is ascending with load-older pagination and no duplicate rendering', async () => {
  const newest = turn('second_id', 'Second question', 2).messages;
  const oldest = turn().messages;
  const fixture = widgetFixture([json(creation, 201), json({ messages: newest, next_before: 3 }), json({ messages: oldest, next_before: null })]);
  fixture.init(); await settle();
  assert.equal(fixture.ui.older.hidden, false);
  fixture.ui.older.dispatch('click'); await settle();
  assert.match(fixture.requests[2].url, /\?limit=20&before=3$/u);
  assert.equal(fixture.ui.messages.children.length, 4);
  assert.match(fixture.ui.messages.children[0].textContent, /How do I start/u);
  assert.match(fixture.ui.messages.children[2].textContent, /Second question/u);
  assert.equal(fixture.ui.older.hidden, true);
});

test('creation retry is explicit and starts fresh without storing a capability externally', async () => {
  const fixture = widgetFixture([new Error('lost creation'), json(creation, 201), emptyHistory()]);
  fixture.init(); await settle();
  assert.equal(fixture.requests.length, 1);
  assert.match(fixture.ui.error.textContent, /new guest chat/u);
  fixture.ui.retry.dispatch('click'); await settle();
  assert.equal(fixture.requests[1].init.method, 'POST');
  assert.equal(fixture.requests[1].init.headers.Authorization, undefined);
  assert.equal(fixture.ui.send.disabled, false);
});

test('authorization failure and conversation-full are terminal safe failures', async () => {
  for (const [code, http, expected] of [['unauthorized', 401, /no longer be accessed/u], ['conversation_full', 409, /50-question limit/u], ['idempotency_conflict', 409, /safely retried/u]]) {
    const fixture = widgetFixture([json(creation, 201), emptyHistory(), fail(code, http)]);
    fixture.init(); await settle(); fixture.submit(); await settle();
    assert.match(fixture.ui.error.textContent, expected);
    assert.equal(fixture.ui.send.disabled, true);
    assert.equal(fixture.ui.retry.hidden, true);
    assert.equal(fixture.requests.length, 3);
  }
});

test('bounded timeout aborts an in-flight POST and triggers history reconciliation only', async () => {
  const fixture = widgetFixture([json(creation, 201), emptyHistory(), (_url, init) => new Promise((_resolve, reject) => {
    init.signal.addEventListener('abort', () => reject(new Error('timeout')));
  }), emptyHistory()]);
  fixture.init(); await settle(); fixture.submit();
  const timeout = [...fixture.timers.values()].find((timer) => timer.delay === 25000);
  assert.ok(timeout);
  timeout.fn(); await settle();
  assert.equal(fixture.requests[2].init.signal.aborted, true);
  assert.equal(fixture.requests[3].init.method, 'GET');
  assert.equal(fixture.ui.send.disabled, true);
});

test('pagehide aborts the request and clears private access without further fetches', async () => {
  const fixture = widgetFixture([json(creation, 201), emptyHistory(), (_url, init) => new Promise((_resolve, reject) => {
    init.signal.addEventListener('abort', () => reject(new Error('aborted')));
  })]);
  fixture.init(); await settle(); fixture.submit();
  fixture.emit('pagehide', {}); await settle();
  assert.equal(fixture.requests[2].init.signal.aborted, true);
  assert.equal(fixture.requests.length, 3);
  assert.equal(fixture.timers.size, 0);
});

test('loader invalid context is visible and does not create iframe or messages', async () => {
  const fixture = await loaderFixture({ 'data-page-path': '/help?secret=1' });
  assert.equal(fixture.launcher, undefined);
  assert.equal(fixture.frame(), undefined);
  assert.match(fixture.document.body.textContent, /could not start/u);
});

test('loader toggle reuses one iframe, accessible control and conversation handshake', async () => {
  const fixture = await loaderFixture();
  assert.equal(fixture.launcher.getAttribute('aria-expanded'), 'false');
  fixture.open();
  const frame = fixture.frame();
  assert.equal(frame.hidden, false);
  assert.equal(fixture.launcher.getAttribute('aria-expanded'), 'true');
  assert.equal(frame.title, 'AI support assistant chat');
  assert.equal(frame.getAttribute('sandbox'), 'allow-scripts allow-same-origin');
  fixture.open(); fixture.open();
  assert.equal(fixture.frame(), frame);
  assert.equal(walk(fixture.document.body).filter((node) => node.tagName === 'IFRAME').length, 1);
});

test('bounded property sweep: arbitrary malformed boundary values never fetch or crash', async () => {
  // Deterministic regression/property corpus; no claim of a browser or native fuzzer.
  let seed = 14641;
  for (let index = 0; index < 128; index++) {
    seed = (Math.imul(seed, 1664525) + 1013904223) >>> 0;
    const value = String.fromCodePoint(seed % 0xD7FF) + `/${seed}?x`;
    assert.doesNotThrow(() => {
      validContext({ page_path: value }); validSource({ id: value, title: value, url: value }); validHistory({ messages: [value], next_before: value });
    });
    const fixture = widgetFixture([], { location: { hash: `#parent_origin=${encodeURIComponent(value)}` } });
    fixture.init();
    assert.equal(fixture.requests.length, 0);
    assert.equal(fixture.ui.error.hidden, false);
  }
});
