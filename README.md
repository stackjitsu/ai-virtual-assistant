# AI Virtual Assistant

AI-first support chat with a build-free web widget, approved-knowledge retrieval and isolated, persistent guest conversations. Backend automated checks are passing; integrated review and real-browser validation remain pending. No live OpenAI/Turso integration or production-readiness claim.

## Run locally

Requires Go 1.26.3; development checks also use Node 26, Python 3 and make. No npm install or global tools are needed. From the directory containing this repository:

```sh
cd ai-virtual-assistant
go mod download
env STORE_MODE=sqlite SQLITE_PATH=./data/assistant.db \
  OPENAI_API_KEY= TURSO_DATABASE_URL= TURSO_AUTH_TOKEN= \
  LISTEN_ADDR=127.0.0.1:8080 ASSISTANT_ORIGIN=http://127.0.0.1:8080 \
  WIDGET_PARENT_ORIGINS='http://localhost:8080 http://127.0.0.1:8080' \
  go run ./cmd/assistant serve --seed-demo
```

`go mod download` fetches pinned public dependencies on a fresh clone. The start command explicitly clears service credentials for this run, creates/migrates `./data/assistant.db`, and seeds fictional approved demo knowledge. It does not need an API key or remote database. **Reseeding updates and reapproves both demo documents**; use it deliberately for this local demo, not as normal production startup.

1. Open <http://127.0.0.1:8080/widget/sample-host.html> and open the assistant launcher.
2. Read the disclosure, enter a free-text question, and select **Send question** (or keyboard-submit). This is AI-first chat, not a direct FAQ-answer lookup.
3. With no key, a question matching demo knowledge shows unavailable (`503 provider_unavailable`), not a canned AI answer. **Retry** is explicit; uncertain sends reconcile history before resending the same question ID.
4. Try an unrelated question such as “Explain photosynthesis.” No relevant knowledge produces an honest clarification without inference. Completed turns appear in chat; **Load older messages** appears when more history is available.
5. Reloading creates a new guest conversation: access is kept only in iframe memory. Stored history is not deleted. Stop the server with Ctrl+C.

The seed contains exactly two fictional documents, not real organization policies:

| Document | Sample fact | Question to try |
|---|---|---|
| `demo-start` | Start the demo by opening the sample support widget and asking an onboarding question. | “How do I start demo onboarding?” |
| `demo-support` | The fictional sample support team is available Monday through Friday; no clock times are specified. | “What are the sample support hours?” |

These questions exercise retrieval; an AI-composed answer still requires separately authorized provider configuration and is not guaranteed. Seed source URLs use `example.org`, not a real support service.

## Configuration and normal workflow

[`.env.example`](.env.example) documents defaults/placeholders; **the app does not load `.env` automatically**. Export only chosen settings in your runtime shell. Never commit credentials, put them in browser code, or paste them into chat. Configuring a server-side `OPENAI_API_KEY` and making live requests require separate authorization; requests can incur charges. OpenAI's backend endpoint is fixed; the default `OPENAI_MODEL` is `gpt-4.1-mini`.

- `make run` is `go run ./cmd/assistant`: serves without adding seed data, using your exported configuration. Use the explicit seeded start above first, or `env STORE_MODE=sqlite OPENAI_API_KEY= go run ./cmd/assistant seed-demo` for local seed-only setup. SQLite start applies local migrations; persisted data survives restart.
- Default `STORE_MODE=sqlite` uses `SQLITE_PATH=./data/assistant.db`. Opt-in `STORE_MODE=turso` requires exported `TURSO_DATABASE_URL` and `TURSO_AUTH_TOKEN`; normal remote start does not migrate or seed. Only after explicit user authorization, `go run ./cmd/assistant migrate --allow-remote` applies remote schema changes. Do not run it as a setup/test default. Demo seeding is local-only.
- Default `LISTEN_ADDR=127.0.0.1:8080` and `ASSISTANT_ORIGIN=http://127.0.0.1:8080` must match your chosen address/origin. For another embedding host, explicitly configure `WIDGET_PARENT_ORIGINS`; see [widget integration](web/README.md). Public origins are not authentication.

## Implementation and limits

- `cmd/assistant`: CLI/composition; `internal/chat`: Go/Chi HTTP, validation, lexical retrieval, OpenAI HTTP and SQLite/Turso storage; `database/migrations`: embedded SQL; `web`: embedded JavaScript loader, iframe and sample host. SQLite uses `modernc.org/sqlite`; no ORM, provider SDK or frontend build.
- `/` redirects to the sample host; `/widget/` serves assets. API: `POST /api/v1/conversations`, and authenticated `GET`/`POST /api/v1/conversations/{id}/messages`. Exact shapes/limits: [chat contract](docs/contracts/chat.md), [backend contract](docs/contracts/backend.md).
- Conversation access needs the creation-only bearer capability, not just its ID; only its hash is stored server-side. The iframe keeps it in memory, never browser storage or parent messages. Completed question/answer or clarification pairs persist atomically.
- AI may be wrong. Questions/history go to the configured server; bounded question/history, approved excerpts and optional `page_path`/`locale` go to OpenAI when inference is enabled. The loader does not scrape pages. Do not submit sensitive data or include it in host context; no PII-filtering promise.
- Retrieval is minimal lexical overlap, not semantic search. Valid citations do not prove every sentence true. Missing/invalid provider responses fail safely; no relevant approved knowledge yields clarification.
- One backend process only. Process-local rate/concurrency limits reset on restart and are not production abuse/spend protection. No sign-in/dashboard, live human handoff, WhatsApp or deployment in this slice.

## Validation

```sh
make format
make lint
make static
make unit
make contract
make fuzz-smoke
make fuzz-full
```

The primary session reported format/lint/static/unit/contract/fuzz-smoke passing. The backend handoff additionally reports all seven gates passing, including full fuzz (30 seconds per target, 2 workers), Go race tests, 36 Node tests, and actual CLI subprocess checks for seeded startup, missing-key behavior, restart replay and remote migration/seed guards. OpenAI/Turso use loopback fixtures, **not live integration**. Integrated review and browser acceptance remain pending. `make setup-check` checks tooling separately. See [project checks](docs/testing/project-checks.md) for exact evidence/budgets. TDD remains undecided.

## Development workflow

The copied her-ness starter supplies development coordination, not an autonomous product runtime. Start with [AGENTS.md](AGENTS.md) and the [pinned goal](.harness/goal.md); `.harness/` holds coordination state. Validation expectations live in the [testing contract](docs/testing/testing-contract.md).

Ownership and permitted paths: [context map](docs/architecture/context-map.md). Coordinate shared contract/dependency changes with the foundation owner and orchestrator. Do not edit the parent academic report.
