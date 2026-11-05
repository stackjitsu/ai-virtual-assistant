# CI-01 independent implementation review handoff

Status: review complete; handed back to orchestrator. No implementation fixes, nested workers, commits, pushes, remote service/credential use, downloads, or other-worktree writes.

## Recommendation

**Hold/reject CI-01 completion acceptance in its current state.** Address the P2 browser failure-path finding below, resolve the four pre-existing ledger violations with genuine evidence or truthful status changes, and retain authorized hosted execution as an external gate. The successful local checks are useful evidence, not GitHub/Playwright, release, or production-readiness proof.

No high/critical implementation-security finding was identified in this bounded review. The ledger failures are not regressions introduced by this patch, and the browser cleanup defect is inherited from an unchanged script; the new adapter exposes it without containing it.

## Reviewed revision, scope and fingerprint

- Base and unchanged HEAD: `12532877e0e4f3ed2e4f7a880567c98a6dfce29e`; no commits since base. Reviewed the existing uncommitted tracked **and untracked** implementation, not merely `git diff HEAD`.
- Independently reconstructed patch SHA-256: `dbea856454a3c5a4d84f1c208e6b445a1de8b714403f5903aed33e4a8abab012`, identical to the worker handoff/evidence.
- Hash input: tracked `git diff --binary` followed by `git diff --no-index --binary /dev/null <new-path>` in the new-file order below. Evidence: `evidence/implementation.diff` relative to this handoff's directory.
- Exact implementation changed paths:
  1. `docs/testing/project-checks.md`
  2. `harness.json`
  3. `scripts/check.py`
  4. `.github/workflows/validation.yml` (new)
  5. `scripts/browser_check.py` (new)
  6. `scripts/requirements-checks.txt` (new)
  7. `scripts/test_browser_check.py` (new)
  8. `scripts/test_tasks_schema.py` (new)
- Read original brief/handoff, root agent rules, historical goal, glossary, context map, architecture/testing/fuzz/delegation rules, ADR 0001, manifest, Makefile, validator/schema/tests, workflow, and the unchanged browser script. Reviewed standards and spec directly; no subagents were spawned.
- Reviewer writes only: claim/status in `.scratch/automation/ci-review-brief.md`, and `.scratch/automation/ci-review/**` (this handoff, `run-local.sh`, `crashing-chromium`, copied Go cache, scratch temporary files, and evidence). No implementation/schema/ledger edits. `evidence/source-sha256.txt` and `source-unchanged.log` confirm all eight implementation files plus ledger/schema/browser script remained unchanged during review.

## Severity-ranked actionable findings

### P2 — New browser adapter does not contain an inherited signal-exit cleanup hang

**Evidence:** `scripts/browser_check.py:28` waits synchronously on the existing Node regression through `scripts/check.py:36–39`, with no subprocess timeout. In unchanged `web/tests/sandbox-send.browser.mjs:214–219`, cleanup only checks `process.exitCode`. For a child terminated by a signal, `exitCode` remains null while `signalCode` is set. Cleanup subscribes to an already-emitted `exit` event and ultimately awaits it forever. This prevents the result write at lines 222–224.

**Independent reproduction:** the real system Chromium aborted inside the review sandbox (SUID sandbox startup error), the test logged `Timeout: Chromium startup`, and the adapter did not exit before the 90-second command limit. This is a reviewer-environment browser startup limitation, not evidence that hosted Chromium will fail. Separately, reviewer-owned `crashing-chromium` answers `--version` successfully and then terminates itself with SIGTERM. Running the unchanged adapter/script against that fixture also logs the startup timeout but does not terminate; an external 18-second timeout exits **124**, with no `results.json`. See `evidence/browser.log`, `browser/chromium.log`, and `browser-crash-fixture.log`. The fixture is solely failure-path evidence, never a substitute for a real browser pass.

**Impact:** an installed but crashing browser can consume the entire 25-minute CI job timeout rather than producing a normal failed check and structured browser result. Job timeout can also interfere with final evidence steps. The workflow still has an overall bound and does not falsely pass; this is a failure-handling/reproducibility defect, not a privilege escalation.

**Next owner:** foundation should bound the adapter's browser subprocess/process group and add an executable-crash/hang regression. If repairing the inherited cleanup instead, obtain widget-owner assignment for the browser-script path and account for both signal termination and already-exited children. Preserve the real trusted-input test and browser sandbox; do not replace it with mocks or disable sandboxing to turn CI green.

### P3 — Ownership documentation has not caught up with the authorized workflow path

`harness.json:15` assigns `.github/workflows/**` to foundation, but `docs/architecture/context-map.md:9` omits that path. Architecture rules require ownership to be visible in the context map. The dispatch explicitly authorized the workflow path, and the worker correctly reported the map as outside its allowed edits; this is a coordination follow-up, not unauthorized worker scope expansion.

**Next owner:** orchestrator should assign the small context-map synchronization to the foundation owner under an explicit allowed-path update.

## Standards assessment

- Ownership boundaries and tests-alongside ADR are respected by the implementation. Formatting remains source-read-only. Schema and ledger were not weakened or normalized.
- Full Draft-07 validation correctly uses `check_schema` and `iter_errors`, validates every record, and reports all four errors. Local schema references resolve without network access. No extra invariants beyond the existing schema are claimed.
- All 18 tooling tests pass, including required fields, types/enums/arrays/additional properties, blocked/completed conditions, later records, nonmutation, missing JSON Schema dependency, missing browser, and simulated regression failure. The missing case is a real child crash/hang (P2).
- Manifest validation is still targeted rather than full-schema, as accurately documented. All seven common manifest checks route to real adapters; browser is a separate explicit workflow/local entry point.

## Spec/security assessment

- PR and manual events are present; no `pull_request_target`, secrets, live-service configuration, privileged runners, or deployment steps. Token is `contents: read`; checkout credentials are not persisted. PR code remains untrusted executable code, but no write-token or secret-bearing trust boundary was found. Administrative approval policies and branch protection are external.
- Seven independent matrix jobs run the six requested make gates plus the real existing Chromium regression. `fail-fast: false` permits other checks to continue when static fails. Shell pipelines propagate nonzero status. Manual full fuzz is explicitly gated by dispatch input and the fuzz matrix entry; it is not implicitly run on PRs.
- Actions are commit-SHA-pinned and tools/Python dependencies/Playwright version-pinned. Browser provisioning selects Playwright's Chromium executable and requires it to exist/be executable. Runner image/native libraries and npm transitives are not fully hermetic; the worker documents this. Action SHA-to-release correspondence and hosted availability were not independently fetched/verified.
- Jobs are bounded to 25 minutes. Evidence setup precedes checkout; tested HEAD is recorded separately from event SHA. Upload uses explicit text/log/JSON/PNG patterns and 14-day retention, excluding SQLite, binaries and Chromium profiles. The browser script saves synthetic fixture requests/screenshots, not capability headers or response bodies. Untrusted PR artifacts are untrusted data; the allowlist is not an arbitrary-code data-loss-prevention boundary. Setup failures leave local SHA/status evidence when the runner survives; cancellation/runner loss/job timeout are not guaranteed uploads.
- Browser uses local SQLite/HTTP, empty provider credentials, actual CDP mouse/keyboard inputs and trusted-event assertions. Its browser traffic suppression/proxy is not general runner egress isolation. Missing executable/dependency cases fail explicitly, independently checked below. Actual hosted Playwright execution and manual full fuzz remain unverified.

## Actual independent check results

Commands used the existing `.scratch/automation/venv` dependencies, Go `1.26.3`, Node `26.1.0`, Python `3.14.8`; attempted system browser was Chromium `154.0.8037.97`. See `evidence/dependencies.log` for exact installed Python pins, which match requirements.

All runs used reviewer-owned `run-local.sh`: bubblewrap read-only root/implementation, writable owned scratch, scratch-backed `/tmp`, copied existing Go build/fuzz cache, isolated network namespace permitting local fixtures, `GOPROXY=off`, `GOSUMDB=off`, `GOTOOLCHAIN=local`, `GOVCS=*:off`, telemetry off, and no bytecode writes. No package downloads were attempted. These are **local isolated runs**, not a hosted-runner emulation.

| Actual command (under review wrapper unless noted) | Result | Evidence under `evidence/` |
|---|---|---|
| `make format` | PASS, exit 0 | `format.log` |
| `make lint` | PASS, exit 0 | `lint.log` |
| `make static` | FAIL, make exit 2; exactly four existing evidence violations | `static.log` |
| `make unit` | PASS, exit 0; 18 Python regressions, Go race suite, 62 Node tests | `unit.log` |
| `make contract` | PASS, exit 0; Go contract race suite, 14 Node tests | `contract.log` |
| `make fuzz-smoke` | PASS, exit 0; seven targets, 2 seconds each, 2 workers | `fuzz-smoke.log` |
| `go test '-run=^$' ./...` | PASS, exit 0; separate compile despite static short-circuit | `compile.log` |
| `.scratch/automation/bin/actionlint -shellcheck= .github/workflows/validation.yml` | PASS, exit 0; ShellCheck not available/run | `actionlint.log` |
| `git diff --check` (direct) | PASS, exit 0 | `diff-check.log` |
| `python3 scripts/check.py tasks-schema` | Expected FAIL, exit 1, same four violations | `tasks-schema.log` |
| `python3 -S scripts/check.py tasks-schema` | Expected FAIL, exit 1, explicit missing-dependency message | `missing-schema-dependency.log` |
| Browser adapter with nonexistent `CHROMIUM` | Expected FAIL, exit 1 | `missing-browser.log` |
| Real system-browser adapter run | NOT PASSED: startup failure, then hang; externally stopped at 90 seconds | `browser.log`, `browser/chromium.log` |
| Adapter with reviewer signal-crash executable fixture, outer `timeout 18s` | Reproduces P2: outer exit 124, no structured result | `browser-crash-fixture.log` |

The initial sandbox omitted a usable `/dev/null`; Python tests passed, but Go failed on device permissions. **One reviewer-setup repair** added a private `/dev`; final results above supersede those setup failures, preserved in `evidence/initial-sandbox/`. The initial combined six-gate command exceeded the 240-second tool limit during fuzz compilation; after no setup/code change, a separate warmed-cache fuzz run completed. No further setup repair or browser sandbox bypass was attempted. No review subprocesses remained afterward.

Fuzz provenance: base/unchanged source seeds plus a copied existing host Go fuzz cache. All seven targets passed, but each 2-second budget was spent gathering cached baseline inputs; **no completed mutation campaign is claimed**. Full fuzz was not run. Worker browser success is separate evidence, not independently reproduced here and not equivalent to CI's Playwright revision.

## Verified blockers and bounded next-owner tasks

1. **Coordination owner:** full independent `Draft7Validator` enumeration found exactly:
   - `/tasks/21/evidence`: `DISC-01`, completed, `[]` — `.harness/tasks.json:321`.
   - `/tasks/22/evidence`: `RATE-01`, completed, `[]` — line 324.
   - `/tasks/23/evidence`: `RATE-02`, completed, `[]` — line 327.
   - `/tasks/24/evidence`: `RATE-03`, completed, `[]` — line 330.
   Schema `.harness/tasks.schema.json:90–92` requires at least one evidence item when completed. Ledger **and schema are byte-identical to base**, verified independently in `evidence/ledger-independent.log`. Supply genuine evidence or correct status; do not invent evidence, waive the condition, or weaken the schema. Rerun static afterward. RATE-04's empty evidence is not an error because it is claimed, not completed.
2. **Foundation owner:** address P2 within a new bounded implementation brief and rerun tooling regressions plus relevant gates; coordinate with widget owner only if touching `web/**`.
3. **Orchestrator/foundation:** synchronize workflow ownership documentation (P3).
4. **Repository administrator/orchestrator, only after explicit authorization:** run the actual GitHub PR/manual workflow, verify pinned setup tools, native browser libraries, real Playwright Chromium trusted-input assertions, failure artifacts, and explicit full-fuzz dispatch. No GitHub execution is authorized or performed by this review. Configure required status checks separately. Local system Chromium or simulated crashes cannot close that gate.

The orchestrator owns central task/goal/blocker/observation updates. This reviewer stops at handoff; existing production no-go remains unchanged.
