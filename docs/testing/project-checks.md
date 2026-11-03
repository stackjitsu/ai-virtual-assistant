# Project validation adapter

The manifest is **active because ownership/contracts and executable commands are configured**, not because every acceptance criterion is complete. The backend and widget now have runtime implementations and behavioral tests; missing packages/tests/fuzz coverage still fail honestly. `make setup-check` is separately scoped evidence for command tooling only. Tests are required alongside development; strict TDD is not required (ADR 0001).

## Current tested baseline

The primary session independently reports `make format`, `make lint`, `make static`, `make unit`, `make contract` and `make fuzz-smoke` passing for the integrated slice. The completed backend handoff additionally reports all seven gates passing, including `make fuzz-full`. Evidence includes 24 ordinary + 12 contract Node tests, Go race tests with local SQLite persistence and OpenAI/Turso HTTP fixtures, and actual CLI subprocess checks for seeded local startup/no key, restart replay and remote migration/seed guards (zero fixture connections for rejected remote operations). These are local/fixture checks, not live-service integration. Integrated review and actual-browser acceptance remain pending; no deployment or production-readiness claim is made here. Documentation-only follow-up uses focused read-only checks rather than duplicating backend runs.

Backend full-fuzz evidence: Go `go1.26.3 linux/amd64`, 2 workers, 30 seconds per target; `FuzzDecodeProvider` 119,499 executions, `FuzzDecodeTurso` 23,344, `FuzzValidateMessage` 4,416; no failing/minimized corpus. The provider smoke budget was spent loading the concurrently expanded cache, so no smoke mutation count is claimed; the full run supplies mutation evidence. Seeds were working-tree inline seeds plus existing Go cache, with no persisted failure corpus. Exact seed-file hashes and logs are recorded in the backend handoff at `/tmp/opencode/ai-chat-backend-resume/HANDOFF.md`; logs: `/tmp/opencode/ai-chat-backend-tests.log`, `/tmp/opencode/ai-chat-backend-gates.log`, `/tmp/opencode/ai-chat-backend-fuzz-full.log`. Temporary paths are local evidence, not portable repository artifacts.

Requirements already available in the development environment: Go 1.26.3, Node 26, Python 3 with `scripts/requirements-checks.txt`, make; Chromium for subsequent independent browser evidence. No global formatter/linter install, npm dependencies, provider keys or remote DB are required. The runner empties OpenAI/Turso credentials for child checks; tests must use injected local fixtures, never live services. Reviewed public Go module downloads are allowed.

| Check command | Actual work / coverage requirement |
|---|---|
| `make format` | Read-only `gofmt -l` on Go sources; UTF-8/final-newline/no-trailing-whitespace/CRLF check on text sources/docs. Does not pretend to be a JS/Markdown pretty printer and never rewrites files. |
| `make lint` | Node `--check` for JS/MJS, Python AST parse, JSON parse; requires Go packages, runs `go vet ./...`. |
| `make static` | Targeted manifest/contract-path/owner-overlap checks, **full Draft-07 JSON Schema validation of `.harness/tasks.json`**, source syntax checks, requires Go packages, compiles Go tests via `go test -run=^$ ./...`. Harness manifest validation remains targeted, not full JSON Schema validation. Invalid task records fail before compilation. |
| `make unit` | Actual `scripts/test_*.py` command-tooling regressions, requires discovered Go `Test…` functions, `go test -race ./...` (includes fuzz seed corpus), requires/runs Node `web/**/*.test.mjs` except `*.contract.test.mjs`. |
| `make contract` | Targeted manifest checks, requires Go `TestContract…` tests, `go test -race -run=^TestContract ./...`, requires/runs `web/**/*.contract.test.mjs`. |
| `make fuzz-smoke` | Discover all compiled `Fuzz…` targets via `go test -list`; require the three named backend targets; run **each** with exact anchored selector, 2 seconds each, 2 workers. |
| `make fuzz-full` | Same discovery/coverage gate, 30 seconds **per target**, 2 workers. No automatic pass for zero targets. |

Native command form for each discovered fuzz target: `go test <package> -run=^$ -fuzz=^<target>$ -fuzztime=2s -parallel=2` (full: `30s`). Runtime grows with target count; no all-target wildcard that Go's fuzz runner rejects. Record actual Go version, target list, budget and corpus revision in worker handoff. Check-adapter tests validate discovery, exact target selection/budget, failure propagation, missing tests/packages and nonmutating formatting. They do not establish chat safety.

## CI and reproducible local entry points

`.github/workflows/validation.yml` runs independent format, lint, static, unit,
contract, fuzz-smoke and browser jobs for pull requests and manual dispatch.
Manual dispatch's `full_fuzz` checkbox additionally runs `make fuzz-full` in the
fuzz job; PRs never run that extended campaign implicitly. Jobs have a 25-minute
limit and do not fail-fast across the matrix. Checkout and upload/setup actions
are pinned to commit SHAs; the token is contents-read-only, checkout credentials
are not persisted, and no secrets or live-service configuration are supplied.
PR checkout tests GitHub's merge SHA (recorded alongside the requested SHA), not
necessarily the contributor's head SHA. Branch protection must be configured by
a repository administrator; adding this workflow alone does not enforce merges.

CI pins Go 1.26.3, Node 26.1.0, Python 3.14.3 and Playwright 1.58.2's bundled
Chromium. Playwright is used only to provision a fixed browser revision, not to
replace the existing test. Ubuntu 22.04 supplies the native runner libraries;
the runner image itself is mutable. Public dependency downloads are required,
but application tests use local fixtures only. Browser background traffic is
disabled/proxied to a closed local port by the existing regression script.

Local setup (keep the virtual environment under excluded `.scratch/`):

```sh
python3 -m venv .scratch/checks-venv
. .scratch/checks-venv/bin/activate
python3 -m pip install -r scripts/requirements-checks.txt
go mod download
make format lint static unit contract fuzz-smoke
# Explicit ledger-only diagnostic; never modifies records:
python3 scripts/check.py tasks-schema
# Requires Chromium; output directory must not already exist:
CHROMIUM=/usr/bin/chromium python3 scripts/browser_check.py .scratch/browser-evidence
# Extended release/high-risk campaign:
make fuzz-full
```

The browser adapter builds the real backend and runs
`web/tests/sandbox-send.browser.mjs` against local SQLite/HTTP fixtures with an
empty AI key. It exercises actual trusted mouse/keyboard sends and explicit
retry (22 assertions at the CI-01 baseline); it is not a replacement for all
widget/browser acceptance tests. Missing browser, script, test coverage or
validation dependency fails rather than skipping.

The POSIX browser adapter bounds browser-version to 15 seconds, backend-build
to 180 seconds, and the trusted-input Node regression to 120 seconds by default.
For a slower local host or a failure-containment test,
`BROWSER_CHECK_TIMEOUT_SECONDS` overrides only the Node regression bound with a
finite number from 0.1 to 300 seconds; it cannot disable the bound. The separate
version/build bounds remain fixed. Each stage starts a new
session/process group with live-service credentials emptied. On timeout,
SIGINT or SIGTERM, the adapter sends SIGTERM to that entire owned group, then
SIGKILL after a short grace period, even if its leader has already exited.
Successful/failed stage exits also clean up leftover group members. Cleanup has
at most two seconds of TERM grace and a two-second leader-reap bound per stage;
three stages therefore have a default subprocess budget of at most 327 seconds
(excluding ordinary filesystem/scheduling overhead). Normal descendants inherit
the group, including backend and Chromium children; deliberate session/group
escape is not a supported containment guarantee. SIGKILL of the adapter, runner
loss and uninterruptible kernel waits cannot guarantee cleanup/evidence.

`adapter-results.json` records overall status/error and stage, timeout, PID,
return code when available, elapsed time and TERM/KILL/leader-reap evidence.
It is separate from the unchanged Node `results.json`, which may be absent when
inherited browser cleanup hangs. Timeout/interruption always fails nonzero,
with stage and configured bound in the log (signals return 128 + signal number).
Only the immediate subprocess can be reaped by the adapter; terminated orphan
grandchildren are reaped by the host's init. Real executable crash/hang tests
exercise the unchanged Node regression and verify there are no live fixture
descendants, including a child that ignores SIGTERM; they also cover SIGINT and
SIGTERM interruption. Those fixtures test containment only, never real-browser
acceptance. No browser sandbox or CSP bypass is introduced.

Python tooling regressions
include schema required fields, types, enums, unknown fields, conditional
blocked/completed requirements, later ledger entries and nonmutation.

Always-uploaded per-job artifacts include tested SHA, final status, dependency
and check logs, and browser result JSON/logs/screenshots when produced. SQLite,
compiled binaries and browser profiles are deliberately excluded. Setup failures
still produce SHA/status evidence; detailed action-internal setup logs remain in
GitHub's job log. Artifacts have 14-day retention; cancellation/runner loss can
prevent upload, so artifacts are not a durability guarantee.

CI-01 local verification passed format/lint/unit/contract/fuzz-smoke and the
trusted browser check with system Chromium 154.0.8037.97, Go 1.26.3, Node 26.1.0,
Python 3.14.8. Full-schema static validation correctly failed on completed tasks
DISC-01, RATE-01, RATE-02 and RATE-03 with empty evidence arrays. The coordination
owner must supply genuine evidence or correct task status; the schema and ledger
were not weakened/edited. GitHub execution, the CI-provisioned Chromium revision,
and the manual full-fuzz branch were not executed locally. Detailed evidence is
in `.scratch/automation/ci-handoff.md` (local worker evidence, not release proof).

## Owner conventions / evidence

- Backend owns ordinary Go tests, published-boundary `TestContract…` tests (HTTP, SQLite/Turso, OpenAI), and `FuzzValidateMessage`, `FuzzDecodeProvider`, `FuzzDecodeTurso` in `internal/chat`. Add seeds/properties and minimized regressions under its own `testdata/fuzz/`. Fuzz invariants: arbitrary input cannot panic/hang; accepted outputs satisfy bounded records/citations/types; no malformed cell/provider input becomes success. All additional discovered targets run too.
- Widget owns ordinary `*.test.mjs` and `*.contract.test.mjs` in `web/`, using Node's `node:test` and assertions. Contract tests exercise loader/iframe handshake shape, source/origin rejection, exact request/response fields and safe DOM rendering. Prefer exported pure functions or injected browser fakes over a framework. A file with zero assertions is not behavioral evidence: review must check meaningful cases, not filenames alone.
- Browser validation uses existing Chromium and a local test host, with one real cross-origin loader/iframe flow plus spoofed-message rejection, submit/retry, safe text rendering and keyboard controls. Node tests are not a substitute for actual browser evidence. If browser proof is unavailable, record it as an owner-assigned blocker; do not claim a screenshot proves plugin integration.
- `make setup-check` runs formatting, syntax, targeted manifest checks and focused check-adapter/schema/browser-runner tests. This tooling-only command does not validate the live task ledger; `make static` does. Application checks remain separate and must be rerun after behavior changes. Do not add app stubs merely to turn these checks green.
- `go mod download` fetches pinned dependencies on a fresh clone. Foundation coordinates any `go mod tidy` or dependency changes with both owners and reruns gates. Ordinary `go test -race ./...` needs the already available Go race/C toolchain, but local SQLite itself is pure Go.
- Required checks follow [testing-contract.md](testing-contract.md): format/lint/static/unit each change; contract on public changes; bounded fuzz on affected input/state changes; full fuzz before release/high-risk changes. Current passing automated checks do not close pending independent review/browser acceptance or establish live-service, deployment or production security evidence.
