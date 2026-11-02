# RATE-04 independent assembled-tree acceptance

## Recommendation and stop point

**ACCEPT the recovered-rating tree and the two supplied test follow-ups for this bounded local acceptance scope.** No acceptance-blocking defect or failing required check was found. Recovery's missing trusted-input/actual-backend rating workflow, strongest storage assertions, and independent full-fuzz evidence are now supplied by independent execution on the assembled tree below. One low-severity test-artifact cleanup finding remains; it does not invalidate the results.

This is a recommendation to the coordinator, **not task closure, integration authorization, deployment, or production readiness**. Main-tree Escape/UI and CI changes were not imported or validated. Validator stops at this handoff; coordinator owns closure and integration decisions.

Claimed `.scratch/automation/rating-acceptance-brief.md` before assembly/checks. Read repository instructions, historical goal, glossary/context map, rating spec and all five issues, architecture/testing/fuzzing/delegation contracts, both published application contracts, ADR 0001/ADR index, project check adapter, and all three supplied handoffs. Independent review/execution only; no nested workers or application/test repairs.

Evidence root throughout: **`.scratch/automation/rating-acceptance/`**. Runtime clock reports **2026-11-02**, whereas session date is 2026-10-22; logs and generated server timestamps retain the actual runtime clock rather than fabricated dates.

## Exact assembly and fingerprint

Worktree: `/home/me/Documents/agent-worktrees/assistant-rating-acceptance`.

- Verified detached base commit: `12532877e0e4f3ed2e4f7a880567c98a6dfce29e`.
- Base Git tree: `1d4c26177d4686c7c4d781717cc4d9e0bd00b7e2`.
- Initial tracked diff was empty; only preexisting untracked `.scratch/automation/` was present.
- Verified supplied SHA-256 values and each patch's single destination, ran **`git apply --check` for both before applying either**, then mechanically applied the supplied files with `git apply`. No patch was authored or altered.
- Independently reconstructed each expected file from base lines plus supplied hunks and required byte equality with the applied file. Browser result also matches its owner's published file hash.
- Final tracked scope is exactly `internal/chat/rating_test.go` (+183/-28) and `web/tests/rating.browser.mjs` (+374/-125). `git diff --check` exits 0. No other tracked file changed.

| Artifact | SHA-256 |
|---|---|
| Supplied `acceptance-inputs/backend.patch` | `98bc40d3700374850f3c8bdc29a2122a418d2c37c2bad6c2fd9625912adb1ad6` |
| Supplied `acceptance-inputs/browser.patch` | `dab81ae7b84b3517a48eadc3ed8fa4278df50eb008a634ddb910086ae4b3efbc` |
| Result `internal/chat/rating_test.go` | `9f01ae85321cd13424a2deab40b6bca9aa0f8e9da6e6eb6d48ab29b34df6d16c` |
| Result `web/tests/rating.browser.mjs` | `4151460c198d7c5248eae618806dbf8263d96f21248fb209c0c740a9b62dd05b` |
| `assembled.patch` (`git diff --binary`) | `8c18b459bded01054667f458c2f1754b3441f3eaea0f4c76ed09d24280242685` |
| `logs/tracked-tree.sha256` manifest | `185dbf0f17493720e5ef6e9ba0ebdca10de0c25fa9d974c8c876c35d0195d2e7` |
| Built `assistant` executable | `a81605e5d75bcfcdbfdab1bcb345cb53768ed4f9ab5ffe94ca4ed1c98bb3cb82` |

The tracked-tree manifest contains `Git mode SHA256  path` for every tracked working-tree file, in `git ls-files -s` order; its digest fingerprints the complete assembled tracked tree, not just the base commit. Untracked evidence is intentionally outside that fingerprint. Details: `logs/preassembly.txt`, `logs/assembly.txt`, `logs/fingerprint.json`, `logs/final-status.txt`.

## Independent commands and actual results

Toolchain: Go `go1.26.3 linux/amd64`, Node `v26.1.0`, Chromium `154.0.8037.97` (Arch Linux). No downloads or installs. For gates/direct Go execution exported `OPENAI_API_KEY= TURSO_AUTH_TOKEN= TURSO_DATABASE_URL= OPENAI_BASE_URL= GOPROXY=off GOTOOLCHAIN=local`; direct Go uses `TMPDIR=/tmp/opencode`. That adapter-required directory already existed. Browser subprocesses receive minimal explicit environments, empty service credentials, loopback origins and fresh disposable SQLite files. No live provider, hosted Turso, mail, production service or remote migration was used.

`E="$PWD/.scratch/automation/rating-acceptance"` in commands below. Logs are relative to the evidence root.

| Command | Actual exit/result | Evidence |
|---|---|---|
| `make format` | 0, PASS; read-only | `logs/format.log` |
| `make lint` | 0, PASS | `logs/lint.log` |
| `make static` | 0, PASS | `logs/static.log` |
| `make unit` | 0, PASS; adapter tests, Go race suite, **62/62 Node tests** | `logs/unit.log` |
| `make contract` | 0, PASS; Go race contract suite, **14/14 Node tests** | `logs/contract.log` |
| `make fuzz-smoke` | 0; seven targets, 2 seconds each; see baseline limitation below | `logs/fuzz-smoke.log` |
| `make fuzz-full` | 0; all seven targets reached mutation, 30 seconds each, two workers | `logs/fuzz-full.log` |
| `go test -race -count=10 -v ./internal/chat -run 'Rating'` | 0; **eight named rating tests each passed ten times**, plus selected rating fuzz seeds; 19.037s | `logs/rating-repeat.log`, `logs/independent-summary.json` |
| `go build -o "$E/assistant" ./cmd/assistant` | 0 | `logs/build.log`, `logs/binary.sha256` |
| `timeout 180s node web/tests/rating.browser.mjs "$E/assistant" "$E/rating"` | 0; **52 assertions passed / 0 failed**, 58 trusted control events | `logs/rating-browser.log`, `rating/results.json`, seven screenshots |
| `timeout 180s node web/tests/sandbox-send.browser.mjs "$E/assistant" "$E/sandbox-send"` | 0; **22 passed / 0 failed** | `logs/sandbox-send-browser.log`, `sandbox-send/results.json`, screenshots |
| `timeout 180s node web/tests/widget-visual.browser.mjs "$E/assistant" "$E/widget-visual"` | 0; **113 passed / 0 failed** | `logs/widget-visual-browser.log`, `widget-visual/results.json`, screenshots |
| `git diff --check` | 0 | `logs/final-diff-check.log` |

`logs/gates-summary.log` and `logs/execution-summary.log` capture actual exits and timing, not inferred success from worker handoffs. No timeouts, retries or validator-owned environment repairs were needed: **0 of 2 repair rounds used**.

### Fuzz coverage and budget honesty

All inline fuzz seeds/target source are fingerprinted by the assembled-tree manifest. Existing Go cached interesting inputs were also loaded; their baseline counts are recorded below, but their pre-run cache bytes were not snapshotted. There is no new repository failure corpus.

| Target | Smoke baseline completion | Full baseline, mutation evidence | Full total executions / new interesting |
|---|---|---|---|
| `FuzzDecodeEscalation` | 118/176; baseline only | 176/176, now fuzzing | 76,001 / 20 |
| `FuzzDecodeProvider` | 132/197; baseline only | 197/197, now fuzzing | 146,591 / 10 |
| `FuzzDecodeRating` | 128/204; baseline only | 204/204, now fuzzing | 98,373 / 9 |
| `FuzzDecodeTurso` | 125/181; baseline only | 181/181, now fuzzing | 119,778 / 11 |
| `FuzzParseImportJSONL` | 115/123; baseline only | 123/123, now fuzzing | 11,999 / 6 |
| `FuzzValidateMessage` | 89/89; mutation reached, 887 executions | 89/89, now fuzzing | 7,504 / 3 |
| `FuzzValidateOpenAIEndpoint` | 118/202; baseline only | 202/202, now fuzzing | 62,357 / 9 |

The six baseline-only smoke exits are **not mutation success claims**. The planned full campaign supplies their bounded longer rerun: baseline loading took approximately 2–5 seconds, leaving mutation time within each 30-second budget. No full target was baseline-only, so no extra budget increase was necessary. Import/message execution counters stopped advancing late in their full budgets; both had already reached mutation and exited successfully, which is bounded campaign evidence, not an unlimited-progress or performance guarantee.

## Acceptance mapping and patch review

### Backend follow-up — meaningful assertions, not count-only success

- **Restart/durability:** `TestSQLiteRatingAtomicRestartAndMigration` directly reads all four persisted rating columns before closing both SQLite handles, compares the complete row after reopen/migration, then replays with a different proposed time and requires unchanged content/time. Also compares complete conversation and turn records and counts. `persistedRating` reads SQL independently of the production RETURNING decoder and checks canonical timestamp text.
- **Distinct races/whole rows/last writer:** `concurrentDistinctRatings` releases 24 distinct submissions through two store handles. Each response is compared with its own submitted score/comment/ID, not just a score range or any submitted pair. Unique later timestamps are required. Inspection of the existing SQL's monotonic replacement rule supports using the greatest returned timestamp to identify the final serialized row; direct persisted-row and post-reopen comparisons verify that row.
- **Uncertain commit:** the loopback Turso-shaped fixture executes the real SQL then hides acknowledgement with HTTP 500. The test captures the actual committed row **before** retry, supplies a different proposed time, and requires returned/durable content and timestamp to equal that captured row. Thus a retry that silently rewrote time would fail.
- **Turso-shaped concurrency:** two adapter instances send concurrent pipelines to the local SQLite-backed HTTP fixture, with per-operation whole-row assertions, final durable row, replay timestamp and unchanged turn count. The fixture retains typed binds and execute/close checking. This is fixture evidence, not hosted Turso certification.
- **HTTP storage failure/privacy:** new `TestContractHTTPRatingStorageFailureReturnsSafe503` injects an actual SQLite update abort after authorization, not a mocked response. It checks fixed safe 503 JSON, no private diagnostic/comment/capability leakage, API security headers via `assertCode`, unchanged durable rating/history, and replay after trigger removal.
- **Existing boundaries:** existing strict JSON/body/comment validation, capability isolation, authorization-before-storage spy, exact-origin rejection, first-turn prerequisite, replay/revision and direct-peer rejected-attempt 429 tests remain intact and pass. SQLite v2-to-v3 upgrade/missing-schema behavior and identical-write races pass. No runtime code or prior assertions were removed to obtain success.

### Browser follow-up — trusted UI plus actual Go/SQLite

- The replacement script starts the independently built Go executable, not the old Node mock API; backend serves its unchanged embedded assets. Cross-origin iframe sandbox remains exactly `allow-scripts allow-same-origin`; CSP is checked. Chromium's process sandbox was **not disabled**.
- Real CDP pointer/keyboard/IME-style input selects native radios 1–5, ArrowLeft/Right and Tab focus, enters a comment, saves and revises. Assertions check associated labels and group name. No `.checked`/`.value` assignment, synthetic DOM click/dispatch, mocked API response or replacement application asset drives these interactions.
- Request-stage pausing deterministically exposes the real in-flight save; trusted repeated activation emits one request. Response-stage CDP Fetch reads real Go response bodies and continues them unchanged. This changes timing, not bodies/status/headers/credentials.
- First canonical feedback is score 5 / `Helpful — local browser`; revision is score 2 / `Revised by trusted input`. Identical UI replay returns the complete same envelope. There are five browser rating attempts (including deliberate offline failure), four successful browser rating responses, plus five separately labelled runner-HTTP rating probes.
- Go process **34478 exited**, then new process **34625** started on the same SQLite file while the original iframe remained open. Offline submission truthfully said **not confirmed**, cleared success and allowed retry. Restarted trusted retry returned exactly the prior revision, including `rated_at: 2026-11-02T10:39:31.418736209Z`.
- Browser reload is separately checked: a new guest gets empty history/hidden rating, not old capability recovery. Runner-only HTTP probes check zero-turn 409, uniform cross-capability 401 both ways, and unchanged original row. These are explicitly HTTP probes, not mislabelled UI actions.
- Privacy checks inspect host/iframe storage, DOM/URL and observed parent messages, redact creation capabilities from reports and avoid retaining headers. No browser exception, response-inspection failure or CSP security error occurred. The deliberate `ERR_CONNECTION_REFUSED` and canceled transfers remain in JSON, not suppressed.
- Independently parsed the JSON reports and opened final SQLite **read-only**: exactly two conversations, one completed turn, one rating, and stored rating exactly equals the revision/replay/restarted response. Results: `logs/independent-summary.json`.
- Inspected this run's restart-success, unavailable and new-guest screenshots; visible messages agree with assertions. Screenshots supplement, not replace, interaction/network/storage evidence.

### Existing behavior and disclosures

- Sandbox-send passes trusted mouse, button Enter/Space, Ctrl/Meta+Enter, textarea plain-Enter newline, honest missing-provider failure and explicit same-ID retry against actual Go/SQLite.
- Widget-visual passes desktop/mobile/mobile-dark/narrow-layout fixtures, pinned composer, disclosure, disabled WhatsApp and actual local recorded-**not-sent** email outcome. Long-history layout fixtures are explicitly synthetic DOM content, not persistence evidence.
- Sharing/reload details remain behind the user-approved native **More information** control, not expanded by default. Bounded-excerpts wording is present and pointer-accessible before send; ordinary/contract tests retain keyboard-native details/summary behavior. Rating keyboard evidence is native controls/CDP, not a screen-reader audit.
- Existing Escape cases in this base passed. They do not validate additional main-tree Escape scenarios absent from this assembly.

## Severity-ranked findings and limitations

1. **Low / P3 — browser profile cleanup is external to the runner. Owner: chat-widget test owner.** `web/tests/rating.browser.mjs` finally stops children/servers and writes JSON but does not remove `chromium-profile`; the two existing browser scripts likewise leave profiles. This run left approximately 2.7 MiB per scenario. No demonstrated raw-capability disclosure was found, but profiles are unnecessary retained browser artifacts. Validator removed **only its three disposable evidence-directory profiles**, after child shutdown; retained SQLite fixtures/logs/JSON/screenshots. `logs/cleanup.log` records this. Future owner work may automate cleanup; no repair was made here.
2. **Informational — scope limits, not failed acceptance criteria.** Local SQLite, loopback provider/Turso-shaped fixtures and generated data do not prove hosted Turso compatibility, live delivery/provider behavior, power-loss crash durability, production abuse controls, deployment security or staff workflows. Rating browser integration uses a completed clarification without paid inference; answered/history eligibility is covered by ordinary/contract tests. CDP trusted input is automation, not physical-device or assistive-technology certification. The privacy assertions are bounded observations, not a comprehensive leak audit.

**No critical/high/medium finding or outstanding blocker within the recovered-rating acceptance scope.** Full-fuzz and browser recovery blockers are satisfied by current independent evidence, not historical pass claims. Production no-go items and main-tree integration remain the coordinator's separate concerns.

## Final boundary/accounting

- No implementation repairs, nested delegation, downloads, other-worktree edits, shared ledger updates, commits, pushes, merges or deployments.
- Application-path writes were only mechanical application of the supplied patches to their two authorized test paths. Other writes were this brief's claim and validator-owned evidence under the allowed root, plus ordinary external Go temporary/cache output.
- Fresh SQLite fixtures contain generated test data only. Browser children and loopback hosts were stopped; no process matching this evidence root remained at cleanup. Disposable Chromium profiles were removed without changing browser security or test code.
- Final status: two authorized modified test paths and untracked `.scratch/automation/`; all other tracked files match base. **Handoff complete; stop.**
