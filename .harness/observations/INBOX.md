# Observation Inbox

Append atomic, factual observations using the schema in `docs/agents/observational-memory.md`. Keep each observation independently meaningful and include a timestamp when it helps establish ordering or recency.

- id: OBS-20261020-001
  observed_at: 2026-10-20
  source: user interaction
  context: first AI-chat slice
  fact: User enabled Build after agreeing to implement the AI-first approved-knowledge chat slice; TDD was not selected and live model/database requests were not authorized.
  evidence: .scratch/ai-chat/spec.md
- id: OBS-20261020-002
  observed_at: 2026-10-20
  source: tool result
  context: SETUP-01
  fact: First-slice contracts and seven executable check commands are defined; independently run setup checks passed10toolingtests, while runtime feature tests remain pending implementation.
  evidence: docs/contracts/chat.md; docs/contracts/backend.md; scripts/test_check.py
- id: OBS-20261020-003
  observed_at: 2026-10-20
  source: tool result
  context: CHAT-01
  fact: Persisted backend/widget code passed independently run format, lint, static, unit, contract and fuzz-smoke commands; backend review subsequently passed race tests and30-second full fuzz runs with actual mutations.
  evidence: .scratch/ai-chat/issues/02-backend.md; /tmp/opencode/ai-chat-backend-resume/HANDOFF.md
- id: OBS-20261020-004
  observed_at: 2026-10-20
  source: worker handoff
  context: DOCS-01
  fact: README now documents explicit local SQLite demo seeding, sample-host URL and unavailable-versus-clarification behavior without an API key; no live service call is evidenced.
  evidence: README.md; .scratch/ai-chat/issues/04-run-docs.md
- id: OBS-20261020-005
  observed_at: 2026-10-20
  source: validator handoff
  context: VERIFY-01
  fact: Independent Chromium acceptance of the sandbox-send fix passed 22/0 (durable regression) and 42/0 (full fixture) using browser-trusted input, exactly one POST per send, sandbox/CSP unchanged and byte-identical served assets; no live OpenAI/Turso evidence.
  evidence: /tmp/opencode/ai-chat-verification-resume/independent-acceptance.md
- id: OBS-20261020-006
  observed_at: 2026-10-20
  source: user interaction
  context: first AI-chat slice
  fact: User authorized commit and push; commit 69ca0f0 (54 files, 5872 insertions) pushed to private origin/main and repo privacy confirmed after push. Academic documents remain outside the repository.
  evidence: .harness/plan.md; git log adf10eb..69ca0f0
- id: OBS-20261020-007
  observed_at: 2026-10-20
  source: worker handoff
  context: ESC-01
  fact: Email escalation backend completed additively: contract section, idempotent migration 002, capability-authorized tickets with delivery_outcome pinned to delivery_not_wired by schema CHECK, atomic 5-ticket cap and byte-for-byte duplicate replay; all gates pass including a new escalation fuzz target; primary independently reviewed diffs and reran gates.
  evidence: .scratch/support-escalation/issues/01-backend.md; /tmp/opencode/ai-escalation-backend/HANDOFF.md; docs/adr/0001-testing-approach.md
- id: OBS-20261020-008
  observed_at: 2026-10-20
  source: worker handoff
  context: ESC-02
  fact: Widget escalation UI completed additively: click-gated affordance, standalone never-prechecked consent checkbox failing closed, server-truthful outcome rendered verbatim as inert text with validEscalationTicket validation, duplicate replay surfaced as stored-ticket feature, pagehide aborts all in-flight requests; sandbox/CSP byte-for-byte unchanged; Node tests 40 to 61 all green; primary reviewed diffs and reran all gates.
  evidence: .scratch/support-escalation/issues/02-widget.md; web/tests/escalation.test.mjs; web/tests/embedding.contract.test.mjs
- id: OBS-20261020-009
  observed_at: 2026-10-20
  source: worker handoff
  context: ESC-03
  fact: Independent escalation verification passed 15/15 scenario records and 190/190 machine-checked assertions on the real backend with real headless Chromium (996 trusted-input events, zero synthetic submits): consent blocks send zero requests, valid escalations persist truthful delivery_not_wired tickets, duplicate replay is byte-for-byte identical, the 5-ticket cap yields 409 with replay still working, wrong capabilities get three byte-identical 401s without existence oracle, three SIGTERM restarts preserve tickets byte-for-byte, and 429 cooldown disables only the escalation send while chat continues; no blockers.
  evidence: .scratch/support-escalation/evidence/independent-acceptance.md; .scratch/support-escalation/evidence/results.json
- id: OBS-20261020-010
  observed_at: 2026-10-20
  source: worker handoff
  context: LLM-01
  fact: Provider endpoint configuration seam completed additively: OPENAI_BASE_URL names the complete chat-completions URL, unset behavior byte-for-byte unchanged (default branch keeps the literal NewOpenAI call), validator rejects control characters, userinfo, fragments and queries, https required except literal loopback hosts; five fail-fast CLI guard cases; all gates pass; academic Milestone 4 report built in parallel from the pinned pushed HEAD and does not depend on this task.
  evidence: .scratch/provider-endpoint-seam/issues/01-backend.md; /tmp/opencode/ai-endpoint-seam/HANDOFF.md; README.md endpoint bullet
