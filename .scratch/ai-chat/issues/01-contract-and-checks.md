# SETUP-01: contracts and checks

Status: resolved

Claimed by: contract-tooling worker (SETUP-01), 2026-10-20. Scope is contracts, ownership and check tooling only; application tests and runtime remain pending.

Owner: contract-tooling worker. Context: project-foundation.

Task contract, acceptance, permitted paths and invariants: ../../../.harness/tasks.json (SETUP-01) and ../spec.md.

Read repository AGENTS.md and the architecture/testing/delegation rules before editing. Record claim before edits, report exact checks, pending test coverage and handoff. No feature implementation until orchestrator reviews the new baseline. No credentials, provider requests, commits or pushes.

## Review / evidence

Orchestrator reviewed the actual data/HTTP/provider/store/embedding contracts, non-overlapping context ownership and real command runner. Independently ran make setup-check:10toolingtests passed. Feature checks must still run after implementations exist; this setup pass does not mean AI chat works. Go dependency downloads and local SQLite dependency probe are tooling evidence only. Proceeded to CHAT-01/WIDGET-01 in independent owned paths.
