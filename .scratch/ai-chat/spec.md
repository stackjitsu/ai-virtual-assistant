# First AI-chat slice

Status: ready-for-agent

## Goal

Implement the approved first slice, not the whole proposal: User opens an embeddable widget, asks a free-text support question, backend retrieves approved support material and returns an AI-composed grounded answer, and stores an isolated conversation. Include explicit unsupported/provider-failure paths, disclosure and optional minimal host context. No academic-report edits, auth dashboard, live human handoff, WhatsApp, deployment, commits or pushes.

## Coordination / ownership

- SETUP-01 contract/tooling owner: docs/architecture/, docs/contracts/, docs/testing/project-checks.md, go.mod/go.sum, Makefile, scripts/, minimal check scripts/tests and relevant README/.env.example/bootstrap manifest updates. Define exact ownership and contracts before feature workers edit.
- CHAT-01 backend owner: cmd/, internal/ (except an explicitly assigned shared contract package if setup owns it), database/migrations/ and backend fixtures/tests. Coordinate with widget owner through docs/contracts/chat.md.
- WIDGET-01 widget owner: web/ and widget tests, no Go/backend file edits. Use only the documented HTTP/embedding contract; flag required changes for coordination.
- VERIFY-01 validation owner: independent verification; new focused regression/browser tests within declared paths after integration, no silent behavior changes.
- Orchestrator owns .harness/ state and .scratch/ task records, not application code. All work stays within this new app repo and temporary validation directories.

## Setup prerequisites

1. Write compact exact HTTP/data/provider/store/embedding contracts including errors, limits, ordering and credential handling. Define data invariants and workstream owners without creating microservices or hypothetical layers.
2. Configure real format/lint/static/unit/contract/fuzz-smoke/fuzz-full commands. Empty test suites must not be reported as a pass for implemented behavior; record pre-implementation gate vs actual verification honestly. Keep TDD undecided.
3. Minimal dependencies via Go modules are allowed for this build; never install global tooling or send workspace contents to remote AI/services. Use existing Go1.26.3, Node26 and Chromium as available. Do not read/print local credentials.

## Acceptance / invariants

- User can open widget and submit a question without registration; message/history rendering treats all text as data (not HTML).
- AI-first: relevant approved FAQ/onboarding text is retrieved before OpenAI. Approved/withdrawn visibility is enforced server-side. Only bounded allowed context/history and retrieved approved text are sent to the provider.
- Without relevant approved information, return an honest clarification/limitation; no invented support instructions or disguised direct FAQ answer.
- Missing API key/timeouts/malformed or ungrounded provider results produce explicit safe outcomes; no fake successful runtime AI. Mock provider outputs are tests only and clearly identified.
- Server-side credentials; opaque random capability returned only at conversation creation, stored only hashed; conversation ID alone is never authorization. Wrong/missing capability cannot read/write history or invoke a paid provider call for that conversation.
- Parameterized persistence; survive local process restart; atomic completed turns and stable ordering; request IDs/retries cannot duplicate persisted answers. Define a bounded concurrent-write policy and test it.
- Allow only declared optional host context, exclude raw pages/credentials/querystrings and bound all input/history/output/HTTP bodies. Rate limits, timeouts, safe errors and no secret/transcript logging by default.
- Network/browser origin handling is explicit; no wildcard plus credentials, no origin string accepted as authentication. Host integration must not expose assistant tokens outside its own trusted widget interface.
- Real Turso-compatible storage adapter and local SQLite development mode, with isolated tests; no remote DB provisioning/migration/paid inference without approval/credentials. Offline development must be accurately labelled, not called live integration.
- Tests: pure validation/retrieval invariants, persistent adapter restart/rollback, mocked OpenAI/Turso HTTP contract tests, isolated conversation HTTP tests, malformed/oversized/injection inputs, UI state/DOM safety tests, sample-host/loader integration and browser proof if feasible. Relevant Go fuzz targets need properties and seed cases.

## Evidence and reporting

Workers report changed paths, exact checks/results and blockers; do not widen scope. README includes runnable local setup, approved sample knowledge (explicit sample content, not real organization policies), env placeholders, live-vs-local distinction and user data disclosure. Keep full test transcripts in temporary storage, concise facts in task evidence. Do not claim production security, automatic hallucination prevention, live integration, or browser checks without actual evidence.
