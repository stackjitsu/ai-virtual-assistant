# STAGE-01 handoff

Status: completed scoped implementation; awaiting coordinator review/integration. Stopped after this handoff. No recursive delegation.

Base: `8410640e027c732a83dc35d354f8c4a56736be0e`, branch `feat/staging-demo-policy`. **Uncommitted**; no commit, merge, push, hosted CI, resources or deployment. No credential files read; tests explicitly empty service credentials, use loopback fixtures/temporary SQLite only, and perform no live AI/database/email calls. Foundation worktree and `.harness/` untouched.

## Contract delivered first

- `ASSISTANT_PUBLIC_DEMO=true` enables `HTTPConfig.PublicDemo`; unset/empty/`false` preserves legacy backend behavior. Only lowercase exact values accepted; invalid values fail before storage I/O. Foundation should set **`ASSISTANT_PUBLIC_DEMO=true`** in Render. No deviation from suggested seam.
- Public same-origin `GET /api/v1/deployment`: exact `{public_demo:boolean,email_support_enabled:boolean}`, inverse booleans derived from server configuration. JSON/no-store/nosniff; existing Origin/no-CORS; query rejected 400, other methods 405. No capability/record/secret fields; creation unchanged.
- Demo escalation returns fixed 403 `email_support_disabled` after existing Origin/rate guards, before body/ID/auth parsing, persistence, replay or outbox interaction. Rate limit still returns 429. Old success tickets cannot replay while disabled. No schema/provider/message/rating changes.
- Widget only enables email for strictly valid non-demo/enabled discovery. Pending, failed, malformed, extra-key or contradictory discovery fails closed for support; direct click/submit handlers cannot POST. Chat/rating continue. Server policy denial also disables support without terminating chat.
- Sample and pre-send widget explicitly identify fictional Jitihada University. Existing AI/sensitive-data, sharing and retained-history disclosures remain; white layout, sandbox/CSP/postMessage checks preserved.

## Exact changed source files

- `cmd/assistant/main.go`
- `cmd/assistant/deployment_test.go` (new)
- `internal/chat/http.go`
- `internal/chat/deployment_test.go` (new)
- `internal/chat/http_test.go` (keyed config literal)
- `internal/chat/escalation_test.go` (keyed config literal)
- `docs/contracts/chat.md`
- `docs/contracts/backend.md`
- `web/assets/widget-core.mjs`
- `web/assets/widget.html`
- `web/assets/sample-host.html`
- `web/tests/harness.mjs` (independent discovery fixture queue; legacy request indexes unchanged)
- `web/tests/deployment.test.mjs` (new)
- `web/tests/embedding.contract.test.mjs`
- `web/tests/rating.browser.mjs` (opt-in `BROWSER_PUBLIC_DEMO=true`, actual backend responses unchanged)

Task-local files only otherwise: `.scratch/staging/brief.md`, this handoff, `source-sha256.txt`, logs, local binary and browser evidence. Do not stage binary, SQLite databases, Chromium profiles or the running worker log. Source SHA-256 values for every file above are in `.scratch/staging/source-sha256.txt`; tested binary SHA-256 `10a24a56f7c33afe09903ae71eaad35e6832702f24f0589300a288757fd1700b`.

## Validation / exits

All paths below are relative to `.scratch/staging/`. Go `go1.26.3 linux/amd64`; Node `v26.1.0`; Chromium `Chrome/156.0.8078.4` at the brief's pinned local path. Python dependency environment used read-only from `assistant-visitor-rc/.scratch/release/checks-venv`, with `PYTHONDONTWRITEBYTECODE=1`.

| Command | Exit | Evidence |
| --- | --- | --- |
| `go test ./cmd/assistant ./internal/chat` | 0 | `go-focused.log` |
| `make format` | 0 | `format.log` |
| `make lint` | 0 | `lint.log` |
| `make static` | 0 | `static.log` |
| `make unit` | 0 | `unit.log`, Go race + 69 ordinary Node tests |
| `make contract` | 0 | `contract.log`, Go race contract + 15 Node contract tests |
| `make fuzz-smoke` | 0 | `fuzz-smoke.log`, 2 seconds/target, 2 workers |
| `make fuzz-full` | 0 | `fuzz-full.log`, 30 seconds/target, 2 workers |
| `go build -o .scratch/staging/assistant ./cmd/assistant` | 0 | binary hash above |
| default `rating.browser.mjs` | 0 | `browser-default.log`, `browser-default/results.json`: 61 passed / 0 failed |
| `BROWSER_PUBLIC_DEMO=true rating.browser.mjs` | 0 | `browser-demo.log`, `browser-demo/results.json`: 69 passed / 0 failed |
| default `sandbox-send.browser.mjs` | 0 | `browser-send.log`, `browser-send/results.json`: 48 passed / 0 failed |
| skill app lint | 0 | `ui-skill-lint.log`: 0 FAIL, 2 existing eyebrow WARN |
| `git diff --check` | 0 | clean |

Finite bounds: 180 seconds per ordinary gate, 300 full-fuzz, 150 per browser suite. Browser scripts launch trusted mouse/key inputs against actual demo/default Go backends, retain sandbox and original API responses, and clean up their subprocesses. Demo browser proves disabled email cannot expose collection or issue a POST, fictional labelling, chat clarification, ratings/revisions/replay/restart, close/reopen, isolation and capability privacy. Default suites preserve existing email draft/send/close and rating behavior. HTTP tests use panic-on-read bodies and forbidden storage adapters to establish disabled route performs no body/auth/replay/write I/O, including old stored ticket, malformed/oversized/unauthorized attempts; assert outbox unchanged, Origin/rate policies, exact discovery and CLI fail-before-storage.

Fuzz targets: `FuzzDecodeEscalation`, `FuzzDecodeProvider`, `FuzzDecodeRating`, `FuzzDecodeTurso`, `FuzzParseImportJSONL`, `FuzzValidateMessage`, `FuzzValidateOpenAIEndpoint`. Inline seeds at source hashes plus preexisting Go fuzz cache; no new persisted failure corpus. Full campaign passed every target; see logs for actual execution counts. No full-fuzz deferral needed.

Initial focused Node run exited 1 due to three obsolete presentation assertions (old sample/disclosure/button markup); one bounded repair updated additive markup expectations, retained the previous safety sentence, and added demo browser mode. All subsequent gates/browser runs passed. `node-focused.log` preserves initial failure, not current acceptance evidence.

## UI review / limits

No redesign or style draw: explicit task constraint retains approved white widget; no CSS/dependencies changed. Applied disabled semantics plus explanatory copy (skill UX disabled-state guidance). Existing eyebrow warnings retained because altering the approved branding/layout is out of scope. Inspected actual trusted-browser screenshots `browser-demo/demo-{1440,768,390}.png` plus `before-turn-disclosure.png`: no horizontal overflow; disabled channel and reason legible, no email input exposed; fictional/sharing/history disclosure accessible before send. Screenshots taken after opening the support menu naturally show the scroll pane at the support section; disclosure screenshot separately proves top content. Generic skill inspector was not run (real handshake/trusted-browser flow used instead); no generic inspector score claimed. No marketing imagery/motion/brand changes.

No implementation blocker. Coordinator owns integration with foundation deployment settings and final independent/integrated acceptance. Passing local tests do not authorize deployment, live model testing or production claims.
