# CHAT-01: backend

Status: resolved

Owner: backend worker. Context: chat-backend.

The reviewed v1 baseline is docs/contracts/backend.md and docs/contracts/chat.md. Task fields and validation expectations: .harness/tasks.json CHAT-01 and .scratch/ai-chat/spec.md. Own cmd/, internal/, database/ only. Claim this issue before edits. Ask orchestrator for any dependency/shared-contract change; widget owner supplies web.Assets independently. Report actual local adapter/HTTP/fuzz results; live OpenAI/Turso are not authorized or verified.

## Implementation brief

Claimed by CHAT-01 backend worker, 2026-10-20. Implement one compact internal/chat package, SQL completed-turn CAS, persistent SQLite and typed Turso HTTP adapters, approved-only retrieval and bounded OpenAI composition, capability-authorized Chi HTTP handlers. Allowed paths: cmd/**, internal/**, database/** and this issue only. Invariants: digest-only capability storage, auth before retrieval/replay/inference, exact stored replay, atomic whole turns, bounded inputs/history/concurrency, no live remote calls or secret reads during checks. Acceptance: normative v1 routes and failure paths, explicit local fictional seed workflow, ordinary/contract/race tests and meaningful mandatory fuzz targets. Required checks: gofmt, vet/compile/tests/race, contract and smoke/full fuzz; full-package build depends on independently owned web.Assets.

## Bounded resume review brief

Resume persisted implementation, without architectural/dependency changes. Review normative contracts against actual source/tests; fix concrete exact-JSON-name/duplicate-key bugs and validate presence-sensitive optional context strings without making optional fields required. Add regressions for rejected inputs and current knowledge versus nonauthoritative historical citations. Exercise local CLI/no-key startup and remote migration guard using subprocesses and loopback fixtures only. Re-run required gates, race tests and all three fuzz targets for 30 seconds each. No live credentials, provisioning, provider requests, frontend/docs/tooling edits, commits or pushes. Deliver run/seed hints and README requests to foundation owner in temporary handoff.

## Answer / review evidence — 2026-10-20

Backend implementation and bounded review complete; ready for primary integration review. Persisted architecture retained. Resume changes: exact JSON field-name and duplicate-key rejection (including escaped duplicates); supplied empty context strings rejected while context/fields stay optional; non-UTF-8 declared charsets rejected. Regressions cover missing required message/composition fields, historical withdrawn citations versus current approved excerpts, and actual CLI subprocess behavior.

- `env OPENAI_API_KEY= TURSO_AUTH_TOKEN= TURSO_DATABASE_URL= go test -race -count=1 -v ./cmd/... ./internal/... ./database/...`: PASS, 20 top-level Test functions (19 behavioral + 1 subprocess helper), 38 subtests, 16 fuzz seeds. Includes 14 `TestContract…` functions. SQLite reopen/digest-only persistence/source snapshots, rejected atomic insert/CAS/concurrency, HTTP auth-before-private/paid work/isolation/exact replay/pagination, provider/Turso safe fixtures and ambiguity recovery all pass.
- CLI tests verify `serve --seed-demo`, embedded sample host, relevant question with empty key → honest 503/no half-turn, unsupported question → completed clarification, process restart → byte-identical replay. Remote migration without `--allow-remote`, remote demo seeding, inappropriate flags and unknown command fail without any remote connection.
- `make format lint static unit contract fuzz-smoke`: PASS. Unit/contract include Go race, 10 Python check-adapter regressions and 36 Node tests (24 ordinary + 12 contract).
- `make fuzz-full`: PASS, Go 1.26.3, 2 workers, 30 s each: `FuzzDecodeProvider` 119,499 executions; `FuzzDecodeTurso` 23,344; `FuzzValidateMessage` 4,416. No failure corpus produced. Smoke passed too; provider smoke spent its 2 s on the concurrently expanded cached baseline, so full-run counts are the meaningful mutation evidence.
- No live OpenAI/Turso verification, credential reads, provider charges, provisioning, commits/pushes, frontend edits or browser claims. Browser review remains primary-owned; local backend evidence has no known blocker. Foundation must replace stale scaffold-only README wording and add the run/seed/remote-command instructions supplied in the handoff.

Handoff, changed paths, exact test-name list, seed corpus hashes, runnable local instructions and limitations: `/tmp/opencode/ai-chat-backend-resume/HANDOFF.md`. Logs: `/tmp/opencode/ai-chat-backend-tests.log`, `/tmp/opencode/ai-chat-backend-gates.log`, `/tmp/opencode/ai-chat-backend-fuzz-full.log`. No shared task ledger edits.
