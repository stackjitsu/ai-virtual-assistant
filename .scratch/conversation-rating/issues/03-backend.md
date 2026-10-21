# RATE-02: durable capability-authorized rating backend

Status: ready-for-agent
Blocked by: 02
Owner: chat-backend worker. Allowed paths: `cmd/**`, `internal/**`, `database/**`, this issue. Depends on RATE-01.

Implement agreed contract in versioned migration and existing SQLite/Turso adapters, atomic one-rating-per-conversation upsert with identical replay, first-turn prerequisite, auth-before-storage, strict request/response validation, rate limiting and safe failures. Exercise concurrency/restart, isolation, null/duplicate/oversized/wrong-type inputs, zero-turn, replacement, failure behavior and Turso loopback fixtures. No live remote migration or credentials. Call tdd skill at suitable boundary; ADR 0001 accepts tests-alongside. Checks: targeted Go tests repeatedly; `make format lint static unit contract fuzz-smoke`, full fuzz before acceptance; `git diff --check`.
