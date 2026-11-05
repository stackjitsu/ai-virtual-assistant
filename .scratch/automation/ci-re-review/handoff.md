# CI-01 independent re-review handoff

Date: 2026-11-03
Session: `01a1a086-281b-73b8-94d2-0aa917788776`
Status: independent re-review complete; stopped at this handoff. Coordinator owns acceptance and central state.

## Recommendation

**Accept the bounded P2 containment / P3 ownership repair. Do not mark all CI-01 validation gates complete or authorize merge/release on this evidence.** No new blocking implementation defect was found. The executable failure-path tests independently pass, the original signal-exit hang is contained, and ownership agrees. Static remains blocked by exactly the historical ledger errors. Independent real-browser success remains unavailable in this review environment, and hosted execution remains outstanding and unauthorized.

This distinguishes repair acceptance from full CI-01 completion: the successful owner browser log is not substituted for independent browser evidence, and the ledger failure is not waived.

## Reviewed tree and preservation

- HEAD remained `12532877e0e4f3ed2e4f7a880567c98a6dfce29e`.
- Read the task/repair briefs, original review and repair handoffs, AGENTS, pinned goal, glossary, context map, architecture/testing/fuzzing rules, delegation protocol, ADR 0001, manifest, check adapter/tests/schema, workflow and unchanged Node browser regression.
- Reviewed exact `git diff HEAD --binary` plus full untracked `.github/workflows/validation.yml`, `scripts/browser_check.py`, `scripts/requirements-checks.txt`, `scripts/test_browser_check.py`, and `scripts/test_tasks_schema.py`. The last file is also untracked implementation and was included even though the brief's retained-file summary omits it.
- Captured combined patch: `evidence/implementation.diff`, SHA-256 `ca1a0f3a30c60e1bcfc187b7a403c07c0e300bd01c9cdaac87db5ccb036f8a32`.
- This differs from the repair handoff hash **only in Git's source-prefix rendering**: `git diff HEAD` uses `c/`, whereas the owner's unstaged diff uses `i/`. Independently normalizing those diff-header prefixes produces byte-for-byte owner patch equality and SHA-256 `d8c84629fa929cde3a64c28b88ef94e5226e1dcaa65897aa94eeefb3ed59cb32`. No content difference was hidden by the normalization.
- `evidence/before-sha256.json` and `after-sha256.json` contain per-file SHA-256 fingerprints for all tracked files, the untracked implementation, and pre-existing automation evidence (800 files; environment/cache/tmp excluded). All remained identical. Checks used a read-only mount of source and existing evidence.
- Ledger, schema and trusted-input script are independently byte-identical to HEAD:
  - `.harness/tasks.json`: `db5a50f485556c381782b8ed0a48b14f26df1cd795744c5d36972fe17614274a`
  - `.harness/tasks.schema.json`: `b142430389bd9186238d022ffd7c0ca691ce57bd2a3ebc2271ae34e9fec4d214`
  - `web/tests/sandbox-send.browser.mjs`: `1577b9553d9913ad56a941e6ce07295bc5b948f19f2bcf083d66b57e3c512dd6`

## Hunk-level review / original finding disposition

| File / hunk | Independent assessment |
|---|---|
| `scripts/browser_check.py:17–56`, constants and `cleanup_group` | **P2 fixed.** Finite stage bounds; TERM targets the owned group, not just leader. Leader exit does not suppress escalation against descendants. SIGINT/TERM are ignored during mandatory cleanup; TERM grace and leader reap are bounded. If group no longer exists, no pointless KILL is required. Immediate child is reaped; orphan grandchildren remain init's responsibility. |
| `scripts/browser_check.py:59–91`, `run_owned` | Starts each stage with `start_new_session=True`; no piped-output deadlock. Clears application credentials. Timeout/interruption remains nonzero and runs cleanup in `finally`; records stage, configured timeout, PID, outcome and cleanup in structured evidence. |
| `scripts/browser_check.py:94–139`, entry point/report | Validates finite override in 0.1–300 seconds. Override affects only Node; version/build remain 15/180 seconds. Default Node bound 120 seconds. Adapter JSON is separate from Node JSON and written after failed cleanup path; no fabricated browser success. Limitations for SIGKILL, group escape and kernel waits are documented rather than overclaimed. |
| `scripts/test_browser_check.py`, retained tests and new process tests | Missing-browser, reused-directory and real-script selection/failure assertions retained. Adds invalid override cases and four real executable regressions: crash, hang, SIGINT, SIGTERM. They assert exact failure status/outcome/bound, leader reaping, KILL escalation, group inheritance and absence of live fixture descendants. Crash test additionally requires inherited `Timeout: Chromium startup`; all forbid browser PASS. |
| Round-2 timing change | Hang fixture uses a 1-second Node bound, strictly before unchanged Node's 5-second browser-startup deadline; this removes the round-1 race between ordinary failed cleanup and the 8-second adapter timeout. Build/version are no longer squeezed into that fixture deadline. Crash keeps 8 seconds to reach the already-signal-exited cleanup hang; interruption waits for fixture readiness. No acceptance predicate was relaxed to allow either timeout or ordinary failure. Existing pre-repair assertions are preserved; new final predicates are strict. This is deterministic ordering of the two relevant deadlines, not a guarantee that arbitrarily overloaded hosts start a fixture within one second. Three independent executions of the process suite passed (unit, targeted suite, repeat). |
| `docs/testing/project-checks.md`, containment addition | Bounds, process-group inheritance/escape limits, cleanup/report semantics and separation of fixture evidence from browser acceptance match implementation. Minor wording precision: interruption console message names only the signal; stage and bound are in JSON, unlike timeout's console message. This does not defeat the brief's adapter-owned JSON requirement. |
| `docs/architecture/context-map.md:9` and `harness.json` ownership hunk | **P3 fixed.** Foundation owns `.github/workflows/**` in both; same accountable owner and orchestrator communication path. No runtime/application ownership expansion. |
| `scripts/check.py`, schema/check-routing hunks; `scripts/test_tasks_schema.py`; dependency pins | Full Draft-07 `check_schema`/`iter_errors`, all records inspected, explicit missing-dependency failure. Static fails before compilation on invalid ledger, as documented. Schema tests cover required fields, types/enums/arrays/unknown properties, conditional requirements, later records and nonmutation. No schema/ledger relaxation. Added text suffix checks and `/tmp/opencode` creation are tooling-only. |
| `.github/workflows/validation.yml`, full new file | PR/manual events, seven non-fail-fast jobs, 25-minute bound, read-only token, nonpersisted checkout credentials, pinned action commits/tools, no supplied application credentials. Manual full fuzz explicitly gated. Final status/artifact steps use `always()`; upload allowlist excludes binaries/SQLite/profiles. Hosted behavior and pin availability are not proved locally. |
| Remaining project-checks/manifest hunks | Dependency/schema descriptions, local commands, ADR alignment and tooling-only setup scope agree. Distinguishes mutable runner libraries, system Chromium evidence, branch-protection administration and hosted limitations; no production claim. |

### Standards

Foundation tooling ownership and bounded repair paths are respected. Application/widget code is unchanged. Tests remain real and nonmutating; the historical ledger is not rewritten to make static green. No new blocking standards finding. The interruption console wording above is a nonblocking documentation precision note, not a missing structured-evidence field.

### Spec

P2 and P3 acceptance criteria are satisfied by code inspection and independently reproduced failure-path evidence. No sandbox/CSP bypass, fixture browser-success claim, runtime expansion or unauthorized hosted action was found. Full successful browser/hosted validation is explicitly not accepted here.

## Independent executions

All commands below were actually run by this reviewer, not inferred from owner logs. Evidence paths are relative to this handoff. `run-local.sh` prepends the existing `.scratch/automation/venv/bin`, disables Python bytecode and Go downloads/telemetry, empties application credentials, mounts implementation/existing evidence read-only, maps `/tmp` to reviewer-owned scratch, uses a copied local Go cache and isolates external networking with bubblewrap. Only reviewer scratch is writable. No downloads/installs or environment repairs were performed (0 of maximum 2). No Chromium sandbox bypass was attempted.

Tool versions (`evidence/dependencies.log`): Go 1.26.3, Node 26.1.0, Python 3.14.8, system Chromium 154.0.8037.97. Existing venv dependencies exactly match `requirements-checks.txt`.

For each row except direct Git checking, command prefix is `bash .scratch/automation/ci-re-review/run-local.sh`. Evidence root `E` in commands below expands to `$PWD/.scratch/automation/ci-re-review/evidence`.

| Command after prefix | Exit / result | Evidence |
|---|---|---|
| `timeout 300s make format` | 0, PASS | `evidence/format.log` |
| `timeout 300s make lint` | 0, PASS | `evidence/lint.log` |
| `timeout 300s make static` | 2, exactly four historical violations | `evidence/static.log` |
| `timeout 300s make unit` | 0; 23 Python tests, Go race suite, 62/62 Node tests | `evidence/unit.log` |
| `timeout 300s make contract` | 0; Go contract race suite, 14/14 Node tests | `evidence/contract.log` |
| `timeout 300s make fuzz-smoke` | 0; seven targets, 2s each, 2 workers | `evidence/fuzz-smoke.log` |
| `timeout 120s env PYTHONPATH=scripts python3 -m unittest -v test_browser_check test_check test_tasks_schema` | 0; 23 tests, 25.868s | `evidence/tooling-tests.log` |
| `timeout 90s env PYTHONPATH=scripts python3 -m unittest -v test_browser_check.BrowserProcessTests` | 0; four process tests repeated, 25.031s | `evidence/process-tests-repeat.log` |
| `.scratch/automation/bin/actionlint -shellcheck= .github/workflows/validation.yml` | 0; ShellCheck deliberately not run | `evidence/actionlint.log` |
| `timeout 45s env CHROMIUM="$PWD/.scratch/automation/ci-repair/failure-chromium" BROWSER_CHECK_TIMEOUT_SECONDS=8 python3 scripts/browser_check.py "$E/crash-fixture"` | **1, expected failure**, not outer timeout 124 | `evidence/crash-fixture.log`, `crash-fixture/adapter-results.json` |
| `timeout 345s env CHROMIUM=/usr/bin/chromium python3 scripts/browser_check.py "$E/browser"` | **1, browser startup blocked**, adapter-contained failure, not outer timeout | `evidence/browser.log`, `browser/chromium.log`, `browser/adapter-results.json` |
| `timeout 180s go test '-run=^$' ./...` | 0; compile separately despite static short-circuit | `evidence/compile.log` |
| `python3 .scratch/automation/ci-re-review/verify-evidence.py` | 0; independent `/proc`, JSON, ledger/base and fingerprint assertions | `evidence/independent-verification.log` |
| `git diff --check` (direct, no wrapper) | 0 | `evidence/diff-check.log` |

Machine-readable exit summary: `evidence/results.txt`.

### Crash / cleanup proof

The executable fixture answers version, then launches a readiness-confirmed SIGTERM-ignoring child and signal-exits itself. Unchanged Node logs its Chromium startup timeout and hangs in inherited cleanup. The repaired adapter reports Node timeout at 8 seconds, completes stage including cleanup in **9.019 seconds**, exits 1 before the 45-second outer limit, records TERM and KILL sent and leader reaped. Node `results.json` is absent, while adapter JSON survives. Independent `/proc` verification found fixture parent **9587** and ignoring child **9588** both absent, with no live members of any recorded adapter-owned group. This proves failure containment only.

### Actual browser attempt / environment blocker

System Chromium aborts with `setuid_sandbox_host.cc:172`: SUID helper not configured for this review sandbox. No successful trusted-input assertion count is claimed; Node results are absent. Adapter bounds the inherited hang at 120 seconds, records **121.036 seconds** including cleanup, exits 1 and reaps its leader. TERM removed the group, so `sigkillSent=false` is correct (there was no remaining group to kill). `/proc` verification found no live recorded group members. This independently demonstrates the original real-browser failure is now bounded, but does **not** independently reproduce the owner's 22-assertion success. Coordinator/foundation owns obtaining successful independent browser evidence in an authorized sandbox-capable environment; no implementation defect is inferred from this host restriction.

### Fuzz limits

Targets: FuzzDecodeEscalation, FuzzDecodeProvider, FuzzDecodeRating, FuzzDecodeTurso, FuzzParseImportJSONL, FuzzValidateMessage, FuzzValidateOpenAIEndpoint. Source seeds are unchanged tracked files fingerprinted in the before/after records; corpus also includes the copied existing Go cache. Each 2-second budget remained in baseline gathering. No mutation campaign, full-fuzz, release or high-risk runtime-boundary acceptance is claimed.

## Severity-ranked disposition and required follow-ups

1. **Original P2: closed for bounded repair.** No further containment implementation fix requested. Real crash/hang and interrupt tests pass; external limits did not rescue a hanging adapter.
2. **Original P3: closed.** Context map and manifest ownership agree.
3. **Blocking validation item, pre-existing / coordination owner:** static independently reports only `/tasks/21/evidence` DISC-01, `/tasks/22/evidence` RATE-01, `/tasks/23/evidence` RATE-02 and `/tasks/24/evidence` RATE-03, all completed with `[]`. Independent schema enumeration and HEAD-byte comparison corroborate this, not merely the adapter's log. No new violations, invented evidence or changed ledger/schema bytes. The brief reports DISC-01/RATE-01 evidence restored centrally and RATE-02/03 pending independent acceptance; this reviewer did not inspect or modify the main tree. Coordinator must reconcile genuine central evidence/status and rerun static on the actual integration tree. This is outside CI-01 tooling ownership.
4. **Blocking independent browser-success evidence / coordinator and foundation validator:** actual attempted browser run is not a pass for the environmental reason above. Obtain a successful authorized independent run without sandbox bypass. Do not silently substitute fixture or owner success.
5. **Outstanding, unauthorized hosted follow-ups / coordinator and repository administrator:** run GitHub PR/manual workflow only after authorization; verify pinned Playwright Chromium/native libraries, successful and failed artifact uploads, and explicit manual full-fuzz dispatch. Configure and verify branch protection/required checks separately. Local actionlint, schema tests and system Chromium do not establish these outcomes.

No implementation edits, nested workers, commits, pushes, merges, deployment, credentials/live services, downloads, other-worktree writes or remote writes. Reviewer writes only: claim in the supplied brief and files/cache/tmp/evidence under `.scratch/automation/ci-re-review/`. Central tasks, plan, blockers and observations were deliberately left to the coordinator. Stop here.
