# ESC-01: email escalation backend and contract

Status: completed

Owner: backend worker. Context: chat-backend. Pre-read: `AGENTS.md`, `.harness/goal.md`, `.scratch/support-escalation/spec.md`, `docs/contracts/chat.md`, `docs/contracts/backend.md`, `docs/adr/0001-testing-approach.md`, this issue. Allowed edits: `docs/contracts/chat.md` (additive section only), `internal/chat/**`, `database/**`, `.scratch/support-escalation/issues/01-backend.md`, and scratch evidence under `/tmp/opencode/ai-escalation-backend/**`. No `web/**`, no harness shared state, no academic files, no commits or pushes.

## Task

1. Claim this file (Status: claimed) before editing anything else.
2. Extend `docs/contracts/chat.md` with an additive escalation section. Do not modify existing route or field text. The contract must specify:
   - Route: `POST /api/v1/conversations/{id}/escalation` with capability bearer authorization, same origin rules, no query string
   - Request: strict JSON, single object, exact names (`email`, `question_note`, `consent`); email bounded/validated; `consent` must be literal true; note optional and bounded
   - Response: 201 with ticket id, recorded_at, delivery outcome code and visitor-safe message; bounded safe error envelopes for invalid/missing consent, invalid email, rate limit, storage unavailable, unauthorized, conflict
   - Data invariants from the spec, including the honest delivery outcome when no outgoing transport is configured
3. Add `database/migrations/002_escalation.sql` additively (new tables only; never alter existing ones). Migrations must stay idempotent in the same style as `001_initial.sql`.
4. Implement in `internal/chat`: record types with strict owned decoding (reuse the existing strictJSON discipline), store methods for ticket creation and at most bounded listing, HTTP handler with the capability `authorize` seam, a local `EmailOutbox` adapter interface whose test/local implementation records the outcome and never performs network I/O. Real SMTP transport stays out of scope.
5. Rate-limit escalations per conversation and peer, consistent with existing message limits; cap escalations per conversation.
6. Tests alongside (per ADR 0001): behavioral tests incl. invalid consent, invalid email, unknown conversation, wrong capability, duplicate request handling, restart persistence of ticket records, and honest outcome without transport wiring. Race tests over new code with `go test -race ./...`. Fuzz target(s) if a new parsing boundary appears (escalation request decoding). Existing suites stay green; no changes to existing route behavior.
7. Update `.scratch/support-escalation/issues/01-backend.md` with changed paths and evidence; no `web/**` claims.

## Acceptance

- Contract section covers routes, fields, consent, limits and honest outcomes
- Migration additive and idempotent; existing schema untouched
- Handler/store implementation matches contract; capability authorization reused, capabilities never stored or logged raw
- Full local gates pass: `make format lint static unit contract fuzz-smoke`; race clean on new code
- No live email attempts, no live provider/database calls, no commits or pushes

Report changed paths, exact gate outputs, and any blocker with evidence rather than widening scope.

## Completion record (2026-10-20)

### Changed paths (exact)

- `docs/contracts/chat.md` — appended one additive section "Escalation — slice 2 additive section"; no pre-existing text altered
- `database/migrations/002_escalation.sql` — new, additive, idempotent (`CREATE TABLE IF NOT EXISTS escalation_tickets …`, `INSERT OR IGNORE INTO schema_versions(version) VALUES(2)`); no semicolons inside statements so the shared migrate splitter is untouched; `database/embed.go` needed no change (the `migrations/*.sql` glob already embeds it)
- `internal/chat/escalation.go` — new: `Escalation` record, `EscalationRequest`, exact-bounds email/note validation (`ValidateEscalationEmail` / `ValidateEscalationNote`, idempotent-canonical), `decodeEscalationRequest` (consent literal `true` fails closed), `validateEscalation`, `EmailOutbox` interface, `LocalEmailOutbox` recording adapter (zero network I/O, 100-entry bound), slice constants (`delivery_not_wired`, truthful visitor message, body 4096, email ≤254 bytes (local ≤64, domain ≤253), note ≤1,000 code points/2,000 bytes, 5 tickets/conversation, 10 attempts/minute/peer)
- `internal/chat/escalation_test.go` — new: 7 behavioral test functions + `FuzzDecodeEscalation`
- `internal/chat/http.go` — additive: `server.outbox`, dispatch of `NewHandler`/`NewHandlerWithOutbox` (v1 constructor behavior unchanged; existing positional test literals untouched), route `api.Post("/conversations/{id}/escalation", s.escalate)`, `escalate` handler (rate → no-query → strict strictJSON decode → validate → existing `authorize()` seam → dedup replay → store create → outbox observation), `rateEscalation` sharing an extracted `window()` helper with the untouched `rate()` semantics, `rateWindow.escalations` counter, one new `apiError` case for `conflict` (no existing code emits it)
- `internal/chat/records.go` — `Store` interface extended with `CreateEscalation` / `FindEscalation` / `EscalationCount` only
- `internal/chat/store.go` — `escalationColumns`, guarded single-statement `insertEscalationSQL` (same `INSERT … SELECT … WHERE ?=(SELECT COUNT(*)) … RETURNING` shape as the turn CAS), `decodeEscalation`, `FindEscalation` (bounded `LIMIT 2`), `EscalationCount`, `CreateEscalation` (reads decide outcomes; dedup priority over the cap; bounded 8-attempt loop)

Not touched: `web/**`, `.harness/*` shared state (only the pre-existing modifications present at start), `cmd/**`, provider/turso/turn flows, `database/embed.go`, seed-demo.

### Migration / schema description

`escalation_tickets(conversation_id→conversations(id), ticket_id CHECK length 32, email CHECK 6–254 bytes, question_note CHECK 0–2000 bytes, recorded_at, delivery_outcome CHECK ('delivery_not_wired' only — dishonest 'sent' outcomes are impossible at the schema level), ticket_number CHECK 1–5; PK(conversation_id,ticket_id); UNIQUE(conversation_id,email,question_note); UNIQUE(conversation_id,ticket_number))`. The per-conversation cap and duplicate dedup are enforced atomically by these constraints; dedup replays take priority over the cap so capped conversations still return their stored ticket. Idempotent on rerun and restart; the embedded glob applies 001 then 002 in order.

### Gate outputs (all on the final tree; env stripped of live-service credentials by scripts/check.py)

- `make format` → `format: PASS`
- `make lint` → `lint: PASS`
- `make static` → `static: PASS`
- `make unit` → `unit: PASS` (check-adapter Python tests + `go test -race ./...` + Node widget unit tests)
- `make contract` → `contract: PASS` (backend `TestContract…` + Node contract tests)
- `make fuzz-smoke` → `fuzz-smoke: PASS` — 4 targets × 2 s: `FuzzDecodeEscalation` (new, auto-discovered by scripts/check.py), `FuzzDecodeProvider`, `FuzzDecodeTurso`, `FuzzValidateMessage`
- Extra race evidence: `go test -race -count=3 ./internal/chat` → ok (12.7 s); `go test -race ./...` green

Test evidence before/after (internal/chat only): 20 → 27 top-level Go test/fuzz functions; 49 → 95 passing (sub)cases at `-v`. New escalation coverage includes: missing/false/null/string/number consent (fails closed, nothing stored), ~30 invalid-request bodies (email shape/bounds, note bound + control, unknown/duplicate/case-folded keys, trailing JSON, invalid UTF-8, wrong types), 413/415/query/405 handling, identical 401 body across missing/malformed/wrong capability and unknown conversation (no existence oracle), byte-for-byte duplicate replay (also across canonicalization differences), append on new canonical pair, 409 `conflict` at the 5-ticket cap with replay still working when full, chat turns/messages untouched by escalations, honest outcome and observer-failure honesty (201 wording never claims a sent email), local outbox bound, SQLite concurrent cap (3 succeed/5 conflict exactly, atomic), identical concurrent duplicates converging on one ticket, restart reopen returning records byte-equal, and a Turso loopback fixture exercising typed binds/`want_rows`/dedup/end-to-end 201 through the remote adapter.

### Contract decisions beyond the brief (explicit)

1. Duplicates: identical canonical `(email, question_note)` per conversation = idempotent replay of the stored 201 envelope (retry-safe, mirrors message replay); a new canonical pair appends a new ticket; dedup replays work even when the cap is full.
2. Cap exceeded → 409 `conflict` (the brief's code name); rate-limit window stays separate per peer.
3. Escalation limits: 10 attempts/minute per direct peer IP (attempts counted, rejected included); 5 ticket records per conversation, enforced atomically (UNIQUE step number + CHECK in the migration).
4. Bounds: request body ≤4 KiB; email ≤254 bytes (local ≤64, domain ≤253, final alphabetic TLD 2–63); note ≤1,000 code points / ≤2,000 bytes trimmed; all error envelope codes from the v1 table (`conflict` added to the fixed-message map for this route only).
5. `delivery_outcome` value is literally `delivery_not_wired` (the brief's name) and is the only value, pinned by a schema CHECK plus record validation, so a stored ticket can never claim a sent email.
6. The outbox seam is injected via the additive `NewHandlerWithOutbox` constructor; `NewHandler` keeps its v1 signature and defaults to the local recording adapter (zero network I/O) — `cmd/assistant/main.go` needed no change.
7. Ticket IDs: 16 `crypto/rand` bytes hex-32; `recorded_at` RFC3339Nano UTC; tickets live in their own table (append-only) and never consume turn sequences.

### Exact local run command to exercise escalation manually

```
SQLITE_PATH=/tmp/opencode/escalation-demo/assistant.db LISTEN_ADDR=127.0.0.1:8931 go run ./cmd/assistant serve
CID=$(curl -s -X POST http://127.0.0.1:8931/api/v1/conversations -H 'Content-Type: application/json' -d '{}' | python3 -c 'import json,sys;print(json.load(sys.stdin)["conversation_id"])')
CAP=$(curl -s -X POST http://127.0.0.1:8931/api/v1/conversations -H 'Content-Type: application/json' -d '{}' | python3 -c 'import json,sys;print(json.load(sys.stdin)["capability"])')
```

**Note:** the capability is returned only once at conversation creation; capture it by creating the conversation and reading its capability in the same response, e.g.

```
CONV=$(curl -s -X POST http://127.0.0.1:8931/api/v1/conversations -H 'Content-Type: application/json' -d '{}')
CID=$(printf '%s' "$CONV" | python3 -c 'import json,sys;print(json.load(sys.stdin)["conversation_id"])')
CAP=$(printf '%s' "$CONV" | python3 -c 'import json,sys;print(json.load(sys.stdin)["capability"])')
curl -s -X POST "http://127.0.0.1:8931/api/v1/conversations/$CID/escalation" \
  -H "Authorization: Bearer $CAP" -H 'Content-Type: application/json' \
  -d '{"email":"Visitor@Example.COM","question_note":"  Please ring me  ","consent":true}'
```

Verified live on this tree (loopback only): 201 with `delivery_outcome: "delivery_not_wired"` and the truthful message; a resubmit with `email` "visitor@example.com" / note "Please ring me" replays the same `ticket_id`/`recorded_at` byte-for-byte; `consent: false` → 400 `invalid_request`. No live OpenAI/Turso/SMTP calls anywhere; missing `OPENAI_API_KEY` is irrelevant to escalations by design.
