# ESC-02: widget escalation affordance with consent and email capture

Status: completed

Owner: widget worker. Context: chat-widget. Pre-read: `AGENTS.md`, `.harness/goal.md`, `.harness/tasks.json` (ESC-02), `.scratch/support-escalation/spec.md`, `docs/contracts/chat.md` (both the v1 section and the appended `## Escalation — slice 2 additive section`), `web/assets/{loader.js,widget.html,widget.css,widget-core.mjs}`, `web/tests/*`, `web/README.md`, this issue. Allowed edits: `web/**`, this issue, scratch evidence under `/tmp/opencode/ai-escalation-send/**`. No backend, contracts, harness shared state, or academic files. No commits or pushes.

## Task

1. Claim this file (Status: claimed) before any other edit.
2. Add a visitor-facing escalation affordance to the existing widget UI. Requirements:
   - A clearly labelled, explicit affordance (for example "Request human support by email") that opens a compact escalation form; never auto-triggered by provider failures or any server state — the visitor must opt in deliberately.
   - Form fields mirroring the additive contract exactly: required email input (client-side bound mirror: trim, lowercase once, ≤254 bytes, plausible local/domain shape — the server stays authoritative), optional bounded note (≤1,000 code points / 2,000 bytes), and consent as an explicit standalone checkbox (no pre-checked or bundled-consent tricks). Submit only with consent true.
   - The request goes to `POST /api/v1/conversations/{id}/escalation` with the same iframe-owned bearer capability and fetch discipline as messages (same-origin, credentials omit, no-store); no other data leaves the iframe.
   - Response handling: 201 shows the server-provided truthful outcome sentence verbatim (`delivery_not_wired` → "Your request has been recorded. Email delivery is not configured, so no email has been sent.") plus the recorded timestamp in a visitor-sane form; never word it as "email sent". 400 shows the safe validation wording; 409 explains the per-conversation bound and that a matching repeat returns the existing ticket; 429 honors Retry-After bounded parsing like existing paths; 401 follows the existing terminal/reauth handling consistent with message routes; 503 offers an explicit retry, no automatic paid or repeated attempts.
   - Duplicate replay is a feature: resubmitting the same canonical pair must show the stored ticket outcome, not an error.
   - All rendered server values remain inert text (no HTML/Markdown interpretation), consistent with existing rendering discipline.
   - The escalation form never disables or mutates the normal send path, history, or disclosures; it is additive UI.
   - Sandbox (`allow-scripts allow-same-origin`), CSP, and every existing handshake behavior remain byte-for-byte unchanged.
3. Tests alongside (ADR 0001): focused Node regressions at real control handlers (open affordance, validation, consent gating, successful 201 rendering, 400/409/429/501/503-ish paths, duplicate replay, inert-text rendering); extend the embedding contract test to cover the new affordance markup; keep all existing 40 tests green; update `web/tests/embedding.contract.test.mjs` expectations only for additive UI presence.
4. Update `web/README.md` (controls description + optional separate browser script usage) and this issue with Status/changed paths/evidence.

## Acceptance

- Affordance + form implemented additively; consent fails closed; truthful outcome wording only
- All existing suites green plus new focused regressions; `make format lint static unit contract` passes
- No sandbox/CSP changes; no live services; no commits or pushes

Report changed paths, gate outputs, and any blocker with evidence rather than widening scope.

## Completion record (2026-10-20)

### Changed paths (exact)

- `web/assets/widget.html` — additive `<section class="escalation">` between the chat form and `<noscript>`: toggle button, hidden compact form (email input, note textarea, standalone unchecked consent checkbox, alert/status paragraphs, hint, Send button). Every pre-existing byte unchanged; no `type="submit"`, no inline styles/handlers, no pre-checked consent.
- `web/assets/widget.css` — appended-only escalation styles (`.escalation`, `#escalate-*`, `label.consent`, `.escalate-footer`); existing selectors untouched.
- `web/assets/widget-core.mjs` — additive escalation state/logic: `escalateOpen/escalateBusy/escalateCooldownUntil/escalateStoredTickets`, `escalateError/escalateOutcome/showTicket/escalateCooldown/submitEscalation`, toggle/submission/defensive-submit listeners, additive `controls()` lines (chat control lines byte-identical). `request()`/pagehide now track an `inFlight` Set of controllers instead of the single `activeController` — behavior-equivalent for chat (pagehide aborts chat requests as before, now provably every in-flight one) and it lets pagehide abort a concurrent escalation POST without widening any permission.
- `web/assets/protocol.mjs` — additive mirror validators only: `canonicalEmail`, `validEscalationEmail` (trim once → lowercase once, raw ≤254 bytes, local ≤64 / domain ≤253 bytes with exactly the server's two regexes), `validEscalationNote` (trim once; empty-or ≤1,000 code points/≤2,000 bytes via existing `validText`), `validEscalationTicket` (exact 4-field envelope, hex-32 ticket id, RFC3339, `delivery_not_wired` only, nonempty message). Existing validators untouched.
- `web/tests/escalation.test.mjs` — new focused regressions (20 tests) driving the real mounted control handlers.
- `web/tests/harness.mjs` — fixture element ids/tags for the new controls only (all prior ids/tags preserved).
- `web/tests/embedding.contract.test.mjs` — one additive presence test for the escalation markup (no pre-existing assertions changed).
- `web/README.md` — new "Email escalation (slice 2)" section + one added line in the checks list.

Not touched: `web/assets/{loader.js,loader.css,widget.js,sample-host.html}`, `web/tests/sandbox-send.browser.mjs`, `web/embed*`, backend/`internal/**`, `database/**`, `docs/**`, `.harness/**`, academic files. Sandbox (`allow-scripts allow-same-origin`) and CSP byte-for-byte unchanged; loader.js diff is empty and the widget HTML carries no CSP.

### Behavior implemented

- Affordance appears (hidden + enabled) only once a conversation exists; never auto-opened by provider failures or server state; toggles the panel with `aria-expanded`.
- Client validation mirrors the server exactly and stays fail-closed before any fetch: consent must be literally `true` (Boolean, not `"true"`/`1`), email shape/bounds, note bounds/controls. Server stays authoritative; typed values go on the wire as `{email, question_note? (sent only when non-empty after trim), consent: true}`.
- POST uses the identical `request()` seam: same-origin, `credentials:"omit"`, `cache:"no-store"`, `redirect:"error"`, `Accept/Content-Type JSON`, bearer capability from iframe memory, bounded abort; capability never in URL/DOM/parent messages.
- 201 → renders the server's fixed truthful sentence verbatim + `recorded_at` as UTC string + opaque ticket reference, all inert `textContent` (no HTML/Markdown interpretation anywhere); envelope validated against the single `delivery_not_wired` outcome, any other claim fails closed.
- Duplicate replay is a feature: same canonical pair re-sent shows the stored outcome again with "This is your previously stored request, replayed unchanged." (session-keyed by the server-canonical pair; no error path, no history mutation).
- 400 → local safe wording (consent/email/note/consent-box); raw server message never rendered. 409 `conflict` → explains the 5-request per-conversation bound and that a matching repeat replays the stored ticket. 429 → escalation-only cooldown honoring bounded `Retry-After` (chat send path untouched). 401/403 → existing `rejectAccess` terminal handling for the whole widget. 503/transport → explicit "use Send request to try again", zero automatic retries, identical-body resubmit is dupe-safe by replay.
- Escalation state machine never disables/mutates question/send/older/history/disclosures; chat and escalation can run concurrently; pagehide aborts both; cooldown timers are separate.

### UX wording decisions (explicit)

1. Toggle label: "Request human support by email"; panel `aria-label` uses the same sentence.
2. Field labels: "Your email" / "Optional note"; note field `name="question_note"` matches the contract field; placeholder "you@example.org" / "An extra detail the team should know (optional)".
3. Consent wording (its own label): "I agree to store this request and note for human support by email. No email is sent while delivery is not configured."
4. Send button: "Send request"; hint: "Consent is required before a request is recorded. Email ≤254 bytes; note ≤1,000 characters (2,000 bytes)."
5. In-flight status: "Recording your support request…" in the panel's own status line (chat status untouched).
6. Consent missing: "Tick the consent box first. Nothing is recorded for you without your consent."
7. Invalid email: "Enter one valid email address of up to 254 bytes, like you@example.org." Invalid note: "The optional note can hold up to 1,000 characters (2,000 bytes) without control characters."
8. 400: "This request could not be validated. Check the email, the note and the consent box, then send again."
9. 409: "This conversation has recorded its limit of 5 support requests. Sending the same email and note again shows your stored request; a new request needs a fresh chat."
10. 429: "Too many support requests. Please wait a moment, then send again." (bounded cooldown)
11. 503: "The support service is temporarily unavailable. Your request was not recorded. Use "Send request" to try again." Transport failure: "The support service could not be reached. Use "Send request" to try again."
12. Generic/unknown failures and non-contract 201 envelopes: "The request could not be completed safely. Please retry later."
13. 201 outcome line: server message verbatim, then "Recorded <Tue, 20 Oct 2026 12:00:00 GMT-style UTC string>." then "Request reference: <ticket_id>." — the honest server wording is displayed exactly as sent; the widget adds only the time and reference, never delivery claims.

### Test evidence

- Before: 40 Node tests green (28 `widget.test.mjs` + 12 `embedding.contract.test.mjs`); verified before any edit.
- After: 61 Node tests green — 28 existing widget unit tests, 20 new `escalation.test.mjs`, 12 pre-existing + 1 new contract test. `unit` → 48/48 unit tests pass; `contract` → 13/13 contract tests pass.
- New coverage at real control handlers: affordance visibility/toggling (+ no affordance without a conversation), email/note mirror semantics (incl. trim-once, byte/code-point bounds, control/multibyte rejection, nearest-boundary cases), honest-ticket envelope validation (rejects `email_sent`/broken shapes with safe wording), consent literal-true gate (0 fetches for `undefined`/`"true"`/`1`/object state), byte-equal duplicate replay of a stored pair (plus case/whitespace-canonical replay and append-on-new-pair replay of the first ticket), 201 verbatim rendering + UTC time + reference as inert text (radioactive `<img>/<script>/<b>` content stays text; no injection elements), 400/409/429/503/transport/401 paths (no auto retries, cooldown bounded to 120 s, chat send untouched in both directions), pagehide abort of an in-flight escalation POST, and the defensive submit listener sharing the identical validated path.
- Escalation-side fetch discipline asserted per request in the test (origin/Authorization/no-store/omit/redirect plus body exactness).

### Gate outputs (final tree)

- `make format` → `format: PASS`
- `make lint` → `lint: PASS`
- `make static` → `static: PASS`
- `make unit` → `unit: PASS` (check-adapter Python tests + `go test -race ./...` + 48/48 Node unit tests)
- `make contract` → `contract: PASS` (backend `TestContract…` + 13/13 Node contract tests)

No backend, contract, harness or academic edits; no commits/pushes; no live OpenAI/Turso/SMTP/network traffic — loopback test fixtures only. Browser-input integration evidence is left to ESC-03 by design (this issue's Node suites are not real-browser proof).
