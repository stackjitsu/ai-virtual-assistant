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
