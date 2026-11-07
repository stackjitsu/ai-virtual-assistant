# VERIFY-01 — independent visitor candidate validation

Status: COMPLETE; worker stopping after handoff.

## Verdict

**ACCEPT — local integration only.** Bounded follow-up independently reviewed completed OWNER full-fuzz evidence and rechecked both worktrees' source integrity, closing the sole pending local evidence blocker. All independently executed checks below passed; no reproducible candidate defect was identified. Full fuzz was **OWNER execution independently reviewed, not independently rerun**. This verdict is not production readiness or permission to publish/deploy; all production blockers below remain.

## Exact candidate and integrity

- Tested HEAD: `8410640e027c732a83dc35d354f8c4a56736be0e` (`release/visitor-rc` candidate).
- Baseline: `65471bea9a414f95329fb998343e551ad2f33a2f`.
- Required CI ancestor: `84bfd724b645c3ea6a14baad7950f58aca237f48`.
- Both `git merge-base --is-ancestor <revision> HEAD` checks returned 0. `git diff --exit-code 65471be HEAD -- cmd internal database web .harness` returned 0. All five Git tree object IDs match exactly; see `identity.json`.
- Baseline/post-run SHA-256 manifests (`before-sha256.json`, `after-sha256.json`) cover all 86 tracked non-`.scratch` files, including application, tests, workflow, docs and `.harness`: **identical**. Initial and final `git status --short` show only untracked `.scratch/release-independent/`; final source diff against HEAD is empty and HEAD is unchanged.
- No source, tests, CI, ledger or docs were edited; no commits, recursive delegation, remote writes, deployments, live inference/storage/email, installations or security bypasses were performed. Writes in the repository were evidence-only under this directory. Native tools used their existing temporary/cache locations; no other worktree was written.

## Fresh independently executed checks

Exact commands, outer bounds, exits and elapsed times: `results.json`; reproducible environment/process-group runner: `run.py`; console record: `execution.log`. Logs are `<check>.log` relative to this handoff.

| Check | Outer bound | Exit/result | Elapsed |
|---|---:|---|---:|
| `make format` | 90s | 0 PASS, nonmutating | 0.468s |
| `make lint` | 180s | 0 PASS | 3.710s |
| `make static` | 180s | 0 PASS, including actual unchanged task ledger schema | 7.456s |
| `make unit` | 300s | 0 PASS, Python tooling, Go race, 66 Node tests | 54.220s |
| `make contract` | 180s | 0 PASS, Go race contracts and Node contracts | 16.490s |
| `make fuzz-smoke` | 180s | 0 PASS, seven targets | 39.265s |
| Trusted browser adapter | 345s | 0 PASS, 48/48 assertions | 17.778s |
| Rating browser suite | 150s | 0 PASS, 61/61 assertions | 10.960s |
| Visual/interaction browser suite | 150s | 0 PASS, 208/208 assertions | 13.998s |

No outer timeout fired. Expected negative-fixture `browser: FAIL` lines inside successful unit logs are asserted containment tests, not actual acceptance-suite failures.

Tools: Go `go1.26.3 linux/amd64`, Node `v26.1.0`, Python `3.14.8`, existing `jsonschema 4.26.0`. Default Python initially could not import jsonschema. Reused the existing installation read-only through `PYTHONPATH=/home/me/Documents/agent-worktrees/assistant-ci/.scratch/automation/venv/lib/python3.14/site-packages`, with bytecode disabled. No dependency installation or global setup. Go downloads/telemetry disabled; application credentials emptied.

### Browser evidence

Read suite invocation/options and local-fixture setup before execution. Existing executable: `/home/me/.cache/ms-playwright/chromium-1248/chrome-linux64/chrome`, actual `Chrome/156.0.8078.4`. This is **not** the workflow's pinned Playwright browser revision.

Commands (absolute evidence/binary paths are recorded in `results.json`):

```sh
CHROMIUM=<existing-chrome> python3 scripts/browser_check.py <fresh trusted-browser directory>
CHROMIUM=<existing-chrome> node web/tests/rating.browser.mjs <trusted-browser/assistant> <fresh rating-browser directory>
CHROMIUM=<existing-chrome> node web/tests/widget-visual.browser.mjs <trusted-browser/assistant> <fresh visual-browser directory>
```

Adapter built the real candidate backend, with seeded local SQLite and loopback HTTP hosts, empty AI/Turso credentials, original CSP and unchanged sandbox. `trusted-browser/adapter-results.json` records version/build/regression success and reaped leaders; `trusted-browser/results.json`, `rating-browser/results.json`, and `visual-browser/results.json` hold assertion evidence. Screenshots and fixture logs are adjacent. Rating checks include trusted input and actual backend restart. Browser traffic uses the suites' existing closed-loopback proxy/background-traffic restrictions. Automated visual-suite success is not a new human design approval or staged HTTPS proof.

### Fuzz scope and limitations

Independently ran only smoke: Go 1.26.3, exact candidate test sources (hashes in integrity manifests), existing shared Go cache plus inline seeds; two seconds per target, two workers, anchored selectors. Seven targets: `FuzzDecodeEscalation`, `FuzzDecodeProvider`, `FuzzDecodeRating`, `FuzzDecodeTurso`, `FuzzParseImportJSONL`, `FuzzValidateMessage`, `FuzzValidateOpenAIEndpoint`.

**All seven smoke budgets expired during cached baseline loading; no independent mutation count or complete cache replay is claimed.** Command success is bounded smoke evidence, not extended fuzz assurance. No failing/minimized corpus was produced.

## OWNER full-fuzz evidence — explicitly not independent execution

Read-only source: `/home/me/Documents/agent-worktrees/assistant-visitor-rc/.scratch/release/evidence/`. Snapshot metadata/time/hashes: `owner-evidence.json`; copied read-only observations: `owner-candidate.txt`, `owner-tools.log`, `owner-fuzz-targets.txt`, `owner-fuzz-full.log`, `owner-results.tsv`.

The original inspection found the campaign unfinished; that historical partial snapshot remains preserved above. The user-authorized bounded follow-up reviewed ONLY the owner's final `fuzz-full.log`, `results.tsv`, `candidate.txt` and current Git/source integrity. No campaigns were rerun and no further owner artifacts were consulted.

Final evidence copies: `final-owner-fuzz-full.log`, `final-owner-results.tsv`, `final-owner-candidate.txt`; source checks, observation time and evidence SHA-256 values: `followup-integrity.json`. Candidate file and both current HEADs are exactly `8410640e027c732a83dc35d354f8c4a56736be0e`. Both worktrees' 86 tracked non-evidence source hashes match this validator's original before-run manifest; source diffs against HEAD and application/`.harness` baseline diffs are empty. Git status contains only each worktree's untracked evidence directory.

Final owner log contains seven anchored 30-second/two-worker campaigns, each reaching mutation execution and ending PASS; overall `fuzz-full: PASS` is present and `results.tsv` records `fuzz-full` exit **0**. Reported final executions (including baseline):

| Target | OWNER executions | Outcome |
|---|---:|---|
| FuzzDecodeEscalation | 30,052 | PASS |
| FuzzDecodeProvider | 83,669 | PASS |
| FuzzDecodeRating | 90,695 | PASS |
| FuzzDecodeTurso | 23,552 | PASS |
| FuzzParseImportJSONL | 27,044 | PASS |
| FuzzValidateMessage | 3,794 | PASS |
| FuzzValidateOpenAIEndpoint | 105,142 | PASS |

Some targets show extended zero-throughput intervals; the log nevertheless records bounded successful completion, not a timeout or failing reproducer. This closes the sole pending local full-fuzz evidence blocker. All extended-fuzz results remain **OWNER execution independently reviewed**, never VERIFY-01 execution. Other owner result rows were not used to replace this validator's own fresh checks. No duplicate campaign, long polling, source changes or remote/live operations were performed.

## Review and findings

Reviewed the complete nine-file non-evidence diff from 65471be, saved as `source.diff`; detailed review: `review.md`. Workflow uses read-only contents permission, pinned action SHAs, no persisted checkout credentials/secrets, bounded jobs, explicit optional manual extended fuzz, required check/browser matrix, SHA/status artifacts, and no publication step. Schema enforcement, isolated tooling regressions, browser descendant cleanup, ownership docs and manifest remain compatible with the unchanged application tree. Hosted execution, browser native libraries and artifact upload behavior were not proved by source review.

**No reproducible source defect found; no repair made.** Therefore there are no invented severity/file:line defect claims. The original extended-fuzz evidence blocker under `docs/testing/testing-contract.md:20` and REL-02 acceptance item 4 is now **closed** by the bounded final-evidence and source-integrity review recorded above. No local integration blocker remains within this task's scope. Production exclusions below are unchanged. Worker stops after this updated handoff.

## Explicit remaining release blockers / exclusions

Hosted CI and pinned hosted browser execution; live-provider/storage integration; staged HTTPS/embed/security verification; real support delivery and truthful delivery wording; approved production content; and privacy, retention and operational controls remain **unvalidated release blockers**. Local fixture checks do not close them. No publishing, production-readiness or deployment authorization is implied.
