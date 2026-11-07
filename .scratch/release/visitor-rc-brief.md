# REL-01 — Visitor-only local release candidate

Status: pending; worker must claim in its handoff before edits.
Owner: delegated project-foundation release integrator. Coordinator owns approval and central .harness state.
Worktree: /home/me/Documents/agent-worktrees/assistant-visitor-rc
Branch: release/visitor-rc, baseline 65471bea9a414f95329fb998343e551ad2f33a2f.

## Goal and authority
User authorized proceeding with an isolated visitor-only candidate, preserving unfinished work, integrating reviewed CI and validating locally. LOCAL integration commit permitted on this release branch only. No push, main merge, hosted CI, deployment, live inference/database/email, credential access, deletion/stashing/resetting other worktrees. No admin, demo, AUTO imports. Do not change app behavior or remove support UI without separate scoped authorization.

## Owned paths
Mechanical merge of tooling/ci-autonomy commit 84bfd72 only; project-foundation-owned .github/workflows/**, scripts/**, harness.json, docs/architecture/context-map.md, docs/testing/project-checks.md, plus incoming .scratch CI evidence. .harness conflicts retain release first-parent baseline; historical source branch ledger failures must not be patched with invented evidence. Evidence/handoff under .scratch/release/** only. Application paths cmd/** internal/** database/** web/** must remain byte-identical to 65471be.

## Contracts and invariants
Read AGENTS.md, GLOSSARY.md, docs/architecture/context-map.md, docs/testing/testing-contract.md, docs/testing/project-checks.md, docs/testing/fuzzing.md and docs/adr/0001-testing-approach.md. Retain guest access isolation, rating semantics, CSP/sandbox and fail-safe provider behavior. No secrets in evidence or environment dumps. Preserve source/evidence in every other worktree; only inspect other branches through git objects if needed. The orchestrator's dirty .harness is intentionally not imported into product candidate.

## Steps / acceptance
1. Claim task and record baseline git status and source SHA in .scratch/release/handoff.md.
2. Inspect 84bfd72 source diff and workflow for permissions, credential/external service use, time bounds and browser validation. Merge locally into release/visitor-rc; resolve only owned-path conflicts additively; stop and report any unexpected conflict. Never blind ours/theirs for source. No global git config edits.
3. Verify application path diff vs 65471be is empty. Record candidate commit, git diff --stat, exact commands/exits and evidence paths.
4. Run make format lint static unit contract fuzz-smoke. Run make fuzz-full (finite bound, record actual mutation coverage) and documented bounded browser check on local fixtures/actual backend only. Use timeout for all potentially long-running stages, keep CSP/sandbox unchanged, do not install system packages or bypass sandbox. Missing dependencies/infrastructure are explicit blockers, not passes. At most one bounded tooling repair round; any app defect must stop for coordinator assignment.
5. Write final handoff with PASS/FAIL/BLOCKED per check and an honest recommendation. This is owner evidence, not independent acceptance.
6. Produce a concise release gap audit with file:line pointers: live support promise vs delivery, fictional content, provider/database/host prerequisites, privacy/retention/rate/spend controls, admin route exclusion. Do not implement fixes in this task.

## Stop boundary
No production-readiness claim. Hosted CI, live-service configuration, named operator, support promise/content decisions, independent candidate acceptance, and rollback/staging remain outside this local task. Stop after final handoff or a concrete blocker; do not start indefinite polling or more workers.
