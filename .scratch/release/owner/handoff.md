# REL-01 release integrator handoff

Status: completed — bounded local integration and owner validation PASS; public release NO-GO.
Claim was recorded in this file before integration edits.
Owner: delegated project-foundation release integrator; no recursive delegation.

## Initial state
- Worktree: `/home/me/Documents/agent-worktrees/assistant-visitor-rc`
- Branch: `release/visitor-rc`
- Baseline HEAD: `65471bea9a414f95329fb998343e551ad2f33a2f`
- Source: `84bfd724b645c3ea6a14baad7950f58aca237f48` (`tooling/ci-autonomy` reviewed checkpoint).
- `git status --short` exit 0: only `?? .scratch/release/` (brief and worker log); no tracked modifications.
- Authority: local CI integration/validation only; no application edits, credentials, live services, push, deployment, or other-worktree modifications.

## Integration and preserved boundaries

- Candidate: `8410640e027c732a83dc35d354f8c4a56736be0e` on `release/visitor-rc`.
- First parent: `65471bea9a414f95329fb998343e551ad2f33a2f`; second parent: `84bfd724b645c3ea6a14baad7950f58aca237f48`.
- `timeout 30s git merge --no-ff --no-commit 84bfd724b645c3ea6a14baad7950f58aca237f48`: exit **0**, no conflicts. Evidence: `evidence/merge.log`.
- `timeout 30s git commit -m "Merge reviewed fixture CI into visitor-only release candidate"`: exit **0**. Evidence: `evidence/commit.log`, `evidence/revision.txt`, `evidence/candidate.txt`.
- Source inspection: `git diff HEAD...84bfd72` before merge saved as `evidence/source.diff`; incoming commit changes only the authorized tooling/docs/manifest and CI evidence paths. Workflow has contents-read-only token, nonpersisted checkout credentials, SHA-pinned actions, no supplied secrets, fixture-only application testing, 25-minute job bounds, manual opt-in full fuzz, browser sandbox preserved, and 14-day artifact retention. Public tool/dependency downloads and GitHub artifact upload occur only if hosted workflow is separately executed; not performed here.
- Candidate diff versus baseline: **56 files changed, 5,129 insertions, 10 deletions**. Full `git diff --stat 65471be HEAD`: `evidence/candidate-stat.txt`; changed-path list: `evidence/changed-paths.txt`.
- `git diff --exit-code 65471be HEAD -- cmd internal database web .harness`: exit **0**, zero-byte `evidence/protected-diff.txt`. Application paths are byte-identical, including guest isolation, rating behavior, CSP/sandbox and provider failure handling. `.harness` first-parent state is unchanged. Separate zero-byte `evidence/application.diff` and `evidence/coordination.diff`; application blob IDs: `evidence/application-tree.txt`.
- `git diff --exit-code HEAD 84bfd72 -- .github/workflows scripts harness.json docs/architecture/context-map.md docs/testing/project-checks.md`: exit **0**, `evidence/incoming-tooling.diff` empty: tooling matches reviewed source exactly.
- A supplementary whole `.scratch/automation` comparison against source returned **1**, because release already carries additional historical automation evidence. This is not a merge conflict or unintended import; first-parent historical evidence was preserved. No historical source ledger failures were patched, no fabricated evidence was added, and no schema was weakened.
- `git diff --check 65471be HEAD`: exit **2**, solely whitespace in imported historical `.scratch/automation/ci-repair/evidence/repair.diff`; evidence retained unchanged (`evidence/diff-check.log`). Scoped tooling/docs diff-check: exit **0**, `evidence/tooling-diff-check.log`. Required `make format` passes because `.scratch` evidence is deliberately excluded; the supplemental whole-tree whitespace diagnostic is **FAIL (historical evidence only)**, not a claimed clean result.
- No application implementation or tooling repair edits were needed. No admin/demo/AUTO branches imported. Existing optional fictional seed/sample-host code remains unchanged (see gap audit). No push/main merge/hosted CI/deployment, credential-file reads, live application calls, recursive delegation, global git configuration, or other-worktree edits performed.

## Environment setup and exact validation commands

Run from the assigned worktree. `E=$PWD/.scratch/release/evidence`.
Reproducible driver: `.scratch/release/run-checks.sh`; commands: `evidence/commands.txt`; exits: `evidence/results.tsv`. Overall `timeout 1500s bash .scratch/release/run-checks.sh` exited **0**; the driver records every stage exit explicitly.

Initial discovery found no `chromium` on PATH (version attempt exit **127**) and no system Python `jsonschema` (import exit **1**), recorded in `evidence/tools.log`; these were resolved before checks, not waived:

- `timeout 30s python3 -m venv .scratch/release/checks-venv`: **0**, `evidence/venv.log`.
- `timeout 120s .scratch/release/checks-venv/bin/python3 -m pip install -r scripts/requirements-checks.txt`: **0**, `evidence/dependencies.log`. Only pinned validation packages installed in task-local virtualenv; no system packages or source changes. Installed versions: `evidence/python-packages.txt`.
- Existing browser discovered at `/home/me/.cache/ms-playwright/chromium-1248/chrome-linux64/chrome`; `timeout 15s <that-path> --version`: **0**, Google Chrome for Testing **156.0.8078.4**. Nothing installed/downloaded for browser setup, no sandbox bypass.
- Toolchain: Go **go1.26.3 linux/amd64**, Node **v26.1.0**, Python **3.14.8**, GNU Make **4.4.1**.

Driver exports task virtualenv first in PATH and explicitly empties `OPENAI_API_KEY`, `TURSO_AUTH_TOKEN`, `TURSO_DATABASE_URL` without printing their original values. Native adapters also empty these for child checks. Local HTTP/SQLite/provider fixtures only; browser uses actual backend with empty AI key and local seed, with background networking disabled/proxied to a closed local port.

| Check | Exact command (with above PATH/environment) | Exit | Result / evidence |
|---|---|---:|---|
| format | `timeout 180s make format` | 0 | **PASS**, `evidence/format.log` |
| lint | `timeout 180s make lint` | 0 | **PASS**, `evidence/lint.log` |
| static | `timeout 180s make static` | 0 | **PASS**, includes unchanged release task ledger schema and compile check, `evidence/static.log` |
| unit | `timeout 240s make unit` | 0 | **PASS**, 23 Python regressions, Go race suite, 66 Node tests, `evidence/unit.log` |
| contract | `timeout 180s make contract` | 0 | **PASS**, Go boundary race tests and 15 Node tests, `evidence/contract.log` |
| fuzz-smoke | `timeout 180s make fuzz-smoke` | 0 | **PASS**, seven targets at 2s each / 2 workers; baseline-cache caveat below; `evidence/fuzz-smoke.log` |
| fuzz-full | `timeout 300s make fuzz-full` | 0 | **PASS**, seven targets at 30s each / 2 workers, actual mutations below; `evidence/fuzz-full.log` |
| browser | `timeout 345s env CHROMIUM=/home/me/.cache/ms-playwright/chromium-1248/chrome-linux64/chrome python3 scripts/browser_check.py "$E/browser"` | 0 | **PASS**, actual Go backend/SQLite, 48/48 assertions; `evidence/browser.log`, `evidence/browser/results.json`, `evidence/browser/adapter-results.json`, `evidence/browser-summary.json` |

Browser internal bounds: 15s version / 180s build / 120s trusted-input regression; recorded durations 0.165s / 1.269s / 10.879s, all stages passed and leader reaped. Trusted click, Enter/Space/Ctrl/Meta-Enter, explicit retry/missing-key 503/history reconciliation, restricted CSP/exact iframe sandbox, close/Escape focus and draft retention were exercised. This is the documented browser suite, not a new exhaustive rating/browser/security acceptance campaign and not the CI-pinned hosted browser revision. Browser SQLite, compiled executable and temporary browser profile remain local test artifacts under the evidence directory, not release artifacts or production data.

## Actual fuzz mutation coverage

Package: `ai-virtual-assistant/internal/chat`. Native form: `go test <package> -run=^$ -fuzz=^<target>$ -fuzztime=30s -parallel=2`. All seven discovered targets reached **now fuzzing** in full campaign; no failures/minimized failure corpus reported. Go's execution totals include baseline seeds; subtracting each logged baseline gives the non-baseline execution count below, not a code/branch coverage percentage.

| Target | Baseline cache seeds | Full executions | Non-baseline executions | Newly interesting |
|---|---:|---:|---:|---:|
| FuzzDecodeEscalation | 141 | 30,052 | 29,911 | 10 |
| FuzzDecodeProvider | 143 | 83,669 | 83,526 | 6 |
| FuzzDecodeRating | 162 | 90,695 | 90,533 | 10 |
| FuzzDecodeTurso | 149 | 23,552 | 23,403 | 12 |
| FuzzParseImportJSONL | 104 | 27,044 | 26,940 | 17 |
| FuzzValidateMessage | 96 | 3,794 | 3,698 | 2 |
| FuzzValidateOpenAIEndpoint | 174 | 105,142 | 104,968 | 11 |

Smoke campaign spent most target budgets gathering existing cache seeds. Only `FuzzParseImportJSONL` explicitly logged mutation phase (116 total executions, 104 baseline); do not infer mutation coverage for the other smoke targets from exit 0. Full campaign supplies actual mutation evidence for every target. Source/inline seed revision is exact candidate above; test source SHA-256 values: `evidence/fuzz-source-sha256.txt`; discovered target references: `evidence/fuzz-targets.txt`. Existing Go cache supplied extra seeds and was not snapshotted as a portable corpus; counts and generated-interesting additions are logged, so this is not a claim of identical future generated inputs.

## Recommendation, blockers, and stop

- Bounded owner task: **PASS / completed**; no unresolved local check blocker or tooling repair needed.
- Release gap audit with source pointers: `.scratch/release/gap-audit.md`.
- Public release: **NO-GO**. Hosted CI/branch protection, independent acceptance of this exact candidate, live-service and host configuration, actual support-delivery promise/content decisions, named operator, privacy/retention/abuse/spend policy, staging and rollback remain **BLOCKED / not attempted in this scope**. Coordinator owns assigning product/backend/operator follow-ups; this handoff is owner evidence only, not independent approval.
- Latest recorded status: only `?? .scratch/release/` (`evidence/final-status.txt`); candidate merge committed, no tracked edits. Task-local handoff/evidence/venv remain untracked deliberately; no broad add or evidence commit.
- Stop here. Coordinator must review before advancing; no additional work or polling started.
