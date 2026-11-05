# CI-01 bounded repair handoff

Status: implementation handed off for independent re-review; not centrally completed or accepted. Two implementation repair rounds used; no third round attempted. Stop at this handoff.

## Revision, scope and fingerprints

- Unchanged HEAD/base: `12532877e0e4f3ed2e4f7a880567c98a6dfce29e`.
- Repair-only patch: `evidence/repair.diff`, SHA-256 **`06db26ad123fad4286d44ce833e0ca57f467a4e4e1ff5283f4a57c1d349607f3`**.
  - Deterministic unified text diffs of the four files below against exact pre-repair snapshots in `evidence/before/`, in the listed order, labeled `a/<path>` and `b/<path>`.
- Complete current CI implementation patch, including the original preserved work: `evidence/implementation.diff`, SHA-256 **`d8c84629fa929cde3a64c28b88ef94e5226e1dcaa65897aa94eeefb3ed59cb32`**.
  - Hash input: `git diff --binary -- docs/architecture/context-map.md docs/testing/project-checks.md harness.json scripts/check.py`, followed by `git diff --no-index --binary /dev/null <path>` for `.github/workflows/validation.yml`, `scripts/browser_check.py`, `scripts/requirements-checks.txt`, `scripts/test_browser_check.py`, `scripts/test_tasks_schema.py`, in that order.
- Scratch briefs, handoffs, fixtures, environments and logs are excluded from implementation fingerprints. `evidence/fingerprints.txt`, `before-sha256.json` and `after-sha256.json` supply hashes. `preserved-source.log` confirms byte-identical workflow, manifest, other implementation scripts, task ledger/schema and unchanged trusted-input Node regression.

Repair implementation changed paths only:
1. `scripts/browser_check.py`
2. `scripts/test_browser_check.py`
3. `docs/testing/project-checks.md`
4. `docs/architecture/context-map.md`

Other writes: claim/round/status in `.scratch/automation/ci-repair-brief.md`; this handoff, `run-checks.sh`, containment-only `failure-chromium`, and `evidence/**` under `.scratch/automation/ci-repair/`. Preserve original CI patch and reviewer evidence; do not commit scratch SQLite, browser profiles, binaries or environments wholesale.

No application/widget/workflow/shared-ledger/schema edits, nested workers, downloads, credentials/live application services, other-worktree writes, commits, pushes, merges or deployments. Existing dependencies only, Go dependency network access disabled for checks. No sandbox bypass or CSP change.

## Acceptance mapping

### P2: finite subprocess containment

- Each stage uses a new owned POSIX session/process group, inherited log streams and cleared OpenAI/Turso credentials. Browser version bound: 15s; backend build: 180s; unchanged trusted-input Node regression: 120s by default.
- `BROWSER_CHECK_TIMEOUT_SECONDS` overrides only the Node bound, accepts finite 0.1–300s, and cannot disable it. Each stage always cleans up its group, including when the leader already exited. Timeout, SIGINT and SIGTERM trigger group SIGTERM then SIGKILL; a SIGTERM-ignoring descendant is not spared merely because the leader exited. Cleanup waits are bounded (up to two seconds TERM grace and two seconds immediate-child reap per stage). Default aggregate subprocess budget is at most 327s, excluding normal filesystem/scheduling overhead.
- Failure is nonzero (1 for timeout/ordinary failure, 128 + signal number for interruption). Timeout logs name stage, configured bound and group. Interruption console output names the signal; stage/bound remain available in JSON.
- `adapter-results.json` is separate from Node's `results.json`: overall status/error plus stage timeout/PID/outcome/elapsed time, return code when available, TERM/KILL and leader-reap evidence. Thus evidence remains available even when inherited Node cleanup never writes results.
- Four actual subprocess regressions run the unchanged Node script with an executable failure fixture: signal-crashed Chromium, hanging Chromium, SIGINT interruption and SIGTERM interruption. Every fixture launches a descendant ignoring SIGTERM; tests check none remain live. Existing missing-browser/reused-directory/real-script-failure tests remain, plus invalid timeout tests. Mocked selection tests are not the containment proof.
- Explicit retained crash run: `evidence/crash-fixture.log`, `crash-fixture/adapter-results.json`, `crash-fixture/fixture-pids.json`, `crash-cleanup-verification.log`. Node hit its inherited startup/cleanup hang; adapter timeout 8s, measured Node stage 9.023s including cleanup, exit 1. TERM and KILL sent, leader reaped, fixture parent PID 37183 and ignoring-child PID 37188 both absent from `/proc` afterward. No outer safety timeout fired.
- Containment fixtures are failure-path evidence only. The real credential-free Chromium adapter independently passed the unchanged CSP/sandbox/trusted-input checks: 22 passed, 0 failed.

Limitations documented in project checks: ordinary descendants must inherit the owned group; deliberate session/group escape is not covered. The adapter reaps its immediate child; host init reaps dead orphan grandchildren. Tests distinguish absent/zombie from live processes, and retained explicit verification found both fixture descendants fully absent. SIGKILL of the adapter, runner loss or uninterruptible kernel waits cannot guarantee cleanup/evidence. No broader security sandbox guarantee is claimed.

### P3: workflow ownership

The foundation ownership row in `docs/architecture/context-map.md` now explicitly includes `.github/workflows/**`, matching the already-authorized `harness.json` assignment. Workflow and manifest themselves are unchanged by this repair.

## Commands and actual results

Environment: existing `.scratch/automation/venv/bin` prepended to PATH; Go `go1.26.3 linux/amd64`, Node `v26.1.0`, Python `3.14.8`, real system Chromium `154.0.8037.97`. Versions/pins: `evidence/dependencies.log`. `PYTHONDONTWRITEBYTECODE=1`, `GOPROXY=off`, `GOSUMDB=off`, `GOTOOLCHAIN=local`, `GOVCS=*:off`, `GOTELEMETRY=off`; application credentials emptied. These are local host runs, not hosted GitHub/Playwright evidence.

Gate reproduction: `bash .scratch/automation/ci-repair/run-checks.sh` (300s outer safety limit per make invocation). Final gate status file: `evidence/results.txt`.

| Actual command | Result | Evidence under this handoff's `evidence/` |
|---|---|---|
| `make format` | PASS, exit 0, read-only | `format.log` |
| `make lint` | PASS, exit 0 | `lint.log` |
| `make static` | FAIL, make exit 2; exactly four pre-existing ledger evidence errors | `static.log` |
| `make unit` | PASS, exit 0; 23 tooling tests, Go race suite, 62 Node tests | `unit.log` |
| `make contract` | PASS, exit 0; Go contract race suite and 14 Node tests | `contract.log` |
| `make fuzz-smoke` | PASS, exit 0; seven targets, 2s/target, 2 workers | `fuzz-smoke.log` |
| `PYTHONPATH=scripts python -m unittest -v test_browser_check test_check test_tasks_schema` | PASS, exit 0; 23 tests including four real process regressions | `tooling-tests.log` |
| `python scripts/browser_check.py "$PWD/.scratch/automation/ci-repair/evidence/browser"` (outer `timeout 345s`) | PASS, exit 0; 22 real-browser assertions, no failures | `browser.log`, `browser-exit.txt`, `browser/adapter-results.json`, `browser/results.json`, screenshots and browser/backend logs |
| `CHROMIUM="$PWD/.scratch/automation/ci-repair/failure-chromium" BROWSER_CHECK_TIMEOUT_SECONDS=8 python scripts/browser_check.py "$PWD/.scratch/automation/ci-repair/evidence/crash-fixture"` (outer `timeout 45s`) | Expected FAIL, exit 1, not outer timeout 124 | `crash-fixture.log`, `crash-fixture-exit.txt` |
| `/proc` and structured-report verification of that fixture run | PASS, exit 0; no live descendants, ignoring child killed | `crash-cleanup-verification.log` |
| `go test '-run=^$' ./...` | PASS, exit 0; separate compile because static short-circuits | `compile.log` |
| `.scratch/automation/bin/actionlint -shellcheck= .github/workflows/validation.yml` | PASS, exit 0; ShellCheck not run | `actionlint.log` |
| `git diff --check` | PASS, exit 0 | `diff-check.log` |

Fuzz targets: FuzzDecodeEscalation, FuzzDecodeProvider, FuzzDecodeRating, FuzzDecodeTurso, FuzzParseImportJSONL, FuzzValidateMessage, FuzzValidateOpenAIEndpoint. Corpus provenance: unchanged working-tree inline source seeds plus existing local Go fuzz cache; hashes in `fuzz-source-sha256.txt`. Most 2s budgets were spent gathering cached baselines; only FuzzValidateMessage reached mutation in the final run (356 total executions, no new interesting cases). No broad mutation-coverage claim, no failing corpus, and no full-fuzz run. No release/high-risk application boundary change is claimed by this tooling repair.

## Reproduction and repair-round history

- Before implementation, `PYTHONPATH=scripts .scratch/automation/venv/bin/python -m unittest test_browser_check.BrowserProcessTests.test_signal_crashed_executable_cleanup_hang_is_bounded` failed on the actual adapter's 25s outer safety limit. The test safety cleanup killed its fixture groups. Log: `evidence/pre-fix-regression.log`. This directly corroborated reviewer P2; no new speculative diagnosis was needed.
- Round 1: group containment and tests added; eight focused tests initially passed, real browser passed, but full unit rerun exposed a timing-dependent hang fixture: Node sometimes completed its own failed-start cleanup within the uniform 8s deadline, producing ordinary failure rather than timeout. Initial results preserved under `evidence/round1/` and `browser-tests-round1.log`.
- Round 2: separated fixed version/build bounds from regression override; hang regression now uses a 1s Node bound, deterministically before Node's inherited 5s startup timeout. Crash case uses 8s to exercise the inherited already-signal-exited cleanup hang; interrupt cases signal after fixture readiness. All final gates except the known static blocker passed. Targeted tooling suite repeated successfully. No further implementation changes after final verification/fingerprinting.

## Remaining blockers and next owners

1. **Independent re-review required / orchestrator:** review this repair diff, process containment, regression evidence and ownership synchronization. This worker does not accept CI-01 or modify central task state. Both repair rounds are exhausted; any further fix requires fresh coordination.
2. **Coordination owner:** static still reports `/tasks/21/evidence`, `/tasks/22/evidence`, `/tasks/23/evidence`, `/tasks/24/evidence` empty for completed DISC-01, RATE-01, RATE-02, RATE-03. Supply genuine evidence or correct status; never invent evidence or weaken schema. Ledger/schema hashes are unchanged. These errors are not waived by local browser/tooling success.
3. **Orchestrator/repository administrator, only when authorized:** hosted GitHub PR/manual workflow, pinned Playwright browser/native dependencies, artifact uploads, and explicit manual full-fuzz branch remain outstanding. No hosted execution or downloads were authorized/performed here. Local system Chromium is not CI's Playwright revision. Required status checks/branch protection remain administrative follow-up.

No release/deployment/production-readiness claim. Orchestrator owns acceptance, central task/goal/blocker/observation updates. Worker stops here.
