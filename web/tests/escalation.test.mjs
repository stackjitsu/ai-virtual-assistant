import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { canonicalEmail, validEscalationEmail, validEscalationNote, validEscalationTicket } from '../assets/protocol.mjs';
import { creation, emptyHistory, fail, json, settle, turn, walk, widgetFixture } from './harness.mjs';

// The only truthful visitor sentence this slice's outcome supports.
const TRUTH = 'Your request has been recorded. Email delivery is not configured, so no email has been sent.';
const ticket = (over = {}) => json({ ticket_id: 'c'.repeat(32), recorded_at: '2026-10-20T12:00:00Z',
  delivery_outcome: 'delivery_not_wired', message: TRUTH, ...over }, 201);
const ready = (responses = [], options = {}) => {
  const fixture = widgetFixture([json(creation, 201), emptyHistory(), ...responses], options);
  fixture.init();
  return fixture;
};
const openForm = async (fixture, { email = 'Visitor@Example.COM', note = 'Email me at that address, please', consent = true } = {}) => {
  fixture.ui['escalate-toggle'].dispatch('click');
  fixture.ui['handoff-email'].dispatch('click');
  fixture.ui['escalate-email'].value = email;
  fixture.ui['escalate-note'].value = note;
  fixture.ui['escalate-consent'].checked = consent;
  return fixture;
};
const send = async (fixture) => { fixture.ui['escalate-send'].dispatch('click'); await settle(); };
const escalationRequests = (fixture) => fixture.requests.filter((request) => request.url.includes('/escalation'));

test('handoff affordance appears only with a live conversation and opens a compact channel menu', async () => {
  const fixture = widgetFixture();
  assert.equal(fixture.ui['escalate-toggle'].hidden, true);
  fixture.init(); await settle();
  assert.equal(fixture.ui['escalate-toggle'].hidden, false);
  assert.equal(fixture.ui['escalate-toggle'].disabled, false);
  assert.equal(fixture.ui['handoff-menu'].hidden, true);
  assert.equal(fixture.ui['escalation-form'].hidden, true);
  assert.equal(fixture.ui['escalate-toggle'].getAttribute('aria-expanded'), 'false');
  fixture.ui['escalate-toggle'].dispatch('click');
  assert.equal(fixture.ui['handoff-menu'].hidden, false);
  assert.equal(fixture.ui['escalate-toggle'].getAttribute('aria-expanded'), 'true');
  // By email is the only available channel; By WhatsApp is disabled.
  assert.equal(fixture.ui['handoff-email'].disabled, false);
  assert.equal(fixture.ui['handoff-whatsapp'].disabled, true);
  fixture.ui['handoff-email'].dispatch('click');
  assert.equal(fixture.ui['escalation-form'].hidden, false);
  assert.equal(fixture.ui['handoff-email'].getAttribute('aria-pressed'), 'true');
  fixture.ui['handoff-email'].dispatch('click');
  assert.equal(fixture.ui['escalation-form'].hidden, true);
  assert.equal(fixture.ui['handoff-email'].getAttribute('aria-pressed'), 'false');
  fixture.ui['escalate-toggle'].dispatch('click');
  assert.equal(fixture.ui['handoff-menu'].hidden, true);
  assert.equal(fixture.ui['escalation-form'].hidden, true);
  assert.equal(fixture.ui['escalate-toggle'].getAttribute('aria-expanded'), 'false');
  const closed = widgetFixture([], { location: { hash: '#parent_origin=https%3A%2F%2Fwrong.example' } });
  closed.init(); await settle();
  assert.equal(closed.ui['escalate-toggle'].hidden, true);
});

test('By WhatsApp is visibly unavailable, disabled, and collects no phone number', async () => {
  const fixture = ready(); await settle();
  fixture.ui['escalate-toggle'].dispatch('click');
  const whatsapp = fixture.ui['handoff-whatsapp'];
  assert.equal(whatsapp.hidden, false);
  assert.equal(whatsapp.disabled, true);
  whatsapp.dispatch('click'); await settle();
  assert.equal(escalationRequests(fixture).length, 0);
  assert.equal(fixture.ui['escalation-form'].hidden, true);
  // The unavailable channel is never exposed as the selected one.
  assert.equal(whatsapp.getAttribute('aria-pressed'), 'false');
  assert.equal(fixture.ui['handoff-email'].getAttribute('aria-pressed'), 'false');
  assert.equal(fixture.ui['handoff-menu'].hidden, false);
  // No phone collection exists anywhere in the widget markup.
  const html = await readFile(new URL('../assets/widget.html', import.meta.url), 'utf8');
  assert.doesNotMatch(html, /type="tel"|inputmode="tel"|name="phone"|name="whatsapp"|wa\.me|api\.whatsapp/iu);
  assert.equal(fixture.ui['escalate-email'].value, '');
});

test('the handoff menu opens only after a conversation exists and no server event opens it', async () => {
  const fixture = widgetFixture([json(creation, 201), emptyHistory(), json(turn())]);
  assert.equal(fixture.ui['escalate-toggle'].hidden, true);
  assert.equal(fixture.ui['handoff-menu'].hidden, true);
  fixture.init(); await settle();
  // Readiness is a server event; the visitor still has to ask for the menu.
  assert.equal(fixture.ui['handoff-menu'].hidden, true);
  assert.equal(fixture.ui['escalate-toggle'].getAttribute('aria-expanded'), 'false');
  // A completed answer is a server event too and leaves the menu closed.
  fixture.submit(); await settle();
  assert.equal(fixture.ui['messages'].children.length, 2);
  assert.equal(fixture.ui['handoff-menu'].hidden, true);
  assert.equal(fixture.ui['escalation-form'].hidden, true);
});

test('Escape requests window close even with handoff form open; other keys leave it open', async () => {
  const fixture = ready(); await settle();
  fixture.ui['escalate-toggle'].dispatch('click');
  fixture.ui['handoff-email'].dispatch('click');
  assert.equal(fixture.ui['handoff-menu'].hidden, false);
  assert.equal(fixture.ui['escalation-form'].hidden, false);
  assert.equal(fixture.ui['handoff-email'].focused, true);
  // A typed letter must never dismiss the menu.
  fixture.emit('keydown', { key: 'q', target: {}, preventDefault() {} });
  assert.equal(fixture.ui['handoff-menu'].hidden, false);
  let prevented = false;
  fixture.ui['escalate-email'].value = 'visitor@example.org';
  fixture.emit('keydown', { key: 'Escape', target: fixture.ui['escalate-email'], preventDefault() { prevented = true; } });
  assert.equal(prevented, true);
  assert.equal(fixture.ui['handoff-menu'].hidden, true);
  assert.equal(fixture.ui['escalation-form'].hidden, true);
  assert.equal(fixture.ui['escalate-toggle'].getAttribute('aria-expanded'), 'false');
  assert.equal(fixture.ui['escalate-toggle'].focused, undefined);
  assert.deepEqual(fixture.sent.at(-1), { data: { type: 'assistant:close', version: 1 }, origin: 'https://host.example' });
  assert.equal(fixture.ui['escalate-email'].value, 'visitor@example.org');
});

test('an outside click closes the handoff menu while a click inside the handoff region does not', async () => {
  const fixture = ready(); await settle();
  fixture.ui['escalate-toggle'].dispatch('click');
  assert.equal(fixture.ui['handoff-menu'].hidden, false);
  // The trigger belongs to the handoff region, so its click does not close it.
  fixture.ui['escalate-toggle'].selectors = ['.escalation'];
  fixture.emit('click', { target: fixture.ui['escalate-toggle'] });
  assert.equal(fixture.ui['handoff-menu'].hidden, false);
  // An unrelated click closes the whole disclosure.
  fixture.emit('click', { target: fixture.ui['status'] });
  assert.equal(fixture.ui['handoff-menu'].hidden, true);
  assert.equal(fixture.ui['escalation-form'].hidden, true);
});

test('an open handoff menu never disables or mutates the chat send path', async () => {
  const fixture = ready([json(turn())]); await settle();
  fixture.ui['escalate-toggle'].dispatch('click');
  fixture.ui['handoff-email'].dispatch('click');
  assert.equal(fixture.ui['handoff-menu'].hidden, false);
  assert.equal(fixture.ui['escalation-form'].hidden, false);
  assert.equal(fixture.ui.send.disabled, false);
  fixture.submit(); await settle();
  assert.equal(fixture.ui['messages'].children.length, 2);
  assert.equal(fixture.ui['handoff-menu'].hidden, false);
  assert.equal(fixture.ui['escalation-form'].hidden, false);
  assert.equal(escalationRequests(fixture).length, 0);
  // A chat interaction closes the disclosure without discarding what the
  // visitor already typed for human support.
  fixture.ui['escalate-email'].value = 'visitor@example.org';
  fixture.emit('click', { target: fixture.ui['question'] });
  assert.equal(fixture.ui['handoff-menu'].hidden, true);
  assert.equal(fixture.ui['escalation-form'].hidden, true);
  assert.equal(fixture.ui['escalate-email'].value, 'visitor@example.org');
  fixture.ui['escalate-toggle'].dispatch('click');
  fixture.ui['handoff-email'].dispatch('click');
  assert.equal(fixture.ui['escalation-form'].hidden, false);
  assert.equal(fixture.ui['escalate-email'].value, 'visitor@example.org');
});

test('handoff menu closes on Escape and outside click, and never auto-opens on a provider failure', async () => {
  const fixture = ready([fail('provider_unavailable', 503, 'DO NOT RENDER'), emptyHistory()]); await settle();
  fixture.ui['escalate-toggle'].dispatch('click');
  fixture.ui['handoff-email'].dispatch('click');
  assert.equal(fixture.ui['handoff-menu'].hidden, false);
  assert.equal(fixture.ui['escalation-form'].hidden, false);
  fixture.emit('keydown', { key: 'Escape', target: {}, preventDefault() {} });
  assert.equal(fixture.ui['handoff-menu'].hidden, true);
  assert.equal(fixture.ui['escalation-form'].hidden, true);
  assert.equal(fixture.sent.at(-1).data.type, 'assistant:close');
  // An outside click closes the disclosure too.
  fixture.ui['escalate-toggle'].dispatch('click');
  assert.equal(fixture.ui['handoff-menu'].hidden, false);
  fixture.emit('click', { target: {} });
  assert.equal(fixture.ui['handoff-menu'].hidden, true);
  assert.equal(fixture.ui['escalation-form'].hidden, true);
  // A provider failure never opens the menu or form.
  fixture.submit(); await settle();
  assert.match(fixture.ui.error.textContent, /AI service is unavailable/u);
  assert.equal(fixture.ui['handoff-menu'].hidden, true);
  assert.equal(fixture.ui['escalation-form'].hidden, true);
});

test('email mirror validation equals the server canonical rules', () => {
  for (const [raw, valid] of [
    ['Visitor@Example.COM', true], ['  visitor@example.com  ', true], ['a.b%c+d@example.co.uk', true],
    ['A@EXAMPLE.ORG', true], [`${'a'.repeat(64)}@${'b'.repeat(60)}.example.com`, true],
    ['a@admin.example.museum', true], ['x@example.a1.com', true], ['\t visitor@example.com \r', true],
    ['', false], ['plain', false], ['a@', false], ['@example.com', false], ['a@@example.com', false],
    ['a@example.com@b.com', false], ['a b@example.com', false], ['a@exam ple.com', false], ['a@b', false],
    ['a@b.c', false], ['a@example.123', false], ['a@example.c0m', false], ['a@example.c', false],
    ['a@-example.com', false], ['a@example-.com', false], ['a@example..com', false], ['a@.example.com', false],
    ['a@exam_ple.com', false], ['niké@example.com', false], ['a@Ｅxample.com', false], ['"quoted"@example.com', false],
    ['user@example.com.', false], ['a_b@[IPv6:::1]', false], [`a@${'b'.repeat(64)}.com`, false],
    [`${'a'.repeat(65)}@example.com`, false], ['a@example.com\n', true],
    [`a@${'b'.repeat(150)}.${'c'.repeat(96)}.com`, false], ['a'.repeat(250) + '@example.com', false],
    [`${'a'.repeat(64)}@${'b'.repeat(150)}.${'c'.repeat(96)}.${'d'.repeat(4)}`, false],
  ]) assert.equal(validEscalationEmail(raw), valid, JSON.stringify(raw));
  assert.equal(canonicalEmail('Visitor@Example.COM  '), 'visitor@example.com');
});

test('note mirror validation trims once and enforces code points and bytes', () => {
  for (const [note, valid] of [
    ['', true], ['   ', true], ['Email me, please.', true], ['line\nbreak\ttab\rcarriage', true],
    ['a'.repeat(1000), true], ['😀'.repeat(500), true],
    ['a'.repeat(1001), false], ['😀'.repeat(501), false], ['a'.repeat(2000), false], ['a'.repeat(2001), false],
    ['😀'.repeat(1000), false],
    ['x\u0000', false], ['x\u000b', true], ['x\u000b y', false], ['x\u007f', false], ['x\u009f', false],
    ['\ud800', false], ['bre\raum between\u0000', false],
  ]) assert.equal(validEscalationNote(note), valid, JSON.stringify(note.slice(0, 24)));
});

test('ticket envelope validation accepts only the honest slice-2 shape', () => {
  assert.equal(validEscalationTicket({ ticket_id: 'c'.repeat(32), recorded_at: '2026-10-20T12:00:00.123Z',
    delivery_outcome: 'delivery_not_wired', message: TRUTH }), true);
  for (const wrong of [
    null, {}, [],
    { recorded_at: '2026-10-20T12:00:00Z', delivery_outcome: 'delivery_not_wired', message: TRUTH },
    { ticket_id: 'c'.repeat(16), recorded_at: '2026-10-20T12:00:00Z', delivery_outcome: 'delivery_not_wired', message: TRUTH },
    { ticket_id: 'c'.repeat(32), delivery_outcome: 'delivery_not_wired', message: TRUTH },
    { ticket_id: 'c'.repeat(32), recorded_at: '2026-10-20 12:00:00', delivery_outcome: 'delivery_not_wired', message: TRUTH },
    { ticket_id: 'c'.repeat(32), recorded_at: '2026-10-20T12:00:00Z', delivery_outcome: 'email_sent', message: TRUTH },
    { ticket_id: 'c'.repeat(32), recorded_at: '2026-10-20T12:00:00Z', delivery_outcome: 'delivery_not_wired', message: '' },
    { ticket_id: 'c'.repeat(32), recorded_at: '2026-10-20T12:00:00Z', delivery_outcome: 'delivery_not_wired', message: 5 },
    { ticket_id: 'c'.repeat(32), recorded_at: '2026-10-20T12:00:00Z', delivery_outcome: 'delivery_not_wired', message: TRUTH, extra: true },
  ]) assert.equal(validEscalationTicket(wrong), false);
});

test('consent gate fails closed and only a literally checked box sends one validated POST', async () => {
  const fixture = ready([ticket(), ticket()]); await settle();
  await openForm(fixture, { consent: false });
  await send(fixture);
  assert.equal(escalationRequests(fixture).length, 0);
  assert.match(fixture.ui['escalate-error'].textContent, /consent/iu);
  assert.equal(fixture.ui['escalate-outcome'].hidden, true);
  for (const notConsent of ['true', 1, { checked: true }]) {
    fixture.ui['escalate-consent'].checked = notConsent;
    await send(fixture);
    assert.equal(escalationRequests(fixture).length, 0);
  }
  assert.match(fixture.ui['escalate-error'].textContent, /consent/iu);
  fixture.ui['escalate-consent'].checked = true;
  await send(fixture);
  assert.equal(escalationRequests(fixture).length, 1);
  const body = escalationRequests(fixture)[0].init.body;
  assert.equal(JSON.parse(body).consent, true);
  assert.doesNotMatch(body, /consent["':\s]+(false|"true"|1)/iu);
});

test('client bounds block guaranteed-invalid requests before any fetch, values keep visitor case', async () => {
  const fixture = ready([ticket(), ticket()]); await settle();
  await openForm(fixture, { email: 'a b@example.com' });
  await send(fixture);
  assert.equal(escalationRequests(fixture).length, 0);
  assert.match(fixture.ui['escalate-error'].textContent, /valid email address/u);
  fixture.ui['escalate-email'].value = 'you@example.org';
  fixture.ui['escalate-note'].value = 'x'.repeat(1001);
  await send(fixture);
  assert.equal(escalationRequests(fixture).length, 0);
  assert.match(fixture.ui['escalate-error'].textContent, /1,000 characters/u);
  fixture.ui['escalate-note'].value = '  Detail for the support team  ';
  await send(fixture);
  assert.equal(escalationRequests(fixture).length, 1);
  assert.deepEqual(JSON.parse(escalationRequests(fixture)[0].init.body), {
    email: 'you@example.org', question_note: '  Detail for the support team  ', consent: true,
  });
});

test('a recorded request renders the server message verbatim with a visitor-sane time and keeps chat intact', async () => {
  const fixture = ready([ticket()]); await settle();
  await openForm(fixture);
  await send(fixture);
  const request = escalationRequests(fixture)[0];
  assert.match(request.url, new RegExp(`/api/v1/conversations/${creation.conversation_id}/escalation$`, 'u'));
  assert.equal(new URL(request.url).origin, fixture.platform.location.origin);
  assert.doesNotMatch(request.url, new RegExp(creation.capability, 'u'));
  assert.equal(request.init.headers.Authorization, `Bearer ${creation.capability}`);
  assert.equal(request.init.credentials, 'omit');
  assert.equal(request.init.mode, 'same-origin');
  assert.equal(request.init.cache, 'no-store');
  assert.equal(request.init.redirect, 'error');
  assert.deepEqual(JSON.parse(request.init.body), {
    email: 'Visitor@Example.COM', question_note: 'Email me at that address, please', consent: true,
  });
  const outcome = fixture.ui['escalate-outcome'];
  assert.equal(outcome.hidden, false);
  assert.ok(outcome.textContent.startsWith(TRUTH), outcome.textContent);
  assert.ok(outcome.textContent.includes(`Recorded ${new Date('2026-10-20T12:00:00Z').toUTCString()}.`), outcome.textContent);
  assert.ok(outcome.textContent.includes(`Request reference: ${'c'.repeat(32)}`));
  assert.equal(fixture.ui['escalate-error'].hidden, true);
  // The chat send path, transcript and disclosures are untouched.
  assert.match(fixture.ui.status.textContent, /Ready/u);
  assert.equal(fixture.ui.send.disabled, false);
  assert.equal(fixture.ui.messages.children.length, 0);
  assert.match(fixture.ui.transmission.textContent, /Questions and recent chat go to the app server/u);
  assert.match(fixture.ui.transmission.textContent, /approved excerpts may go to the configured AI provider/u);
  assert.ok(!fixture.ui.transmission.textContent.includes(fixture.platform.location.origin));
});

test('every server value stays inert text without markup interpretation', async () => {
  const evil = ticket({ message: '<img src=x onerror=alert(1)><b>bold</b><script>x()</script>' });
  const fixture = ready([evil, fail('not_found', 404, '<iframe src="//evil"></iframe><img src=x>'), ticket()]);
  await settle();
  await openForm(fixture);
  await send(fixture);
  const outcome = fixture.ui['escalate-outcome'];
  assert.equal(outcome.hidden, false);
  assert.ok(outcome.textContent.includes('<img src=x onerror=alert(1)>'));
  assert.ok(outcome.textContent.includes('<script>x()</script>'));
  assert.equal(walk(outcome).some((node) => ['SCRIPT', 'IMG', 'B', 'IFRAME'].includes(node.tagName)), false);
  // A non-contract failure keeps raw server text out of the error display.
  await openForm(fixture, { email: 'again@example.com' });
  await send(fixture);
  assert.match(fixture.ui['escalate-error'].textContent, /could not be completed safely/u);
  assert.doesNotMatch(fixture.ui['escalate-error'].textContent, /<|evil|iframe/u);
  // A later success returns to the plain opaque inert reference text.
  await send(fixture);
  assert.ok(fixture.ui['escalate-outcome'].textContent.includes(`Request reference: ${'c'.repeat(32)}`));
  assert.equal(walk(fixture.ui['escalate-outcome']).some((node) => node.tagName !== 'P'), false);
});

test('a non-contract 201 claim (sent outcome or broken envelope) fails closed', async () => {
  for (const wrong of [ticket({ delivery_outcome: 'email_sent' }), ticket({ ticket_id: 'short' })]) {
    const fixture = ready([wrong, ticket()]);
    await settle();
    await openForm(fixture);
    await send(fixture);
    assert.match(fixture.ui['escalate-error'].textContent, /could not be completed safely/u);
    assert.equal(fixture.ui['escalate-outcome'].hidden, true);
    assert.doesNotMatch(fixture.document.body.textContent, /email_sent|email has been sent to/u);
    // The visitor can still reach the recorded ticket with an explicit retry.
    await send(fixture);
    assert.equal(fixture.ui['escalate-outcome'].hidden, false);
    assert.ok(fixture.ui['escalate-outcome'].textContent.startsWith(TRUTH));
    assert.equal(escalationRequests(fixture).length, 2);
  }
});

test('400 shows safe local wording with no automatic resend, then retries explicitly', async () => {
  const fixture = ready([fail('invalid_request', 400, '<script>smtp password</script>'), ticket()]);
  await settle();
  await openForm(fixture);
  await send(fixture);
  assert.match(fixture.ui['escalate-error'].textContent, /could not be validated/u);
  assert.match(fixture.ui['escalate-error'].textContent, /consent box/u);
  assert.doesNotMatch(fixture.document.body.textContent, /smtp password/u);
  assert.equal(escalationRequests(fixture).length, 1);
  await settle();
  assert.equal(escalationRequests(fixture).length, 1);
  assert.equal(fixture.ui['escalate-send'].disabled, false);
  await send(fixture);
  assert.equal(escalationRequests(fixture).length, 2);
  assert.ok(fixture.ui['escalate-outcome'].textContent.startsWith(TRUTH));
});

test('409 explains the 5-request bound and that a matching repeat replays the stored ticket', async () => {
  const fixture = ready([fail('conflict', 409, 'DO NOT RENDER'), ticket(), ticket()]);
  await settle();
  await openForm(fixture);
  await send(fixture);
  assert.match(fixture.ui['escalate-error'].textContent, /limit of 5 support requests/u);
  assert.match(fixture.ui['escalate-error'].textContent, /same email and note again shows your stored request/u);
  assert.equal(escalationRequests(fixture).length, 1);
  assert.equal(fixture.ui.send.disabled, false);
  await send(fixture);
  assert.equal(escalationRequests(fixture).length, 2);
  assert.ok(fixture.ui['escalate-outcome'].textContent.startsWith(TRUTH));
});

test('429 honors bounded Retry-After on its own cooldown without touching the chat send path', async () => {
  const fixture = ready([fail('rate_limited', 429, 'DO NOT RENDER', { 'Retry-After': '999999' }), ticket()]);
  await settle();
  await openForm(fixture);
  await send(fixture);
  assert.match(fixture.ui['escalate-error'].textContent, /Too many support requests/u);
  assert.equal(fixture.ui['escalate-send'].disabled, true);
  assert.equal([...fixture.timers.values()].some((timer) => timer.delay === 120000), true);
  assert.equal(fixture.ui.send.disabled, false);
  assert.equal(escalationRequests(fixture).length, 1);
  await settle();
  assert.equal(escalationRequests(fixture).length, 1);
  [...fixture.timers.values()].find((timer) => timer.delay === 120000).fn();
  await settle();
  assert.equal(fixture.ui['escalate-send'].disabled, false);
  await send(fixture);
  const attempts = escalationRequests(fixture);
  assert.equal(attempts.length, 2);
  assert.equal(attempts[0].init.body, attempts[1].init.body);
  assert.ok(fixture.ui['escalate-outcome'].textContent.startsWith(TRUTH));
});

test('503 offers an explicit retry only; transport loss never auto-resends', async () => {
  const fixture = ready([fail('storage_unavailable', 503, '<script>queue</script>'), ticket()]);
  await settle();
  await openForm(fixture);
  await send(fixture);
  assert.match(fixture.ui['escalate-error'].textContent, /temporarily unavailable/u);
  assert.match(fixture.ui['escalate-error'].textContent, /Send request/u);
  assert.doesNotMatch(fixture.document.body.textContent, /queue/u);
  assert.equal(fixture.ui['escalate-outcome'].hidden, true);
  assert.equal(fixture.ui['escalate-send'].disabled, false);
  assert.equal(escalationRequests(fixture).length, 1);
  await settle();
  // No automatic retries of any kind; the next POST is an explicit click.
  assert.equal(escalationRequests(fixture).length, 1);
  await send(fixture);
  const attempts = escalationRequests(fixture);
  assert.equal(attempts.length, 2);
  assert.equal(attempts[0].init.body, attempts[1].init.body);
  assert.ok(fixture.ui['escalate-outcome'].textContent.startsWith(TRUTH));
  const lost = ready([new Error('lost'), ticket()]);
  await settle();
  await openForm(lost);
  await send(lost);
  assert.match(lost.ui['escalate-error'].textContent, /could not be reached/u);
  assert.equal(escalationRequests(lost).length, 1);
  await settle();
  assert.equal(escalationRequests(lost).length, 1);
  await send(lost);
  assert.equal(JSON.parse(escalationRequests(lost)[0].init.body).email, 'Visitor@Example.COM');
  assert.ok(lost.ui['escalate-outcome'].textContent.startsWith(TRUTH));
});

test('401 follows the existing terminal handling for the whole widget', async () => {
  const fixture = ready([fail('unauthorized', 401, 'DO NOT RENDER'), ticket()]);
  await settle();
  await openForm(fixture);
  await send(fixture);
  assert.match(fixture.ui.error.textContent, /no longer be accessed/u);
  assert.match(fixture.ui.status.textContent, /Chat unavailable/u);
  assert.equal(fixture.ui.send.disabled, true);
  assert.equal(fixture.ui['escalate-toggle'].hidden, true);
  assert.equal(fixture.ui['handoff-menu'].hidden, true);
  assert.equal(fixture.ui['escalation-form'].hidden, true);
  assert.equal(fixture.ui['escalate-send'].disabled, true);
  assert.equal(escalationRequests(fixture).length, 1);
  await send(fixture);
  assert.equal(escalationRequests(fixture).length, 1);
});

test('resubmitting a stored pair replays the stored ticket as a feature, not an error', async () => {
  const fixture = ready([ticket(), ticket()]);
  await settle();
  await openForm(fixture);
  await send(fixture);
  assert.ok(fixture.ui['escalate-outcome'].textContent.startsWith(TRUTH));
  assert.equal(fixture.ui['escalate-outcome'].textContent.includes('replayed unchanged'), false);
  await send(fixture);
  assert.equal(escalationRequests(fixture).length, 2);
  const outcome = fixture.ui['escalate-outcome'];
  assert.ok(outcome.textContent.startsWith(TRUTH));
  assert.ok(outcome.textContent.includes('previously stored request, replayed unchanged'));
  assert.equal(fixture.ui['escalate-error'].hidden, true);
  // The chat wiring never saw anything: two POSTs produced no history reload
  // and no transcript entry.
  assert.equal(fixture.ui.messages.children.length, 0);
  assert.equal(fixture.requests.length, 4);
});

test('a differently cased repeat of the same canonical pair replays the same stored ticket', async () => {
  const fixture = ready([ticket(), ticket()]);
  await settle();
  await openForm(fixture, { email: 'Visitor@Example.COM', note: 'Please ring me' });
  await send(fixture);
  await openForm(fixture, { email: '  VISITOR@example.com  ', note: '  Please ring me ' });
  await send(fixture);
  assert.equal(escalationRequests(fixture).length, 2);
  const outcome = fixture.ui['escalate-outcome'];
  assert.ok(outcome.textContent.startsWith(TRUTH));
  assert.ok(outcome.textContent.includes('replayed unchanged'));
  assert.deepEqual(JSON.parse(escalationRequests(fixture)[1].init.body), {
    email: '  VISITOR@example.com  ', question_note: '  Please ring me ', consent: true,
  });
});

test('submitting a second pair appends, and resubmitting the first replays its own ticket', async () => {
  const second = ticket({ ticket_id: 'd'.repeat(32), recorded_at: '2026-10-20T13:00:00Z' });
  const fixture = ready([ticket(), second, ticket()]);
  await settle();
  await openForm(fixture, { email: 'first@example.com', note: 'First request' });
  await send(fixture);
  assert.ok(fixture.ui['escalate-outcome'].textContent.includes(`Request reference: ${'c'.repeat(32)}`));
  await openForm(fixture, { email: 'second@example.com', note: 'Second request' });
  await send(fixture);
  assert.ok(fixture.ui['escalate-outcome'].textContent.includes(`Request reference: ${'d'.repeat(32)}`));
  assert.equal(fixture.ui['escalate-outcome'].textContent.includes('replayed unchanged'), false);
  await openForm(fixture, { email: 'First@Example.com', note: 'First request' });
  await send(fixture);
  const outcome = fixture.ui['escalate-outcome'];
  assert.ok(outcome.textContent.startsWith(TRUTH));
  assert.ok(outcome.textContent.includes(`Request reference: ${'c'.repeat(32)}`));
  assert.ok(outcome.textContent.includes('replayed unchanged'));
  assert.equal(JSON.parse(escalationRequests(fixture)[2].init.body).email, 'First@Example.com');
});

test('escalation runs without disabling or mutating the chat send path, both directions', async () => {
  let finish;
  const deferred = new Promise((resolve) => { finish = resolve; });
  const fixture = ready([() => deferred, json(turn())]);
  await settle();
  await openForm(fixture);
  await send(fixture);
  assert.equal(escalationRequests(fixture).length, 1);
  assert.equal(fixture.ui['escalate-send'].disabled, true);
  assert.equal(fixture.ui.send.disabled, false);
  fixture.ui.question.value = 'How do I start?';
  fixture.ui.send.dispatch('click');
  await settle();
  assert.equal(fixture.ui.messages.children.length, 2);
  assert.match(fixture.ui.status.textContent, /AI answer received/u);
  assert.equal(fixture.ui['escalate-outcome'].textContent, 'Recording your support request…');
  finish(ticket()); await settle();
  assert.ok(fixture.ui['escalate-outcome'].textContent.startsWith(TRUTH));
  assert.equal(escalationRequests(fixture).length, 1);
  // Reverse direction: a chat turn in flight never blocks escalation.
  let blockChat;
  const chatDeferred = new Promise((resolve) => { blockChat = resolve; });
  const secondID = '12345678-1234-4123-8123-999999999abc';
  fixture.platform.crypto.randomUUID = () => secondID;
  fixture.responses.push(() => chatDeferred.then(() => json(turn(secondID, 'Second question', 2))), ticket());
  fixture.ui.question.value = 'Second question';
  fixture.ui.send.dispatch('click');
  await settle();
  assert.equal(fixture.ui.send.disabled, true);
  assert.equal(fixture.ui['escalate-send'].disabled, false);
  await send(fixture);
  assert.equal(escalationRequests(fixture).length, 2);
  assert.ok(fixture.ui['escalate-outcome'].textContent.startsWith(TRUTH));
  blockChat();
  await settle();
  assert.equal(fixture.ui.messages.children.length, 4);
});

test('pagehide aborts an in-flight escalation request like any chat request', async () => {
  const fixture = ready([(_url, init) => new Promise((_resolve, reject) => {
    init.signal.addEventListener('abort', () => reject(new Error('aborted')));
  })]);
  await settle();
  await openForm(fixture);
  fixture.ui['escalate-send'].dispatch('click');
  fixture.emit('pagehide', {});
  await settle();
  assert.equal(escalationRequests(fixture)[0].init.signal.aborted, true);
  assert.equal(fixture.timers.size, 0);
});

test('defensive form submission shares the identical validated consent path', async () => {
  const fixture = ready([ticket(), ticket()]);
  await settle();
  await openForm(fixture, { consent: false });
  fixture.ui['escalation-form'].dispatch('submit', {});
  await settle();
  assert.equal(escalationRequests(fixture).length, 0);
  assert.match(fixture.ui['escalate-error'].textContent, /consent/iu);
  fixture.ui['escalate-consent'].checked = true;
  fixture.ui['escalation-form'].dispatch('submit', {});
  await settle();
  assert.equal(escalationRequests(fixture).length, 1);
  assert.ok(fixture.ui['escalate-outcome'].textContent.startsWith(TRUTH));
});
