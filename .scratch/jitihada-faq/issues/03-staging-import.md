# JITI-03 — Controlled reviewed FAQ import

Status: completed; controlled reviewed import implemented locally, checks passed, no staging write performed.

Owner: chat-backend worker. Allowed paths: `internal/chat/**`, `cmd/assistant/main.go`, and this issue. Do not edit `web/**`, `.harness/**`, README/shared contracts, or unrelated uncommitted work. No live Turso migration/import, database token creation, provider call, deployment, commit or push during implementation.

Dependency: reviewed JITI-01/JITI-02 local corpus and the user's decision that staging import work may continue without consuming the existing 8090/8091 previews. The actual staging write remains a later user-authorized operation.

## Goal and contract

Add an explicit `import-faq` CLI command, with `--file` JSONL and `--allow-remote`, to apply a reviewed corpus through existing SQLite/Turso SQL semantics. It must not reuse the local-only `seed-demo` path or silently approve existing records.

- JSONL: exactly `{"id":"…","title":"…","url":"…","text":"…","approved":true}` per record. Unknown/missing/duplicate field names, wrong types, non-boolean approved values, blank/oversized files, duplicate record IDs, invalid UTF-8 and trailing JSON fail before any database write. All file records are validated in memory before any mutation.
- Existing `Source`/knowledge limits apply: `[a-z0-9][a-z0-9_-]{0,63}` IDs, valid title/text (≤8,000 bytes), approved boolean. URL must either be absent (database NULL) or a valid HTTPS URL without credentials/controls, ≤2,048 bytes. The command must reject any non-HTTPS URL before mutation; it cannot promise the page resolves. At most 100 approved records is a *retrieval* bound: refuse an import that would create a new approved record above that count, not silently drop it.
- Approval semantics: explicit `--approve` permits `approved:true` records to be approved on insert/refresh; `approved:false` is always stored withdrawn. Without `--approve`, any true record fails before mutation. Existing approval flags are never overwritten; an import with true approval can refresh text/title/url for existing IDs only when `--approve` is supplied, while existing false rows stay false. Do not delete or discover unrelated records.
- Atomicity: parameter-bound single-statement SQLite-style `INSERT … ON CONFLICT(id) DO UPDATE SET title=excluded.title,url=excluded.url,text=excluded.text,approved=excluded.approved` is sufficient for SQLite. Add an equivalent Turso adapter behavior; either all validated records commit or no SQL mutation begins. A partial remote/network failure must report unavailable and leave no half-applied import claim.
- Preview is mandatory and cannot be combined with mutation: `import-faq --file X --preview` prints a deterministic bounded JSON summary of what would insert/refresh/no-op/reject (never record text), exits successfully, and performs no write. Remote import requires both `--allow-remote` and a non-preview invocation. SQLite import is local-only and must not use `example.org` URLs as a substitute for a real approved HTTPS source page.
- Existing `serve`, `seed-demo`, and `migrate` behavior, credentials rules, and chat HTTP contract remain byte-compatible. `main` fails early for unknown commands, missing/empty/invalid files, remote-without-allow-remote, preview+mutate ambiguity, and Turso seed/import attempts before network I/O. Remote import never happens automatically on serve/start/migrate.

## Orchestrator review follow-up (required before acceptance)

1. **All-or-nothing write.** The current implementation issues one INSERT per record, so a failure midway leaves a partially applied corpus. `docs/contracts/backend.md` forbids independent BEGIN/INSERT/COMMIT in a Turso pipeline, and the established pattern in this codebase is that **one statement's atomicity is the transaction** (see the completed-turn `INSERT … SELECT`). Change `ImportKnowledge` to apply the whole validated batch in a **single parameterized statement**: one multi-row `INSERT INTO knowledge(id,title,url,text,approved) VALUES(?,?,?,?,?),(?,?,?,?,?)… ON CONFLICT(id) DO UPDATE SET title=excluded.title,url=excluded.url,text=excluded.text`. Duplicate IDs are already rejected at parse time, so the multi-row upsert is safe. Confirm the bound-parameter count stays within SQLite's variable limit for 100 records (500 parameters) and that both adapters behave identically. Remove the now-redundant second approved-cap computation; keep one authoritative check. A remote failure must leave **nothing** applied and report failure honestly.
2. **Automated CLI guard tests.** `cmd/assistant/main_test.go` is now an allowed path for this task. Add real tests (not manual runs) for: `import-faq` without `--file`; `import-faq` with `--seed-demo`; `--allow-remote` with `serve`; unknown command; `import-faq` in turso mode without `--allow-remote` failing **before any network I/O**; `import-faq --preview` producing a plan and writing nothing; `import-faq` in sqlite mode applying a valid file and printing the bounded JSON summary; and the existing serve/seed-demo/migrate behavior staying unchanged. Follow the existing subprocess style already used in that file.
3. Re-run all required checks plus `fuzz-full` and report the results, the exact statement used, and any blocker. Update the handoff with the corrected atomicity semantics and the documented limitation that an import can refresh content but cannot withdraw an existing approved row (withdrawal needs a separate reviewed operation).

## Evidence

Add ordinary Go tests using temp SQLite: valid import/reopen, no-op versus content refresh, preserve existing withdrawal, reject unknown/duplicate/wrong-type/malformed record before mutation, max approved guard, strict preview-no-write, remote-allowed early rejections. Add `TestContract…` coverage for remote/CLI policy as appropriate and extend fuzzable parsing for arbitrary import bytes. Do not claim semantic accuracy or real-page resolution. Run `make format`, `make lint`, `make static`, `make unit`, `make contract`, `fuzz-smoke`; `fuzz-full` before any release/high-risk boundary change. Report files, exact command semantics, and blockers. The staging database must remain empty through this task.

## Worker handoff (2026-10-20, revised after orchestrator review)

Status: implementation complete in the chat-backend context, revised for the review follow-up (all-or-nothing write + automated CLI guard tests). **No staging write was performed**: no Turso contact, no token creation, no remote migration or import, no provider call, no deployment, no commit and no push. The staging database is still empty; applying the reviewed corpus to staging remains a separate, user-authorized operation.

### Files changed

- `internal/chat/import.go` (new) — import records, strict JSONL parsing, plan and apply.
- `internal/chat/import_test.go` (new) — ordinary Go tests, one `TestContract…` fixture test and the `FuzzParseImportJSONL` target.
- `cmd/assistant/main.go` — new `import-faq` command with `--file`, `--preview`, `--approve` and the reused `--allow-remote` flag.
- `cmd/assistant/main_test.go` — new automated CLI guard tests for the import command and the unchanged local seed/migrate behavior.
- `.scratch/jitihada-faq/issues/03-staging-import.md` — this issue only.

No other file was edited. The other uncommitted work in the tree (`web/**`, `README.md`, `.harness/**`, `internal/chat/{http,provider,store,records}*`) is byte-identical to how it was found.

### The exact statement used

The whole validated batch is applied by **one** parameterized statement. Only literal placeholders are used; every value stays a bound parameter:

```sql
INSERT INTO knowledge(id,title,url,text,approved) VALUES (?,?,?,?,?),(?,?,?,?,?),… ON CONFLICT(id) DO UPDATE SET title=excluded.title,url=excluded.url,text=excluded.text
```

- One `(?,?,?,?,?)` tuple per record, in file order: `id`, `title`, `url`, `text`, `approved`. The `approved` value is bound as the integer `1` only when the record asks for approval **and** `--approve` was supplied, otherwise `0`.
- The conflict clause refreshes `title`, `url` and `text` only. The stored approval flag of an existing row is never rewritten by the statement.
- Parameter count: 5 per record, so 5 for one record, 10 for two and **500 for the maximum batch of 100 records**. `TestImportKnowledgeAppliesMaximumBatchInOneStatement` applies the full 100-record batch through the real SQLite adapter, which proves the count stays inside the driver’s variable limit; the statement is built from a fixed placeholder template, so no value is ever interpolated into the SQL text.
- The Turso adapter sends the identical statement with the identical bound values: `TestContractImportKnowledgeOverTursoFixture` asserts the exact statement string, five bound arguments per row, one insert statement per import, and no record value present in the SQL text.

### Corrected atomicity semantics

- The import is now **all-or-nothing for the batch**: one statement is one atomic unit of work, so the validated corpus is applied completely or not at all. The earlier per-record loop was removed.
- `TestImportKnowledgeAppliesWholeBatchAtomically` installs a SQLite trigger that rejects the 51st row of a 60-row batch: the import fails and the corpus still holds **zero** rows. After the trigger is dropped the same batch applies completely (60 rows). A failure in the middle of the batch leaves nothing applied.
- A lost acknowledgement is the one honest exception, and it follows the established completed-turn pattern: after a failed write the stored rows are read back, and if they already hold the state the import would have written, the import is reported as applied. Otherwise the failure is reported (`import failed` / `storage_unavailable`) and nothing is claimed. No partial or half-applied import is ever reported as success.
- Every validation, approval and bound failure still happens **before** the first write, so a rejected import leaves the corpus untouched.

### Approval semantics and the withdrawal limitation

- A new row is stored approved only when the record asks for approval **and** the operator passed `--approve`. A record with `"approved":true` without `--approve` fails the whole import before any write (`import rejected`). `"approved":false` records are always stored withdrawn.
- **An import can refresh content but cannot withdraw an existing approved row.** The conflict clause never rewrites the stored approval flag, so an existing approved row stays approved and an existing withdrawn row stays withdrawn, even when the import refreshes its title, url and text. Withdrawal of an existing approved row needs a separate reviewed operation that this command does not provide.
- Exactly one authoritative approved-bound check remains: existing approved rows plus newly approved rows must not exceed 100. The planner and the applied import share that single computation, and the whole import is refused (never silently trimmed) when the bound would be exceeded. The earlier duplicate cap computation was deleted.

### Exact CLI usage

```
assistant import-faq --file <path> [--preview] [--approve] [--allow-remote]
```

- `--file` is required. Missing file → `import file required`. Unreadable, empty, blank, invalid or oversized file → `import file is invalid`. The file is read and fully validated in memory before the store is touched.
- File format: strict JSONL, one object per line, exactly `{"id":"…","title":"…","url":"…","text":"…","approved":true}`. `url` may be omitted; `id`, `title`, `text` and `approved` are required. Unknown fields, duplicate keys, case-folded keys, missing required keys, wrong types, a non-boolean `approved`, duplicate record IDs, invalid UTF-8, control characters, a blank/empty file, a file above 256 KiB and more than 100 records are all refused before any write. Record bounds are the retrieval bounds: ID `[a-z0-9][a-z0-9_-]{0,63}`, title ≤160 code points/640 bytes, text ≤8,000 bytes, URL empty or a valid HTTPS URL without credentials or controls ≤2,048 bytes.
- SQLite mode (default): local only, `--allow-remote` is not required, and the local schema is migrated first exactly as for any local start. `--seed-demo` stays local-only and is refused in combination with `import-faq`.
- Turso mode: `--allow-remote` is required and the refusal (`remote import requires import-faq --allow-remote`) happens **before any network I/O**. The remote schema is never migrated automatically; run `migrate --allow-remote` first. A remote import never runs automatically on `serve`, `seed-demo` or `migrate`.
- `--preview` plans only: it prints the plan and writes nothing. A preview never mutates and is never combined with a mutation.
- Output is one bounded, deterministic JSON summary on stdout. It contains counts and record IDs only, never record text:

```json
{
  "mode": "preview",
  "plan": {
    "inserted": 1, "updated": 0, "unchanged": 0, "rejected": 1,
    "records": [{"id": "faq-one", "action": "reject"}, {"id": "faq-two", "action": "insert"}]
  }
}
```

```json
{
  "mode": "applied",
  "result": {"inserted": 2, "updated": 0, "unchanged": 0, "rejected": 0}
}
```

- The plan reports one action per record: `insert`, `update`, `unchanged` or `reject`. A record is `reject` when it is invalid, when it needs an approval that was not granted, or when it would exceed the approved bound. The plan never contains record text.

### Evidence

Tests added in `internal/chat/import_test.go`: valid import and reopen of the same database file; insert versus update versus unchanged; existing withdrawal preserved while the content is refreshed; an approved record without `--approve` refused with no rows written; unknown field, duplicate key, missing key, wrong type, non-boolean approved, trailing JSON, invalid UTF-8, control character, insecure URL, invalid ID, blank/oversized text and title, duplicate record IDs, empty/blank file, oversized file and more than 100 records all rejected; the 100-approved bound refuses the whole import without writing; preview performs no write and leaks no record text; **all-or-nothing batch write** (a trigger that rejects the 51st row of a 60-row batch leaves zero rows applied, and the same batch applies completely once the trigger is dropped); **maximum batch in one statement** (100 records, 500 placeholders, applied through the real SQLite adapter, with the placeholder and row counts asserted); a Turso protocol fixture asserting the exact multi-row statement, five bound arguments per row, one insert statement per import and no interpolated record data; `FuzzParseImportJSONL` asserts no panic and that every accepted record satisfies the knowledge rules.

Tests added in `cmd/assistant/main_test.go` (subprocess style of the existing file, no network and no credentials): `import-faq` without `--file` → `import file required`; `import-faq --seed-demo` and `import-faq <positional file>` → `invalid command or flags`; `serve --allow-remote` → `invalid command or flags`; unknown command → `invalid command or flags`; `import-faq` in turso mode without `--allow-remote` → `remote import requires import-faq --allow-remote` with **zero** connections to the remote fixture (no network I/O); `import-faq --preview` prints the bounded plan JSON, prints no record text and leaves the corpus empty; an approved record without `--approve` → `import rejected` with no rows written; `import-faq --approve` applies the file and prints the bounded `applied` summary with the expected counts, and only the approved record is retrievable; `migrate` and `seed-demo` in sqlite mode still succeed with the unchanged message and still leave the ten demo records.

Check results (all run in this repository, sequentially, after the review follow-up):

- `make format` — PASS
- `make lint` — PASS (`go vet ./...`)
- `make static` — PASS
- `make unit` — PASS (`go test -race ./...` plus 55 Node widget unit tests)
- `make contract` — PASS (`TestContract…` Go tests plus 14 Node widget contract tests)
- `make fuzz-smoke` — PASS (5 Go fuzz targets at 2 s each, including `FuzzParseImportJSONL`)
- `make fuzz-full` — PASS (5 Go fuzz targets at 30 s each; run because this adds a new parse-and-write boundary)
- `git diff --check` — clean

The existing `TestContractCLIRemoteMigrationAndSeedGuards`, `TestContractCLILocalSeedNoProviderAndRestart` and `TestContractCLIEndpointOverrideGuard` still pass unchanged: unknown commands, `serve --allow-remote` and the Turso demo-seed guards are refused exactly as before.

### Known limits and notes for the reviewer

1. **Empty URL instead of SQL NULL.** The shared schema `knowledge.url TEXT NOT NULL` (`database/migrations/001_initial.sql`, outside this task's allowed paths) cannot store NULL, so an omitted `url` is stored as an empty string. Retrieval and `validSource` treat an empty URL exactly as the seeded records do. Making it a true NULL needs a schema change by the database/foundation owner.
2. **An import refreshes content but cannot withdraw an existing approved row.** The conflict clause refreshes `title`, `url` and `text` only, so an existing approved row stays approved and an existing withdrawn row stays withdrawn. Withdrawal of an existing approved row needs a separate reviewed operation; this command does not provide one and must not be used to imply one.
3. **Batch atomicity is now one statement, with one honest exception.** The whole batch is one parameterized multi-row statement, so it commits completely or not at all. A lost acknowledgement is resolved by reading the stored rows back: if they already hold the intended state the import is reported as applied, otherwise the failure is reported and nothing is claimed. The adapter contract still forbids `BEGIN`/`COMMIT` batches in a Turso pipeline, so the single statement is the transaction, exactly like the completed-turn `INSERT … SELECT`.
4. **Reading of the approval-refresh sentence.** The brief's "an import with true approval can refresh text/title/url for existing IDs only when `--approve` is supplied" is implemented as: a record carrying `"approved":true` can only be applied with `--approve`, and without `--approve` the whole import fails before any mutation; records that do not ask for approval are stored withdrawn and may still refresh an existing row's title/url/text without ever changing that row's approval flag.
5. **The plan and the applied import can disagree in one direction only.** When the approved bound would be exceeded, the preview marks the records that do not fit as `reject`, while the applied import refuses the whole batch rather than silently dropping records. The plan is advisory; the import is strict.
6. **Not a Turso certification.** The remote test is a local protocol fixture. It proves the statement, the bindings and the single-write behavior; it is not live Turso compatibility evidence.
7. The corpus for a real staging import still needs reviewed, resolving HTTPS FAQ pages (the JITI-01 `example.org` placeholders are not usable citations), and that write needs separate user authorization.
