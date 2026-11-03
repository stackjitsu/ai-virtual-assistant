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
- id: OBS-20261020-011
  observed_at: 2026-10-20
  source: tool result
  context: live provider evaluation
  fact: Live composition was verified once at revision 190ffe4 against the Gemini OpenAI-compatible endpoint with an operator-held credential: browser-trusted input produced an answered turn grounded in the seeded onboarding entry with its matching approved source; the credential stayed outside the repository and was never printed; no deployment, Turso storage or email transport was involved.
  evidence: .scratch/ai-chat/evidence/live-evaluation.md; /tmp/opencode/live3/evidence/chat-live-answer.png
- id: OBS-20261020-012
  observed_at: 2026-10-20
  source: Turso CLI read-only verification after user-authorized creation
  context: staging database setup
  fact: Created assistant-staging in the stackjitsu organization's default group at aws-ap-northeast-1 on the starter plan with overages disabled; sqlite_master reported no tables. No migration, knowledge seeding, database token creation or app configuration was performed.
  evidence: turso db list; turso db show assistant-staging; turso db shell assistant-staging "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;"
- id: OBS-20261020-013
  observed_at: 2026-10-20
  source: backend and foundation worker handoffs plus primary gate review
  context: JITI-01 and JITI-02
  fact: Explicit local-only seed now contains ten fictional Jitihada University FAQ records; provider grounding and clarification identify the fictional institution, and repeat seeding preserves withdrawn approvals. README marks reserved example.org citation links unresolved. No Jitihada material was imported into assistant-staging, which still has no tables. Primary independently ran format, lint, static, unit, contract and fuzz-smoke checks successfully.
  evidence: .scratch/jitihada-faq/issues/01-backend-content.md; .scratch/jitihada-faq/issues/02-run-docs.md; internal/chat/store.go; README.md
- id: OBS-20261021-014
  observed_at: 2026-10-21
  source: worker handoff plus primary independent CLI verification
  context: JITI-03 controlled FAQ import
  fact: A preview-first import-faq command now applies a reviewed JSONL corpus through one parameterized multi-row upsert, so a batch commits completely or not at all; a lost acknowledgement is resolved by reading stored rows back rather than guessing. Approval requires an explicit operator flag, existing withdrawal is never overwritten, and the summary prints counts and IDs only. Primary confirmed preview writes nothing, apply without --approve is refused, apply with --approve stores approved rows, and turso mode without --allow-remote fails before network I/O. assistant-staging still has no tables.
  evidence: .scratch/jitihada-faq/issues/03-staging-import.md; internal/chat/import.go; cmd/assistant/main_test.go
- id: OBS-20261021-015
  observed_at: 2026-10-21
  source: local preview and one authorized live provider request
  context: manual Jitihada review
  fact: Two local previews ran for manual review: a no-key instance that correctly reported provider unavailable for matching FAQ questions, and a Gemini-backed instance using the operator-held key file, which produced one grounded answered turn citing demo-programmes. The cited example.org URLs do not resolve, and both preview servers were stopped when the session restarted.
  evidence: .scratch/ai-chat/evidence/live-evaluation.md; internal/chat/provider.go
- id: OBS-20261103-001
  observed_at: 2026-11-03
  source: coordinator resume verification
  context: INTEG-01
  fact: Widget integration owner delivered merge commit db38c323f2d0913f541d551707f2b8e70df35f41 on integration/ui-rating, combining approved close-UI checkpoint 6507570 with accepted rating source eface12; owner-run gates and trusted-input browser suites (rating 61/0, sandbox-send 48/0, visual 208/0) passed. The previously dispatched independent validator's claim recorded 2026-11-02 14:35 is stale: the tmux server died leaving no evidence beyond the claim line and the copied owner handoff, so a fresh independent validator was re-dispatched against the exact commit.
  evidence: /home/me/Documents/agent-worktrees/assistant-ui-rating-integration/.scratch/automation/ui-rating-integration/handoff.md; /home/me/Documents/agent-worktrees/assistant-ui-rating-validation/.scratch/automation/combined-validation-brief.md
- id: OBS-20261103-002
  observed_at: 2026-11-03
  source: coordinator resume verification
  context: CI-01
  fact: Bounded CI browser-containment repair was delivered in assistant-ci after two implementation rounds and a handoff recording bounded process-group cleanup verified against a signal-crashed Chromium fixture with expected-fail behavior; gates pass except the static blocker flagging four historical empty-evidence ledger entries at base 1253287, which stays a coordination-owner item. An independent re-review brief was written and its validator dispatched. Prior dispatch model openai/gpt-6.1-sol auth check returned invalid on 2026-11-03; dispatches use verified-ready openai/gpt-6-astra.
  evidence: /home/me/Documents/agent-worktrees/assistant-ci/.scratch/automation/ci-repair/handoff.md; /home/me/Documents/agent-worktrees/assistant-ci/.scratch/automation/ci-re-review-brief.md
- id: OBS-20261103-003
  observed_at: 2026-11-03
  source: coordinator design review (user-delegated decision)
  context: ADMIN-01
  fact: Admin design review accepted the draft staff contract with one modification — password KDF upgraded to Argon2id (OWASP minimum m=19456 KiB, t=2, p=1) via the approved golang.org/x/crypto dependency, keeping stdlib PBKDF2-600k as the documented fallback; session/CSRF/throttling/CAS/audit/privacy design retained unchanged, all-staff visibility accepted for the invite-only v1. Benchmarks consulted: OWASP Password Storage Cheat Sheet and Zammad's groups/conflict-historization/audit model. A bounded contract-revision round was dispatched with required reruns of the earlier quota-blocked gates; no admin implementation is authorized yet.
  evidence: /home/me/Documents/agent-worktrees/assistant-admin/.scratch/admin-dashboard/issues/05-kdf-revision.md; /home/me/Documents/agent-worktrees/assistant-admin/docs/contracts/admin.md; .harness/plan.md
- id: OBS-20261103-004
  observed_at: 2026-11-03
  source: coordinator acceptance plus user decisions
  context: CI-01 and CI-02
  fact: User approved commit, standing push-after-acceptance for the feature branch, and closing CI-01 on local evidence with hosted execution deferred. The coordinator checkpointed the independently accepted CI implementation as 84bfd72 on tooling/ci-autonomy (55 tracked source/evidence files; binaries, profiles, screenshots and sqlite deliberately untracked), marked CI-01 completed, and added CI-02 (hosted deterministic PR validation follow-up, pending user scheduling). Not claimed: hosted run, independent happy-path browser rerun, or a green static on that worktree's historical ledger. AUTO-01's dependency on CI-01 is now satisfied.
  evidence: .harness/plan.md; .harness/tasks.json; /home/me/Documents/agent-worktrees/assistant-ci/.scratch/automation/ci-re-review/handoff.md
