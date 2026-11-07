# RATE-04 recovery — independent validation handoff

## Revision, scope and disposition

- Reviewed exact HEAD **`12532877e0e4f3ed2e4f7a880567c98a6dfce29e`** in `/home/me/Documents/agent-worktrees/assistant-rating-review`.
- **All requested automated commands and existing browser checks passed. Acceptance remains qualified by the coverage blockers below; do not close the central ledger on this evidence alone.** No production/release readiness claim.
- Read the recovery brief, repository instructions, pinned goal, glossary, context map, rating spec and all five issues, chat contract, testing/fuzzing contracts and ADR 0001. Reviewed rating handler/decoder, atomic SQL store, migration 3, backend tests, widget controls/protocol/tests and browser scripts.
- Owners: RATE-02/chat-backend owns `internal/**`, `database/**`, `cmd/**`; RATE-03/chat-widget owns `web/**`. Coordination owns follow-up and acceptance.
- No implementation fixes, commits, pushes, nested workers, branch changes, remote migrations or other-worktree edits. Writes are review evidence under `.scratch/automation/` and temporary/cache test output. Initial and final tracked diffs are empty. `git status --short` shows only `?? .scratch/automation/`.
- All service fixtures used loopback and disposable SQLite databases. Build dependencies were available; Chromium was available. No live provider, Turso or email service was used. Make's adapter clears provider/Turso credentials; browser backend processes use explicitly empty credentials and fresh evidence-directory databases.

## Commands and actual results

Evidence root: `.scratch/automation/evidence/`. Toolchain: Go `go1.26.3 linux/amd64`, Node `v26.1.0`, Chromium `154.0.8037.97` (Arch Linux).

| Command | Result | Evidence |
|---|---|---|
| `make format` | exit 0, PASS; non-mutating | `format.log` |
| `make lint` | exit 0, PASS | `lint.log` |
| `make static` | exit 0, PASS | `static.log` |
| `make unit` | exit 0, PASS; Go race tests, adapter tests, 62 Node tests | `unit.log` |
| `make contract` | exit 0, PASS; Go race contract tests, 14 Node contract tests | `contract.log` |
| `make fuzz-smoke` | exit 0, PASS; discovered targets, 2 seconds each, parallelism 2 | `fuzz-smoke.log` |
| `env OPENAI_API_KEY= TURSO_AUTH_TOKEN= TURSO_DATABASE_URL= TMPDIR=/tmp/opencode go test -race -count=10 -v ./internal/chat -run 'Rating'` | exit 0; ten repetitions of rating tests, with fuzz seeds also selected | `rating-repeat.log` |
| `env OPENAI_API_KEY= TURSO_AUTH_TOKEN= TURSO_DATABASE_URL= TMPDIR=/tmp/opencode go test ./internal/chat -run='^$' -fuzz='^FuzzDecodeRating$' -fuzztime=15s -parallel=2` | exit 0; 199/199 baseline entries completed; 60,281 executions, five new interesting inputs, no failure | `rating-fuzz-15s.log` |
| `env OPENAI_API_KEY= TURSO_AUTH_TOKEN= TURSO_DATABASE_URL= go build -o "$PWD/.scratch/automation/evidence/assistant" ./cmd/assistant` | exit 0 | `build.log` |
| `node web/tests/rating.browser.mjs "$PWD/.scratch/automation/evidence/assistant" "$PWD/.scratch/automation/evidence/rating"` | exit 0; script ignores arguments and uses its own temporary mock-API fixture | `rating-browser.log` |
| `node web/tests/sandbox-send.browser.mjs "$PWD/.scratch/automation/evidence/assistant" "$PWD/.scratch/automation/evidence/sandbox-send"` | exit 0; 22 passed, 0 failed | `sandbox-send-browser.log`, `sandbox-send/results.json`, screenshots |
| `node web/tests/widget-visual.browser.mjs "$PWD/.scratch/automation/evidence/assistant" "$PWD/.scratch/automation/evidence/widget-visual"` | exit 0; 113 passed, 0 failed | `widget-visual-browser.log`, `widget-visual/results.json`, screenshots |
| `git diff --check` | exit 0 | terminal review; no tracked modifications |

`gates-summary.log` and `browser-summary.log` retain exit codes. The smoke budget ended during baseline gathering for several targets, including rating (85/199); the additional 15-second rating campaign above explicitly reached mutation fuzzing. `make fuzz-full` was **not** independently run in this bounded recovery review. Historical worker full-fuzz claims are not substituted for current independent evidence.

## Acceptance evidence, with limits

### Backend/storage

- **Isolation and authorization:** `TestContractHTTPRatingValidationAuthorizationAndRevision` exercises wrong-conversation capabilities both directions, unknown IDs, missing/malformed bearer, null Origin, method/media/query/ID rejection. `TestContractRatingAuthorizationBeforeStorage` checks identical unauthorized bodies and zero `SaveRating` calls before authorization. Code inspection confirms authorization precedes the single store operation; the SQL operation itself handles eligibility and replay.
- **Validation:** contract tests reject fractional/exponent/string/bool/null/out-of-range scores, duplicate/escaped-equivalent/unknown keys, invalid UTF-8, oversized bodies, oversized comments and NUL. Fuzzing covers the canonical decoder boundary. Widget tests cover both Unicode and byte bounds and malformed responses.
- **Prerequisite:** HTTP zero-turn submission is 409; persisted clarification unlocks rating. Store tests cover zero-turn rejection and completed clarification/answer eligibility.
- **Replay/revision:** HTTP tests check trimmed canonical replay preserves timestamp, changed score/comment replaces with a later timestamp, and omitted comment becomes empty. SQL tests force a backward proposed clock and still require a distinct later revision timestamp. Handler/store inspection shows one conditional `INSERT ... ON CONFLICT ... RETURNING` with whole-row results, not a detached read/write sequence.
- **Durability/migration:** fresh migration, v2 upgrade, repeat migration and missing-table failure are tested. `TestSQLiteRatingAtomicRestartAndMigration` closes/reopens SQLite and verifies one rating row and one unchanged turn. This is database-handle reopen evidence, **not** an OS-process/browser restart workflow; it does not compare all pre/post-restart fields.
- **Concurrency:** 24 distinct revisions run concurrently; one row remains after reopening. `TestSQLiteRatingConcurrentIdenticalWritesReplaySingleTimestamp` uses two independently opened SQLite stores for 20 paired races and checks both responses and durable timestamp agree. All passed under race detection and ten repetitions. Distinct-write test only checks score range/comment prefix rather than the exact submitted score/comment pairing, so its “mixed row” assertion is weaker than its label.
- **Failure/replay:** SQLite trigger-induced write abort is exercised. Local Turso-shaped HTTP fixture executes SQL against local SQLite, drops a post-commit response, requires `ErrUnavailable`, then retries successfully. This proves adapter/fixture behavior, not hosted Turso behavior. The retry assertion checks revised content/later timestamp but does not independently capture the uncertain committed timestamp before retry. HTTP rating-specific store-failure-to-503 is apparent in code, not directly injected by these rating HTTP tests; widget 503 handling uses a mocked response.
- **Limits:** rejected/unauthorized attempts consume the independent direct-peer rating window; spoofed forwarding headers do not evade it; 11th attempt yields 429 with `Retry-After: 60`.

### Widget/browser and existing behavior

- Node rating tests cover hidden-before-turn, clarification/history eligibility, response validation, revision/retry, unconfirmed response honesty, double-click exclusion, independently expiring rating/escalation cooldowns and terminal 401 handling.
- Rating Chromium script runs real cross-origin iframe/sandbox/CSP loading, but uses a **Node mock API**, not the Go backend. It sets radio `.checked` and textarea `.value` and invokes `.click()` through JavaScript for rating/revision. Therefore its passing “1–5 selection” label is **not evidence of trusted pointer/keyboard selection or submission**. Actual CDP mouse/Enter input only proves the More information disclosure opens; it does not exercise radio keyboard navigation despite the nearby comment.
- Sandbox-send browser regression runs the actual built Go server and fresh SQLite. It proves browser-trusted chat mouse, Enter, Space, Ctrl+Enter and Meta+Enter; plain Enter inserts a newline; missing provider key fails honestly and explicit retry preserves request identity. This is real-input **chat**, not rating, evidence.
- Visual browser regression runs actual Go/SQLite at mobile-dark, desktop, mobile and narrow/zoom fixture dimensions. It passes pinned composer, disclosure text, disabled WhatsApp, handoff focus/close behavior and recorded-not-sent email assertions. Desktop clarification and email recording use real browser input. Sharing and reload details remain behind the native More information control, not expanded by default; the rating fixture also asserts “bounded approved excerpts” wording.
- **Escape scope:** only behavior already present in this SHA was exercised by the existing visual script. No uncommitted Escape patch was imported, inspected in another worktree, or credited. Passing existing Escape cases does not validate any additional scenarios addressed by that absent patch.

## Blockers and coverage gaps, ordered by severity

1. **Acceptance blocker — missing rating real-input/actual-backend browser integration. Owner: RATE-03/chat-widget, coordinated with RATE-02 and independent validation.** Existing browser evidence does not connect accessible pointer/keyboard rating selection, comment entry, save/revise and truthful outcomes to the actual Go/SQLite backend. No browser rating restart workflow is covered. Add/rerun an owned local integration regression; the present mock UI and HTTP tests cannot be called that regression. No observed production defect is asserted from this coverage gap.
2. **Coverage blocker — incomplete strongest storage assertions. Owner: RATE-02/chat-backend.** Restart verifies counts rather than complete rating replay content/timestamp; distinct concurrent writes do not assert exact submitted field pairs/last committed row; uncertain-commit retry does not compare the timestamp to the actual hidden committed row. Turso fixture has no concurrent-rating race scenario. These are missing evidence for the strongest durability/whole-row/remote replay claims, not demonstrated corruption. Strengthen owner tests before claiming those properties fully validated. Also add direct HTTP storage-failure evidence if required for RATE-04 closure.
3. **Release gate outstanding — independent full fuzz not run. Owner: orchestrator/validation with backend owner.** The requested recovery smoke suite and extra rating mutation campaign passed; neither replaces `make fuzz-full` before release/high-risk boundary acceptance. No release is authorized here.

No failing automated command or demonstrated implementation regression was found in this review. Follow-up belongs to the named owners, not speculative validator fixes. The central task ledger and other coordination files were deliberately left untouched.
