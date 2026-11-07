# RATE-04 recovery — independent validation

Status: claimed; independent recovery validator. Base: 1253287 (rating-integration-clean), not the main dirty tree.

Goal: independently validate the already committed rating integration and report actual evidence. Read AGENTS.md, existing rating spec/issues, docs/contracts/chat.md and testing contracts. Do not implement fixes. Allowed writes: .scratch/automation/** in your own isolated worktree, temporary test output. Never touch other worktrees, user databases, credentials, live services, branches or commits.

Acceptance: inspect rating backend/widget changes and acceptance criteria; run make format lint static unit contract fuzz-smoke and existing relevant browser checks using local fixtures only. Record isolation/replay/revision/restart/concurrent-write and accessible real-input evidence or explicit missing coverage. Keep Escape uncommitted patch out of claims: it is not on this revision. Report missing browser/dependencies honestly. Public build dependency downloads allowed. No production calls, pushes, commits, merge or deployment.

Write .scratch/automation/rating-recovery-handoff.md with exact SHA, test commands/results, coverage gaps and blockers ordered by severity. Any failure goes to owning worker, not a speculative fix. Maximum 30 minutes. Do not mark central ledger complete. Runtime workers are not currently known active; do not wait on historic session claims.
