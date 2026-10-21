# UI-01: formal corner-widget visual refresh

Status: completed (visual pass accepted with documented mobile-scroll limitation)

Owner: chat-widget worker. Context: chat-widget. Read `AGENTS.md`, `.harness/goal.md`, `docs/architecture/context-map.md`, `docs/contracts/chat.md`, `web/README.md`, this brief, and the current `git diff` before editing. Claim this issue before implementation. Allowed implementation paths: `web/**` and this issue only. No commits or pushes.

## Goal

Make the existing corner launcher and iframe chat feel clean, formal and restrained, inspired by the clarity of Google's login UI rather than copying its identity. Complete the visitor-visible visual treatment of the launcher, disclosures, messages, compose form, loading/error/retry/history states, and email escalation form. Keep the sample host a sample, not a new product page. Do not add visitor login, staff pages, feedback, WhatsApp, or backend changes.

## Existing work / compatibility

- There are already uncommitted edits in `web/assets/loader.css`, `web/assets/widget.css`, and `web/assets/widget-core.mjs`. Treat these as user work: preserve and build on them, inspect the diff, never replace them wholesale. Existing blue/light token direction is the starting point.
- The loader stylesheet is loaded on an arbitrary host page, while widget CSS lives in the iframe; tokens in the iframe do not automatically apply to the host launcher. Make launcher styling self-contained with safe defaults, avoiding dependence on or accidental capture of host CSS variables. Preserve embedding, sizing and mobile behavior.
- Keep the published HTTP/embedding contract, exact iframe sandbox/CSP, origin checks, guest capability isolation, inert text rendering, chat retry/ID semantics, and consent/delivery honesty unchanged. No external fonts, npm install, or new network dependency. Do not present unconfigured email as sent.
- If the pre-existing `widget-core.mjs` Escape/q panel change needs adjustment for consistency/accessibility, make only a narrowly scoped fix and test it; do not turn this into a new feature.

## Acceptance

1. Corner widget stays a corner widget, opens/closes clearly, fits narrow mobile and desktop viewports, and remains legible in light/dark OS themes as applicable. A single restrained accent and clear primary/secondary actions are consistent across parent launcher and iframe.
2. Ready, loading, empty, answer/clarification, error/retry, older history, and email consent/recorded/error states have clear visual hierarchy and visible focus/disabled states. Long text and zoom do not hide the send/handoff controls; labels/disclosures remain readable.
3. No behavioral regressions: ordinary chat, keyboard send, explicit handoff, consent gating and truthful recorded-not-sent outcome work. Document any accessibility/layout limitations discovered.
4. Add/adjust focused tests for any changed behavior/markup where feasible and capture actual-browser visual evidence at desktop and narrow mobile widths if Chromium is available. Screenshots are visual evidence, not proof of browser behavior; use real interaction for behavior. Do not leak capabilities, messages or credentials into artifacts.

## Checks / handoff

Run `make format`, `make lint`, `make static`, `make unit`; `make contract` for affected embedding/widget contract; `make fuzz-smoke` only if changing fuzzable parsing/state transitions. Report paths changed, exact checks/results, screenshots/browser results, outstanding blockers and next decision. No live provider, Turso or SMTP traffic. Do not edit harness state or academic report files.

## Worker completion / evidence (2026-10-20)

- Updated `web/assets/loader.css` with independent literal launcher/frame defaults, dark OS colors, scoped host-safe selectors, a working `[hidden]` rule and reduced-motion behavior. Built on the pre-existing blue/light CSS rather than resetting it. Updated `web/assets/widget.css` for readable focus/disabled and secondary actions, narrow-screen button layout, dark contrast and paired consent control/text. `web/assets/widget.html` wraps the checkbox and its label together; `web/tests/embedding.contract.test.mjs` asserts the revised semantic markup. Preserved the existing `web/assets/widget-core.mjs` Escape/q change untouched. Added opt-in `web/tests/widget-visual.browser.mjs` and documented it in `web/README.md`.
- Checks from repository root after final edits: `make format` PASS; `make lint` PASS; `make static` PASS; `make unit` PASS (48 Node unit tests, Go and tooling checks); `make contract` PASS (13 Node contract tests and Go checks). `go build -o /tmp/opencode/ui-01-assistant ./cmd/assistant` PASS. `node web/tests/widget-visual.browser.mjs /tmp/opencode/ui-01-assistant /tmp/opencode/ui-01-visual-20261020-evidence2` PASS (23 actual-Chromium assertions). No fuzz-smoke: changed CSS and consent association only, no fuzzable parser/state transition changed by this worker.
- Browser evidence: `/tmp/opencode/ui-01-visual-20261020-evidence2/{desktop,mobile,mobile-dark}-{ready,email-consent}.png` and `results.json`. The script uses a different-origin local host with hostile CSS variables, clears provider/Turso/SMTP credentials, blocks external Chromium network via a closed proxy, drives trusted clicks for open/close, consent validation, clarification and a truthful recorded-not-sent support request. Screenshots are limited to unfilled form/empty chat states; no bearer token or user message appears in screenshots/results. Evidence directory also contains a temporary SQLite database and Chromium profile; keep it private.
- Layout limitation: the capped-height iframe scrolls internally on narrow screens/long content; controls are reachable by scrolling, but the send and handoff buttons are not simultaneously fixed in view. 200% browser zoom and arbitrary third-party host `!important` overrides were not independently exercised. Existing source links still navigate within the sandboxed frame as documented.
- Next decision: orchestrator reviews the visual evidence and chooses whether to accept UI-01 or request changes. No commit/push, deployment or live provider traffic performed.

## Orchestrator review

Reviewed desktop, mobile and mobile-dark screenshots and Chromium results; independently reran `make format lint static unit contract` and `git diff --check` (all pass). Preserved pre-existing widget-core work; launcher values are independent of parent host variables. The responsive iframe fits the viewport and all controls are reachable by scrolling, but the Send button begins below the fold on a 360×640 ready-state screen; keep a sticky/shorter mobile composer as a potential next UX task, not a claim that the button is always visible. Browser zoom and hostile host `!important` CSS remain unverified. UI-01 closes as the first visual pass, subject to user visual approval before any commit/push.
