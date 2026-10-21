# RATE-02: durable capability-authorized rating backend

Status: claimed
Blocked by: 02
Owner: chat-backend worker. Allowed paths: `cmd/**`, `internal/**`, `database/**`, this issue. Depends on RATE-01.

Implement agreed contract in versioned migration and existing SQLite/Turso adapters, atomic one-rating-per-conversation upsert with identical replay, first-turn prerequisite, auth-before-storage, strict request/response validation, rate limiting and safe failures. Exercise concurrency/restart, isolation, null/duplicate/oversized/wrong-type inputs, zero-turn, replacement, failure behavior and Turso loopback fixtures. No live remote migration or credentials. Call tdd skill at suitable boundary; ADR 0001 accepts tests-alongside. Checks: targeted Go tests repeatedly; `make format lint static unit contract fuzz-smoke`, full fuzz before acceptance; `git diff --check`.

## Worker claim

Claimed on rating-backend at integration tip 8fb8e2a. Own only `cmd/**`, `internal/**`, `database/**`, and this issue. Seams: HTTP `POST /api/v1/conversations/{id}/rating` (wire/auth/limits) and `SQLStore` on local SQLite/Turso fixture (atomic durability/restart/failure); HTTP tests cannot alone prove SQL atomicity, and store tests cannot prove HTTP authorization. Invariants and checks are those in the accepted rating contract above; no remote services or migrations.

## Worker evidence

Implemented the one-row guarded upsert and version-3 migration, exact integer JSON validation and bounded direct-peer window. SQLite tests cover zero-turn/clarification, upgrade from v2, rollback, concurrency and restart. HTTP contract tests cover canonical replay/revision, isolation, auth-before-write, Origin and rejected-attempt rate accounting; the Turso loopback covers typed execute/close and uncertain commit with confirmed retry. Local tests ran ten consecutive targeted repetitions. `make format lint static unit contract fuzz-smoke` passes; `make fuzz-full` passed (Go 1.26.3, 30 seconds per target, including FuzzDecodeRating). No remote DB was contacted or migrated. Ready for orchestrator review; issue remains claimed until integration verification.
