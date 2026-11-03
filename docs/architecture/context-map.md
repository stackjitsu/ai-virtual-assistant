# Context Map

One support-chat domain, one deployable modular monolith. These are work-ownership contexts, not microservices or competing domain vocabularies. Owners communicate through the orchestrator and the contracts below before changing shared behavior.

## Ownership map

| Context | Owner | Owned paths | Published contract | Collaborators / communication path |
|---|---|---|---|---|
| project-foundation | SETUP-01 contract-tooling worker | `.github/workflows/**`, `docs/architecture/**`, `docs/contracts/**`, `docs/testing/project-checks.md`, `go.mod`, `go.sum`, `Makefile`, `scripts/**`, `harness.json`, `README.md`, `.env.example`, `GLOSSARY.md` | [Backend](../contracts/backend.md), [chat](../contracts/chat.md), [checks](../testing/project-checks.md) | Backend/widget owners request contract or dependency changes via orchestrator; foundation coordinates affected owners |
| chat-backend | CHAT-01 backend owner | `cmd/**`, `internal/**`, `database/**` | [Backend](../contracts/backend.md), [chat](../contracts/chat.md) | Widget consumes HTTP; backend consumes widget-owned embedded assets; orchestrator resolves changes |
| chat-widget | WIDGET-01 widget owner | `web/**` (including `embed.go`, `assets/**`, all widget Node tests) | [Embedding/HTTP](../contracts/chat.md) | Backend imports exported `web.Assets`; widget uses HTTP only, never backend private Go records |
| coordination | Primary orchestrator | `.harness/**`, `.scratch/**` | [Delegation](../agents/delegation-protocol.md) | Owns goal, tasks, claims, blockers and gate review; SETUP-01 may only claim its issue before edits |

VERIFY-01 is a review role, not another implementation context: read all evidence and run checks without overlapping write ownership. New regression/browser-test paths must be explicitly assigned by the orchestrator from the relevant owner's paths before edits. No worker edits the parent academic documents, her-ness source, or unrelated repository policy files. Dependency changes remain foundation-owned and must be requested, not silently made by backend/widget workers.

## Relationships

The compact context definitions below supply purpose, private data, invariants and validation without duplicating five template files per workstream.

| Upstream | Downstream | Contract/data exchanged | Coordination owner |
|---|---|---|---|
| project-foundation | chat-backend, chat-widget | Published v1 contracts, dependencies and seven check commands | contract-tooling worker via orchestrator review |
| chat-backend | chat-widget | Conversation capabilities, bounded messages/history, sources, explicit fallback/errors | backend owner; widget owner acknowledges contract changes |
| chat-widget | chat-backend | Go embedded static assets (`web.Assets`) and browser loader/iframe | widget owner; backend owner acknowledges asset-path changes |
| coordination | all workers | Bounded task briefs, allowed paths and review decision | primary orchestrator |

## Context definitions / data and tests

- **project-foundation** owns contract/check compatibility, not runtime. Private implementation: command runner and manifest checks. Invariants: all seven commands are real; missing application coverage fails honestly; formatting does not mutate source. Verify with `make setup-check`, plus actual gate attempts recorded separately.
- **chat-backend** owns capability digests, SQL conversations/completed turns/approved knowledge, pure validation/retrieval and external HTTP adapters. Private data never crosses widget interface except the documented JSON. [Backend records/invariants](../contracts/backend.md) cover atomic pairs, parameterization, replay, auth-before-inference and bounded ordering. Required evidence: ordinary Go tests, `TestContract…`, mandatory Go fuzz targets, restart/rollback/concurrency and mocked provider/Turso checks. No live secrets needed.
- **chat-widget** owns browser state and assets. It keeps capability only in iframe memory; all text is data; message routing validates source and exact origin. Required evidence: Node ordinary/`*.contract.test.mjs` tests plus browser loader/sample-host verification; see [chat contract](../contracts/chat.md). No direct SQL/provider access or npm framework required.
- **coordination** owns engineering state, not product autonomy. Task records and observations are development artifacts; the app does not spawn agents or maintain a heartbeat. Gate completion means contracts/tooling reviewed, never implicit feature completion.

## Boundary review

- Every context has one accountable owner and a non-overlapping set of owned paths.
- Every dependency names a contract and a communication path.
- No context depends on another context's private data or implementation.
- Proposed boundaries match the real team's or agents' collaboration topology.
