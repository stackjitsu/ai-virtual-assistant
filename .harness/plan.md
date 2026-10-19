# Active Plan

Status: bootstrap; application implementation pending

| Step | Outcome | Owner | Status | Depends on |
|---|---|---|---|---|
| 01 | Adopt reviewed starter and pin application identity and scope | repository-bootstrap worker | completed | |
| 02 | Define first-slice ownership, contracts, and data invariants | orchestrator | pending | 01 |
| 03 | Configure real required validation adapters | owners to be assigned in 02 | pending | 02 |
| 04 | Implement and validate the first AI-chat slice | owners to be assigned in 02 | pending | 03 |

## Decision log

- The manifest remains `template`; application contexts are empty and validation is `unconfigured`.
- Bootstrap file checks do not satisfy the future application's validation gates.
- TDD remains undecided. Durable architectural decisions belong in `docs/adr/` once resolved with the affected owners.

## Next decision

Who owns the first slice's contracts and validation adapters, and should its implementation follow TDD? Resolve these before feature implementation.
