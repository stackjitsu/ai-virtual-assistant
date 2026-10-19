# Delegation Protocol

Every delegated task is a bounded work contract. The orchestrator keeps the goal and global plan; the assigned worker owns implementation and evidence within the task boundary.

## Required task fields

- Stable task ID and status (`pending`, `claimed`, `blocked`, `completed`)
- Goal and observable acceptance criteria
- Owning context and accountable owner
- Allowed paths and relevant contract references
- Data invariants and compatibility constraints
- Required validation commands and expected evidence
- Dependencies, blockers, and next decision if blocked

## Handoff

The orchestrator records the task before dispatch. A worker claims it before editing, reports changed paths and check results, and records blockers rather than widening scope. The orchestrator reviews the result against the task and updates `.harness/tasks.json`, the plan, and blockers. Cross-context work names each affected owner and contract before implementation.
