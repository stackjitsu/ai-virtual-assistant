import test from 'node:test';
import assert from 'node:assert/strict';
import { validRatingComment, validRatingResult } from '../assets/protocol.mjs';
import { creation, emptyHistory, fail, id, json, settle, turn, widgetFixture } from './harness.mjs';

const result = (rating = 5, comment = 'Helpful', rated_at = '2026-10-20T12:00:00Z') => json({ rating, comment, rated_at });
const ratingRequests = (fixture) => fixture.requests.filter(({ url }) => url.endsWith('/rating'));

test('rating response requires exact server fields, canonical comment and UTC timestamp', () => {
  assert.equal(validRatingResult({ rating: 5, comment: 'Helpful', rated_at: '2026-10-20T12:00:00Z' }), true);
  for (const invalid of [
    { rating: 1, comment: ' hi ', rated_at: '2026-10-20T12:00:00Z' },
    { rating: 0, comment: '', rated_at: '2026-10-20T12:00:00Z' },
    { rating: 1.5, comment: '', rated_at: '2026-10-20T12:00:00Z' },
    { rating: 3, comment: '', rated_at: 'yesterday' },
    { rating: 3, comment: '', rated_at: '2026-02-30T12:00:00Z' },
    { rating: 3, comment: '', rated_at: '2026-10-20T12:00:00Z', reviewed: true },
    { rating: 3, rated_at: '2026-10-20T12:00:00Z' },
  ]) assert.equal(validRatingResult(invalid), false);
});

test('comment bounds are applied after trimming in Unicode code points and UTF-8 bytes', () => {
  for (const value of ['', '  ', `  ${'a'.repeat(1000)}  `, '😀'.repeat(500), 'a\nb\tc\rd']) assert.equal(validRatingComment(value), true);
  for (const value of ['a'.repeat(1001), '😀'.repeat(501), 'a\u0000b', 'a\u007fb', 'a\u009fb', 'a\ud800']) assert.equal(validRatingComment(value), false);
});

test('rating stays hidden before a completed turn and is available after clarification, including history', async () => {
  const fixture = widgetFixture([json(creation, 201), emptyHistory(), json(turn(id, 'How do I start?', 1, true)), result()]);
  assert.equal(fixture.ui['rating-section'].hidden, true);
  fixture.init(); await settle();
  assert.equal(fixture.ui['rating-section'].hidden, true);
  fixture.submit(); await settle();
  assert.equal(fixture.ui['rating-section'].hidden, false);
  fixture.ui['rating-choice-5'].checked = true;
  fixture.ui['rating-comment'].value = '  Helpful  ';
  fixture.ui['rating-send'].dispatch('click'); await settle();
  assert.deepEqual(JSON.parse(ratingRequests(fixture)[0].init.body), { rating: 5, comment: 'Helpful' });
  assert.match(fixture.ui['rating-outcome'].textContent, /recorded/iu);
  const recovered = widgetFixture([json(creation, 201), json({ messages: turn(id, 'How do I start?', 1, true).messages, next_before: null })]);
  recovered.init(); await settle();
  assert.equal(recovered.ui['rating-section'].hidden, false);
});

test('rating revision, transport retry and malformed 200 do not invent success', async () => {
  const fixture = widgetFixture([json(creation, 201), json({ messages: turn().messages, next_before: null }), result(), new Error('lost'), result(2, 'Changed', '2026-10-20T12:00:01Z'), result(3, 'Wrong')]);
  fixture.init(); await settle();
  fixture.ui['rating-choice-5'].checked = true;
  fixture.ui['rating-comment'].value = 'Helpful';
  fixture.ui['rating-send'].dispatch('click'); await settle();
  fixture.ui['rating-choice-5'].checked = false;
  fixture.ui['rating-choice-2'].checked = true;
  fixture.ui['rating-comment'].value = ' Changed ';
  fixture.ui['rating-send'].dispatch('click'); await settle();
  assert.match(fixture.ui['rating-error'].textContent, /not confirmed/iu);
  assert.equal(fixture.ui['rating-outcome'].hidden, true);
  fixture.ui['rating-send'].dispatch('click'); await settle();
  assert.equal(ratingRequests(fixture)[1].init.body, ratingRequests(fixture)[2].init.body);
  assert.match(fixture.ui['rating-outcome'].textContent, /updated/iu);
  fixture.ui['rating-send'].dispatch('click'); await settle();
  assert.equal(fixture.ui['rating-outcome'].hidden, true);
  assert.match(fixture.ui['rating-error'].textContent, /could not be confirmed/iu);
});

test('rating double-click is locked, cooldown is bounded, and chat remains available', async () => {
  let finish;
  const deferred = new Promise((resolve) => { finish = resolve; });
  const fixture = widgetFixture([json(creation, 201), json({ messages: turn().messages, next_before: null }), () => deferred,
    fail('rate_limited', 429, 'secret', { 'Retry-After': '999999' }), result(4, '')]);
  fixture.init(); await settle();
  fixture.ui['rating-choice-4'].checked = true;
  fixture.ui['rating-send'].dispatch('click');
  fixture.ui['rating-send'].dispatch('click');
  assert.equal(ratingRequests(fixture).length, 1);
  assert.equal(fixture.ui['rating-send'].disabled, true);
  assert.equal(fixture.ui.send.disabled, false);
  finish(fail('storage_unavailable', 503, '<script>secret</script>')); await settle();
  assert.match(fixture.ui['rating-error'].textContent, /could not be confirmed/iu);
  assert.doesNotMatch(fixture.document.body.textContent, /secret/iu);
  fixture.ui['rating-send'].dispatch('click'); await settle();
  assert.equal(ratingRequests(fixture).length, 2);
  assert.equal(fixture.ui['rating-send'].disabled, true);
  assert.equal([...fixture.timers.values()].some(({ delay }) => delay === 120000), true);
  fixture.ui['rating-send'].dispatch('click'); await settle();
  assert.equal(ratingRequests(fixture).length, 2);
  [...fixture.timers.values()].find(({ delay }) => delay === 120000).fn();
  fixture.ui['rating-send'].dispatch('click'); await settle();
  assert.equal(ratingRequests(fixture).length, 3);
  assert.match(fixture.ui['rating-outcome'].textContent, /recorded/iu);
});

test('rating unauthorized response is terminal and never reuses guest access', async () => {
  const fixture = widgetFixture([json(creation, 201), json({ messages: turn().messages, next_before: null }), fail('unauthorized', 401)]);
  fixture.init(); await settle();
  fixture.ui['rating-choice-1'].checked = true;
  fixture.ui['rating-send'].dispatch('click'); await settle();
  assert.equal(fixture.ui['rating-section'].hidden, true);
  assert.equal(fixture.ui.send.disabled, true);
  assert.equal(ratingRequests(fixture).length, 1);
  fixture.ui['rating-send'].dispatch('click'); await settle();
  assert.equal(ratingRequests(fixture).length, 1);
});
