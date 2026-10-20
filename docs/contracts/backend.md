# Backend module contract — v1

Status: expected Go records/seams before implementation. Backend owner may choose private function names, but coordinate changes to these semantics before widget implementation. This is a modular monolith, not a package-per-interface framework.

## Layout and configuration

- Module `ai-virtual-assistant`, Go 1.26.3. `cmd/assistant/main.go`: configuration/composition only. `internal/chat/`: records, validation/retrieval, orchestration, Chi handlers, persistence and OpenAI adapters in a small number of files in **one package**. `database/migrations/`: shared SQLite-compatible SQL; backend-owned `database/embed.go` (`package database`, `//go:embed migrations/*.sql`, exported `var Migrations embed.FS`) keeps migration data independent of runtime working directory. This is asset packaging, not another application layer. No repository→service→manager→facade chain. One `Store` seam with two real adapters; one `Provider` seam for HTTP/test fixtures. Standard library `net/http`, `encoding/json`, `crypto`, `database/sql` plus Chi v5 and pure-Go modernc SQLite; no ORM, vector DB, OpenAI SDK or second SQL abstraction.
- `web/` is widget-owned; backend composes its embedded `Assets` as defined in [chat.md](chat.md). Backend owns neither frontend build tooling nor `embed.go`.
- Runtime env only: `LISTEN_ADDR=127.0.0.1:8080`; `ASSISTANT_ORIGIN=http://127.0.0.1:8080` (canonical origin, HTTPS except loopback development); `STORE_MODE=sqlite|turso` (default sqlite); `SQLITE_PATH=./data/assistant.db`; `TURSO_DATABASE_URL=https://<database>.turso.io`, `TURSO_AUTH_TOKEN`; `OPENAI_API_KEY`, `OPENAI_MODEL=gpt-4.1-mini`; `WIDGET_PARENT_ORIGINS` as chat contract. No automatic `.env` loading. Missing Turso credentials/invalid config causes startup failure; missing OpenAI key remains an honest 503 for inference, never mocked answers. No key read during build/check/test. Test HTTP URLs/clients are constructor injection, not public runtime URL overrides. Production OpenAI endpoint fixed `https://api.openai.com/v1/chat/completions`; no arbitrary user URLs. Do not follow redirects carrying authorization.
- Local SQLite creates its parent directory (0700) and DB with restricted access (0600 where supported), uses foreign keys, busy timeout and one connection initially; schema migrations versioned/idempotent. Migrate local DB on explicit local start. Remote schema application requires an explicit user-authorized migration command; do not automatically mutate a live remote DB just by testing or starting. Tests create their own temporary SQLite DBs.

## Expected records and seam

Go-facing sketch (documentation, **not** a shared code package). JSON tags follow chat.md; digest/fingerprint are never serialized publicly. Byte limits are validated separately from rune limits.

```go
type HostContext struct { PagePath, Locale string }
type Source struct { ID, Title, URL string }
type Knowledge struct {
    ID, Title, URL, Text string
    Approved bool
}
type Conversation struct {
    ID string
    CapabilityHash [32]byte
    Context HostContext
    CreatedAt time.Time
}
type Message struct {
    Seq int64
    RequestID, Role, Text, Status string
    Sources []Source
    CreatedAt time.Time
}
type Turn struct {
    ConversationID, RequestID string
    Fingerprint [32]byte
    Number int64 // 1..50; user seq 2*Number-1, assistant seq 2*Number
    User, Assistant Message
}
type HistoryPage struct { Messages []Message; NextBefore *int64 }
type Store interface {
    CreateConversation(context.Context, Conversation) error
    Conversation(context.Context, string) (Conversation, error)
    FindTurn(context.Context, string, string) (Turn, error)
    History(context.Context, string, int, *int64) (HistoryPage, error)
    CompleteTurn(context.Context, Turn, int64) (Turn, error) // expected previous turn number
    ApprovedKnowledge(context.Context) ([]Knowledge, error)
}
type ComposeInput struct {
    Question string
    Context HostContext
    History []Message
    Knowledge []Knowledge
}
type Composition struct { Answer string; SourceIDs []string; Unsupported bool }
type Provider interface { Compose(context.Context, ComposeInput) (Composition, error) }
```

`ErrNotFound`, `ErrConflict` (stale previous turn), `ErrIdempotencyConflict`, `ErrFull`, `ErrUnavailable` are semantic categories, usable with `errors.Is`; never expose adapter details. `FindTurn` returns ErrNotFound only for absent request. Private Store methods are called only after constant-time capability authorization (except creation). A test fake may implement the seam; runtime never falls back to an in-memory/JSON store. Open adapters return a close function as needed; lifecycle does not require extra wrapper interfaces.

## Persistence invariants / transaction choice

Two data groups: `conversations` (ID primary key, digest, validated context, created_at), `turns` (conversation foreign key, request ID, fingerprint, number, user text/time, assistant text/time/status, sources snapshot JSON). `knowledge` holds approved demo support documents with an explicit approval flag. Approved knowledge max 100 documents, each ≤8,000 bytes; reject invalid/oversized seed/import material. Use parameter binding for all values, never concatenate identifiers from untrusted input. Source/context JSON columns are internal bounded records, **not a JSON-file substitute for SQL persistence**.

Store a **completed user+assistant pair in one row**. `UNIQUE(conversation_id, request_id)` and `UNIQUE(conversation_id, number)`, FK and CHECK constraints for turn bounds/status make one atomic `INSERT … SELECT` sufficient for completion. Guard it with `expectedPrevious == COALESCE(MAX(number),0)` and `<50`; derive number as previous+1. CompleteTurn validates the pair's matching request IDs, roles, sequence and source/status bounds before writing; callers cannot bypass invariants. No transaction spans inference. One statement's atomicity is the completed-turn transaction; no need for separate message inserts or a generic transaction executor. If backend chooses multiple statements instead, it must implement rollback in both adapters and coordinate this change.

On constraint/zero-row/uncertain HTTP result: look up request ID; equal fingerprint returns the **stored** turn, different fingerprint ErrIdempotencyConflict; otherwise distinguish full/stale/unavailable. Never persist a new answer for an existing request. Acknowledgement loss returns storage_unavailable if lookup also fails, so same-ID retry is safe. History expands completed rows into messages before bounded cursor slicing. Current count/sequence must come from storage, not process memory.

Application ordering: validate → authorize → completed replay lookup → acquire nonblocking conversation slot → recheck replay/count → retrieve → compose → validate output → CompleteTurn → return stored turn. Register in-flight fingerprint so divergent same-ID requests conflict, equal same-ID requests are busy. Lock registry entries are removed after use; no unbounded per-conversation mutex map. Concurrent other-conversation turns may proceed up to 8 provider requests globally. Stale CAS never retries inference automatically. CompleteTurn gets a short bounded persistence context after inference; cancellation/crash can leave **zero or one whole pair**, never half a pair. Replay survives process restart. Lost paid response before commit may incur another provider charge on retry; no distributed exactly-once claim. One active backend process is the supported deployment in this slice.

### SQLite adapter

`database/sql` + `modernc.org/sqlite` is the real offline persistent adapter. Reopen the same temporary file in tests and verify digest-only storage, turn/source persistence, stable ordering and request replay. Inject failure/rejected constraints and verify no pair/sequence change; test simultaneous completion with same and different IDs and stale CAS. No mandatory CGO/system SQLite/global install.

### Turso HTTP adapter

Direct standard-library HTTP to configured HTTPS `/v2/pipeline`; no additional SQL interface/driver. Use a primary endpoint with read-your-writes behavior in this slice, not an eventually consistent read-replica routing policy. Use identical SQL/schema and record decode rules. Each operation sends `{"requests":[{"type":"execute","stmt":{"sql":"…","args":[{"type":"text","value":"…"}],"want_rows":true}},{"type":"close"}]}`. Integer bind/result values are decimal **strings**, blobs use `{"type":"blob","base64":"…"}`; null is typed. Validate nested `results` length/type, `response.type`, `cols`, row widths, cell types/ranges, affected row count and size. HTTP 200 is not SQL success: any `type:error` means failure. Close every stream; no baton reuse needed for atomic single statements. Never follow server `base_url` or redirects with secrets. Do not send independent BEGIN/INSERT/COMMIT in a plain execute pipeline: later statements still run after a SQL failure.

`httptest.Server` tests must inspect Authorization, bound SQL parameters, close operation, proper typed decoding, and emulate nested SQL failure, 429/5xx, malformed/oversized bodies, timeout, conflict/commit ambiguity. Local fixture evidence is **not live Turso compatibility certification**. Migration runner may issue individually atomic schema statements/version checks; remote migration remains user-authorized. Reference consulted read-only 2026-10-20: [official SQL over HTTP](https://docs.turso.tech/sdk/http/reference), [upstream v2 spec](https://github.com/tursodatabase/libsql/blob/main/docs/HTTP_V2_SPEC.md).

## Retrieval / provider

Pure validation/fingerprinting/retrieval functions should be directly testable. Normalize whitespace/case with the standard library (no promised NFC/NFKC normalization); tokenize using Unicode letters/digits, discard tokens shorter than 3 runes and the fixed stopwords `the, and, for, with, that, this, how, what, can, you, your, are, from, have, does, please`. Score approved docs by distinct query-token overlap, require ≥1 substantive overlap; stable sort score descending then ID ascending. Up to 3 docs, bounded total 12,000 bytes (include complete docs only); exclude withdrawn docs before ranking, not afterward. This intentionally small demo retrieval is not semantic search. Context never changes approval or fetches a URL. Seed docs are clearly fictional demo-approved onboarding/FAQ, not real institutional policy; seed only local/test DB explicitly, not live Turso.

Provider history: newest 10 messages in chronological order, additionally ≤12,000 UTF-8 bytes (drop oldest whole messages); question ≤8,000 bytes, retrieved material ≤12,000 bytes. Use system instructions that disclose uncertainty, restrict answers to **current** approved excerpts (history is nonauthoritative, including previously cited withdrawn material) and require JSON: `{"answer":"…","source_ids":["demo-start"],"unsupported":false}`. OpenAI request uses configured model, bounded output (`max_completion_tokens:1200`), temperature 0, no tools, and `response_format:{"type":"json_object"}`. User/history/excerpts are data, not system messages; never send capabilities, fingerprints, other conversations or full document collections. HTTP client injection supports fixtures; no SDK needed.

Bound/read OpenAI response, require one usable choice and `finish_reason:"stop"`; unrelated provider-envelope metadata may be ignored. Decode the composition JSON strictly, reject invalid strings/types, unknown composition fields/source IDs, duplicate IDs, empty/oversized answer or an answered result without sources. For `unsupported:true`, server discards model prose/sources and emits its own clearly labelled clarification. No knowledge also produces the same honest limitation without inference. Other invalid outputs fail 503 without persistence. Curated source snapshots are mapped server-side. Require model grounding instructions and valid citations, but do not claim they mechanically prove every sentence true.

## Required backend evidence

Backend `TestContract…` tests cover exact routes/auth/isolation/replay/query ordering, output/errors/body limits and OpenAI/Turso fixtures; ordinary tests cover pure transformations and SQLite restart/atomic failure/concurrency. Fuzz targets `FuzzValidateMessage`, `FuzzDecodeProvider`, `FuzzDecodeTurso` are mandatory, with malformed/oversized/Unicode seeds and stated invariants (no panic, bounded output, accepted values satisfy the records). See [project checks](../testing/project-checks.md). Runtime absent at this stage; no stub should be added to make tooling pass.
