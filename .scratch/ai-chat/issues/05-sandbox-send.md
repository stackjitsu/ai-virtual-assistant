# WIDGET-02: sandboxed User submission

Status: accepted and completed (orchestrator) after independent validator PASS, 2026-10-20

Owner: widget worker. Context: chat-widget. Allowed edits:web/** and this issue only. Contract:docs/contracts/chat.md. Preserve sandbox="allow-scripts allow-same-origin" and existing CSP; no implicit permission expansion or backend changes.

Claim: bounded frontend implementation and validation only; temporary tools/evidence live under `/tmp/opencode/ai-chat-send-fix/`. The delegated `.harness/tasks.json` WIDGET-02 brief is acknowledged; no runtime/session restart or regeneration is needed. Existing conversation, pending-ID/text, history-before-retry and capability-privacy invariants remain unchanged. Required gates: format, lint, static, unit, contract, focused real-input browser red/green.

## Reproduced failure

Real Chromium validation33passed/1blocked: native Send button activation/keyboard Enter produces zero POST requests because sandbox disables native form submission before the submit handler runs. web/assets/loader.js:50, widget.html:27–30, widget-core.mjs submit handler. Reproducer: go build -o /tmp/opencode/ai-chat-verification-resume/assistant ./cmd/assistant then node /tmp/opencode/ai-chat-verification-resume/verify.mjs. Evidence: report.md/keyboard-debug.json/run.log in that temporary directory. Later synthetic SubmitEvent checks do NOT prove working User submission.

## Fix / acceptance

- Establish a fast real-browser red-capable regression before fixing; distinguish native sandbox block from JS handler wiring or validation blocking.
- Preserve the deliberately restricted sandbox. An explicit non-native-send click action and textarea keyboard shortcut may share the actual send path, while retaining proper semantic labels, disabled-state/validation, accessible keyboard action and safe prevention of navigation.
- Use actual Chromium mouse activation and keyboard-generated action after fix; require exactly one POST with same request-ID/retry rules and real backend unavailable/clarification behavior. No programmatically dispatched submit event in the final User-submission assertions.
- Retain token privacy, strict messaging, DOM text safety, uncertain-history recovery and36existingNodechecks. Add focused Node regression(s) at real control handlers, not only another form-dispatch test. Keep browser regression evidence under /tmp/opencode/ai-chat-verification-resume/ or /tmp/opencode/ai-chat-send-fix/.
- No new product feature/remote call/dependency/commit/push/academic edit. Run format/lint/static/unit/contract and report original-browser loop rerun; full backend fuzz already passed and need not be repeated for a frontend-only fix.

## Hypotheses

1. Sandbox prevents native form submission before JS submit event: explicit click-driven action works with permissions unchanged.
2. Handler is not wired or event target differs: even direct shared send action would fail; inspect listeners/control state.
3. Native validation or disabled control blocks events: validity/enabled state changes would explain missing events independent of sandbox. Prior Chromium evidence says valid/enabled, making this less likely.

## Implementation and evidence

Confirmed cause: Chromium blocks native submission before the form submit listener when `allow-forms` is absent. The original controls were enabled/valid; browser-trusted mouse and button Enter/Space reproduced the security error. No sandbox/CSP expansion was made.

- `web/assets/widget.html`: named semantic form, explicit `type="button"` Send, accessible textarea hint/shortcut metadata.
- `web/assets/widget-core.mjs`: direct Send click and textarea Ctrl/Meta+Enter invoke one shared validation/send path. Native button Enter/Space generates its standard click. Plain Enter remains newline; composition/repeated shortcuts do not send. Defensive form listener still cancels navigation but is not required by actual User controls. Existing pending-ID/text, busy lock, history-first recovery and explicit retry remain unchanged.
- `web/tests/widget.test.mjs`: four new red-before-fix handler regressions; existing 36 tests retained. `web/tests/harness.mjs` now routes existing fixture sends through the actual Send click listener, not synthetic form submit. The existing embedding packaging test checks the labelled form/non-submit button/associated hint.
- `web/tests/sandbox-send.browser.mjs`: separate opt-in, dependency-free CDP regression; real cross-origin loader/iframe, actual Go CSP/SQLite, empty provider key, browser-trusted mouse/keyboard, no dispatched submit or permission relaxation. `web/README.md` documents controls and invocation.

Commands from repository root:

```sh
go build -o /tmp/opencode/ai-chat-send-fix/assistant ./cmd/assistant
node web/tests/sandbox-send.browser.mjs /tmp/opencode/ai-chat-send-fix/assistant /tmp/opencode/ai-chat-send-fix/red-confirmed
node --test --test-name-pattern='Send click|Textarea' web/tests/widget.test.mjs
# Above two ran BEFORE the production fix: browser 11 pass/5 fail; focused Node 0 pass/4 fail.
node web/tests/sandbox-send.browser.mjs /tmp/opencode/ai-chat-send-fix/assistant /tmp/opencode/ai-chat-send-fix/green
# Rebuilt embedded binary after production fix: browser 22 pass/0 fail.
make format lint static unit contract
# All PASS; Node 28 ordinary + 12 contract = 40. Go race suites and vet included.
node /tmp/opencode/ai-chat-send-fix/full-verify.mjs
# Copied original broader fixture, actual keyboard sends/no synthetic-submit fallback: 39 pass/0 fail.
```

Evidence: `/tmp/opencode/ai-chat-send-fix/{red-confirmed,green}/results.json`, `green/{mouse,button-enter-complete,button-space-complete,ctrl-enter-complete,meta-enter-complete}.png`, `gates.log`, `full-confirmed/{assertions.json,run.log}` and `/tmp/opencode/ai-chat-send-fix/report.md`. Actual mouse demo question gives 503 + visible unavailable + history GET; real retry preserves exactly the same UUID/text. Each keyboard action produces one POST and a stored clarification pair; plain textarea Enter inserts newline without POST. Broader fixture includes independent history/replay across SQLite process restart, safe HTML/source rendering and cross-origin privacy checks.

Fixture qualification: the first copied broad run passed 37 assertions then timed out posting wrong-version init before its module listener was installed. Waiting for actual `assistant:ready` fixed only that temporary fixture race; incomplete evidence remains under `full/`. Broad fixture launcher/retry still uses programmatic click, while focused portable evidence uses actual CDP mouse for both. No broad fixture synthetic submit remains.

Cleanup: zero owned assistant/Chromium processes after runs; no production debug instrumentation. No backend, loader permissions, contracts, dependencies, root README, runtime state or academic files edited; no live services/keys, commits or pushes. No full backend fuzz rerun for this frontend-only interaction change. Independent full validation remains the acceptance owner; successful live AI/Turso and other browser/OS combinations were not tested.
