# RATE-02 bounded storage-test follow-up

Status: dispatched; worker must record claim here before editing.
Context/owner: chat-backend / rating backend follow-up worker.
Worktree: /home/me/Documents/agent-worktrees/assistant-rating-review at 1253287.
Goal: close storage evidence gaps found in `.scratch/automation/handoffs/rating-recovery.md` (read the original recovery handoff in this worktree if the copied path is absent).
Allowed writes: internal/chat/*test.go and .scratch/automation/backend-followup/** only; this brief may be updated to claim. No runtime changes without reporting a demonstrated defect and requesting a new boundary.
Read AGENTS.md, pinned goal (historical), GLOSSARY.md, context map, rating spec/issues/contracts, testing-contract.md, fuzzing.md and relevant ADRs before work. This brief is the current coordinator direction.
Acceptance: compare every persisted rating field/timestamp after restart; exact submitted score/comment pairs under distinct concurrent writes and final durable row; capture uncertain committed row before retry and prove replay timestamp unchanged; local Turso fixture concurrent rating writes; direct HTTP rating storage-error 503 mapping with safe body. All fixtures loopback/disposable SQLite only. Preserve chat/escalation behavior and capability isolation.
Checks: make format lint static unit contract fuzz-smoke; targeted rating race repetitions; report exact commands, revision and changed paths. At most two repair rounds, then report blocker. Do not fabricate evidence or weaken assertions.
Handoff: .scratch/automation/backend-followup/handoff.md with acceptance mapping, logs and remaining gaps. No nested workers, other-worktree writes, shared ledger edits, commits, pushes, merges, deployment, credentials or live services. Independent acceptance remains RATE-04 after review; do not mark central completion.
