# UX-01G: explicit handoff actions with concise disclosure

Status: resolved — UX-01G chat-widget worker (2026-10-20); awaiting orchestrator review

Owner: chat-widget worker; context: chat-widget. Read `AGENTS.md`, `.harness/goal.md`, `docs/contracts/chat.md`, `web/README.md`, current widget markup/CSS/core/tests and the existing git diff. Claim this brief before editing. Allowed edits: `web/**` and this issue only. No commits, pushes, backend, shared contracts, harness files, academic files, credentials or live services.

## User-approved design

Keep the existing corner chat and white/larger-type appearance. Use the specification's visitor handoff model:

1. Normal questions continue to use **Ask the assistant**. A separate **Request human support** action opens a compact menu only after a conversation exists.
2. The menu contains:
   - **By email** — available and opens the existing consent-required email form.
   - **By WhatsApp** — visibly unavailable and disabled. It must never appear to be an operational channel. Label its reason briefly, for example “Unavailable: this support service has not enabled WhatsApp.” Do not request a phone number or imply messages will be sent.
3. The handoff menu itself must be keyboard-accessible, toggle with `aria-expanded`, close on Escape and outside click where practical, expose the selected channel state, and never auto-open on provider failure or any other server event. Keep the email path additive: it must not disable, hide or mutate normal chat, history, retry or disclosure behavior.
4. Use a compact disclosure pattern: keep the immediate safety warning visible (“AI may be wrong; avoid sensitive data; sample content isn’t official guidance”), with a clearly labelled accessible **More information** disclosure containing concise server/AI-provider sharing and guest-history facts. Avoid unnecessary technical wording, URLs, ports and demo labels. The required facts remain available before a visitor submits a question.
5. Conversation rating is a separate later action after a completed answer; do not add rating here. Staff/admin, WhatsApp delivery, notifications and backend changes are not part of this task.

## Compatibility invariants

- Preserve iframe sandbox/CSP/origin/postMessage protocol, capability isolation, request IDs/replay, inert text rendering, 2,000-character limits, explicit consent, the current recorded-not-sent truthful result, and every email field/status.
- Keep the current white surfaces in both browser color-scheme preferences and the visible 360×640 question/Send composer.
- Do not replace existing uncommitted user work. Update only relevant tests for changed semantics/wording and keep test fixtures/validation representative, not live data.

## Acceptance / checks

- Browser regression: after ready, Request human support opens menu; By email opens existing form and consent still fails closed; WhatsApp is disabled/unavailable and no request is possible; menu Escape/outside interaction behaves correctly; normal question submit still works. At 360×640, main Send remains visible without iframe scrolling, and long history scrolls independently.
- Run `make format`, `make lint`, `make static`, `make unit`, `make contract` and `git diff --check`; run `make fuzz-smoke` only for changed fuzzable parsing/state transitions. Capture safe desktop/mobile light/dark screenshots. Do not restart the running preview; report exact changed paths/evidence and the orchestrator will create a fresh plain preview URL.

## Answer

The visitor handoff model is implemented in the widget only; chat, consent, delivery honesty, sandbox/CSP/origin and the shared contracts are unchanged.

Behaviour now proven:

- **Request human support** appears in the conversation pane only once a conversation exists, opens a compact chooser (`aria-expanded`, labelled group "Ways to request human support"), and is never opened by a provider failure, a completed answer or any other server event.
- **By email** is the only available channel (`aria-pressed` reflects the selection) and reveals the existing consent-required email form inside the chooser.
- **By WhatsApp** stays `disabled`, is explicitly pinned to `aria-pressed="false"`, carries the truthful reason "Unavailable: this support service has not enabled WhatsApp." and collects nothing: a real click on it changes nothing and no phone input exists in the markup.
- **Escape** dismisses the whole handoff disclosure and returns focus to its trigger; it is the only key that closes it, so a typed letter can never cancel a request in progress. A click outside the handoff region closes it as well.
- Closing the disclosure hides the email form **without discarding what the visitor typed**: reopening it restores the same email, note and consent state. A chat interaction (clicking the question box) closes the disclosure exactly like an outside click and never disables or mutates the chat, history, retry or disclosure behaviour.
- The handoff panel lives in the scrollable conversation pane, never in the pinned composer, so the question and **Send question** controls stay visible at 360×640 with the panel open.
- The immediate warning ("AI may be wrong; avoid sensitive data. Sample content isn't official guidance.") stays visible; the labelled **More information** disclosure keeps the app-server/AI-provider sharing and guest-history facts, with no origin, URL, port or provider brand.

Changes made by this worker (the chooser markup, core wiring and CSS were already present as uncommitted UI-01/UX-01 work and were kept, not replaced):

- `web/assets/widget-core.mjs` — Escape-only keyboard dismissal (dropped an unapproved `q` shortcut); the unavailable channel is explicitly pinned to `aria-pressed="false"` on every control update.
- `web/assets/widget.html` — the chooser is a labelled group (`role="group"`, `aria-label="Ways to request human support"`).
- `web/tests/harness.mjs` — fixture `focus()`, `matches()` and `closest()` so focus and click-region behaviour are observable.
- `web/tests/escalation.test.mjs` — fixed the broken `readFile` import that failed the suite; added coverage for "no server event opens the menu", Escape + focus return + no other key closing it, outside-click close versus inside-click, an open menu never blocking chat, and typed details surviving a close/reopen.
- `web/tests/embedding.contract.test.mjs` — updated the chooser markup assertion; added a test that only the visitor toggle writes the menu state, that Escape is the only closing key and that the composer sits outside the scrollable pane.
- `web/tests/widget-visual.browser.mjs` — real mouse/key regression for the whole handoff flow.
- `web/README.md` — documents the chooser, the close semantics, the retained details and the new browser checks.

Checks and evidence:

- `make format`, `make lint`, `make static`, `make unit` (Go race suite + 55 Node unit tests), `make contract` (Go contract tests + 14 Node contract tests): PASS.
- `git diff --check`: clean.
- `make fuzz-smoke`: PASS (run as an unchanged-code safety check; no fuzzable parsing or Go state transition was touched).
- Offline visual/interaction browser run with a freshly built local binary: 113/113 (`/tmp/opencode/ux-01g-final-20261020/visual/results.json`), including real Escape and outside-click input, the disabled WhatsApp click, the composer-visible-with-panel-open geometry at 360×640, 180×320 and desktop, light and dark browser preference, and the recorded-not-sent outcome.
- Offline real-input send browser run: 22/22 (`/tmp/opencode/ux-01g-final-20261020/send/`).
- Safe screenshots (no visitor input): `desktop/mobile/mobile-dark/zoom-200-{ready,handoff-menu,long,email-consent}.png` and `sample-host-dark.png` under `/tmp/opencode/ux-01g-final-20261020/visual/`.

No commit, no push, no preview restart, no live service call, and no file outside `web/**` and this brief was changed.
