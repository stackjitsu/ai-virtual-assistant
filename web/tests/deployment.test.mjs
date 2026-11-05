import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { json, settle, widgetFixture, turn } from './harness.mjs';

test('email policy fails closed for demo, failed, malformed and contradictory discovery without blocking chat/rating', async () => {
  for (const policy of [
    json({ public_demo: true, email_support_enabled: false }), new Error('offline'),
    json({}), json({ public_demo: false, email_support_enabled: true, secret: 'bad' }),
    json({ public_demo: true, email_support_enabled: true }),
    json({ public_demo: false, email_support_enabled: false }),
    json({ public_demo: 'false', email_support_enabled: true }),
    json({ public_demo: false, email_support_enabled: true }, 503),
  ]) {
    const f = widgetFixture(undefined, { policy }); f.init(); await settle();
    assert.equal(f.ui.send.disabled, false);
    assert.equal(f.ui['handoff-email'].disabled, true);
    assert.equal(f.ui['escalate-email'].disabled, true);
    assert.equal(f.ui['escalate-send'].disabled, true);
    f.ui['escalate-toggle'].dispatch('click'); f.ui['handoff-email'].dispatch('click');
    f.ui['escalate-email'].value = 'user@example.org'; f.ui['escalate-consent'].checked = true;
    f.ui['escalate-send'].dispatch('click'); f.ui['escalation-form'].dispatch('submit'); await settle();
    assert.equal(f.ui['escalation-form'].hidden, true);
    assert.equal(f.requests.some((r) => r.url.endsWith('/escalation')), false);
    assert.match(f.ui['support-policy'].textContent, /unavailable/u);
    assert.doesNotMatch(f.ui['support-policy'].textContent, /recorded|secret|bad/u);
    assert.equal(f.policyRequests.length, 1);
    assert.equal(f.policyRequests[0].init.headers.Authorization, undefined);
    assert.equal(f.policyRequests[0].init.credentials, 'omit');
    f.responses.push(json(turn())); f.submit(); await settle();
    assert.equal(f.ui.messages.children.length, 2);
    assert.equal(f.ui['rating-send'].disabled, false);
  }
});

test('email starts disabled while policy is unresolved', async () => {
  const f = widgetFixture(undefined, { policy: new Promise(() => {}) });
  // A non-Response also fails closed, without opening collection through direct callbacks.
  f.init(); await settle();
  f.ui['handoff-email'].dispatch('click');
  assert.equal(f.ui['escalation-form'].hidden, true);
  assert.equal(f.ui['escalate-send'].disabled, true);
});

test('sample and widget explicitly label the fictional institution before sending', async () => {
  for (const name of ['sample-host.html', 'widget.html']) {
    const source = await readFile(new URL(`../assets/${name}`, import.meta.url), 'utf8');
    assert.match(source, /Fictional Jitihada University demo/u);
    assert.match(source, /AI may be wrong; avoid sensitive data/u);
    assert.doesNotMatch(source, /support requests are recorded, but/u);
  }
});
