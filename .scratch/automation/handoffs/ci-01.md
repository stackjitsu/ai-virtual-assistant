# CI-01 foundation worker handoff

Status: implementation handed off; orchestrator acceptance pending; static gate blocked by existing ledger records. No central completion claim.

## Revision and scope

- Base SHA: `12532877e0e4f3ed2e4f7a880567c98a6dfce29e` (HEAD unchanged).
- Implementation diff SHA-256: `dbea856454a3c5a4d84f1c208e6b445a1de8b714403f5903aed33e4a8abab012`.
- Hash input: `.scratch/automation/evidence/implementation.diff`, consisting of `git diff --binary` for tracked changes followed by `git diff --no-index --binary /dev/null <path>` for the five new implementation files, in the order below. Scratch brief/handoff/logs/tool installations are excluded to avoid self-reference.
- No commits, pushes, nested workers, credentials, live application-service calls or changes to other worktrees. No application/shared ledger/schema/goal edits.

Changed implementation paths:
1. `docs/testing/project-checks.md`
2. `harness.json`
3. `scripts/check.py`
4. `.github/workflows/validation.yml` (new)
5. `scripts/browser_check.py` (new)
6. `scripts/requirements-checks.txt` (new)
7. `scripts/test_browser_check.py` (new)
8. `scripts/test_tasks_schema.py` (new)

Worker-local paths: `.scratch/automation/ci-brief.md` (plan/invariants added), this handoff, `evidence/**`, `venv/**`, `bin/actionlint`. Do not include local tool environments, browser profile/SQLite/binary or evidence wholesale in a commit.

## Delivered

- PR/manual matrix runs format, lint, static, unit, contract, fuzz-smoke and the real trusted-input Chromium test. Independent matrix jobs continue when static fails. Explicit manual `full_fuzz` runs the existing 30-second-per-target command.
- Read-only token, no persisted checkout credentials or secrets, no privileged PR event, 25-minute job timeout. Actions pinned by SHA; Go/Node/Python/Playwright and Python validation dependencies pinned by version. Native runner image and npm transitive dependencies are not fully hermetic.
- SHA/status/log artifacts upload on failure; browser screenshots/results/logs included, SQLite/profile/binary excluded. Runner-loss/cancellation can prevent uploads; GitHub action-internal setup errors remain in job logs.
- Static now validates the entire task ledger with the existing Draft-07 schema and reports all errors. Schema unchanged. Harness manifest checks remain targeted (explicitly documented).
- Negative regressions cover missing required fields/dependency, invalid enums/types/arrays/extra fields, conditional blocked/completed fields, later records and nonmutation. Browser adapter fails on missing executable, reused evidence directory and regression failure.
- Read-only formatting retained, including `gofmt -l`; workflow/requirements text now included. Adapter creates its pre-existing `/tmp/opencode` scratch directory when invoking subprocesses, so fresh CI runners do not depend on local pre-creation.

## Local verification

Environment: Go `go1.26.3 linux/amd64`, Node `v26.1.0`, Python `3.14.8`, system Chromium `154.0.8037.97`. Python dependencies installed only in `.scratch/automation/venv`; public downloads allowed by brief. `actionlint v1.7.7` installed to `.scratch/automation/bin`.

Commands run with `.scratch/automation/venv/bin` prepended to PATH:

| Command | Result | Evidence under `.scratch/automation/evidence/` |
|---|---|---|
| `make format` | PASS, read-only | `format.log` |
| `make lint` | PASS | `lint.log` |
| `make static` | FAIL (make exit 2): four existing ledger violations | `static.log` |
| `make unit` | PASS: 18 tooling regressions, Go race suite, 62 Node tests | `unit.log` |
| `make contract` | PASS: Go contract race suite, 14 Node tests | `contract.log` |
| `make fuzz-smoke` | PASS: seven targets, 2 seconds/target, 2 workers | `fuzz-smoke.log` |
| `python3 scripts/browser_check.py "$PWD/.scratch/automation/evidence/browser"` | PASS: 22 assertions, zero failures | `browser.log`, `browser/results.json`, browser screenshots/backend and Chromium logs |
| `go test -run='^$' ./...` | PASS separately (static stopped before compile) | `compile.log` |
| `.scratch/automation/bin/actionlint -shellcheck= .github/workflows/validation.yml` | PASS (ShellCheck not run) | `actionlint.log` (empty on success) |
| `git diff --check` | PASS | `diff-check.log` (empty on success) |

Final gate exit statuses: `results.txt`. Dependency installation: `dependencies.log`, `actionlint-install.log`. Base SHA: `base-sha.txt`. Patch: `implementation.diff`.

Fuzz targets: FuzzDecodeEscalation, FuzzDecodeProvider, FuzzDecodeRating, FuzzDecodeTurso, FuzzParseImportJSONL, FuzzValidateMessage, FuzzValidateOpenAIEndpoint. Corpus provenance: base revision inline seeds plus existing local Go fuzz cache; source hashes in `fuzz-source-sha256.txt`. Some 2-second budgets were spent gathering cached baseline inputs; passing smoke is not claimed as mutation coverage for every target. No new failure corpus. Full fuzz was not run.

One repair round: actionlint rejected `runner.temp` at job-level env; moved evidence initialization into the first shell step and exported it via GITHUB_ENV. Actionlint then passed. Final six gates rerun after repair/docs; browser adapter/script unchanged since its passing run. No further repair attempted on shared ledger because it is outside scope.

## Blockers / next owner decisions

1. **Coordination owner / orchestrator:** existing `.harness/tasks.json` records 21–24 (`DISC-01`, `RATE-01`, `RATE-02`, `RATE-03`) have `status: completed` but empty `evidence`. Existing schema requires at least one evidence item for completed tasks. `static.log` records all four `/tasks/<index>/evidence` violations. Provide genuine evidence or correct status; do not weaken the schema or add invented evidence. Static and CI will remain red until resolved.
2. **Orchestrator/repository administrator:** actual GitHub execution is unverified (no push/run authorization). CI-provisioned Playwright Chromium, Ubuntu native libraries, pinned setup-tool downloads, artifact uploads, and manual full-fuzz branch were not exercised here. Validate after authorized integration; configure required checks in branch protection separately. Local browser proof uses system Chromium, not CI's downloaded revision.
3. **Orchestrator:** `harness.json` now assigns `.github/workflows/**` to foundation per this dispatch. Coordinate the matching context-map ownership-table update if desired; `docs/architecture/context-map.md` was outside allowed write paths and was not edited.

Not release/deployment evidence; existing production no-go remains outside this task. Orchestrator owns acceptance and central ledger/plan/blocker updates.
