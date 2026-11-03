# Pinned Goal

Immediate user-directed step (resumed 2026-11-03): user-approved close UI and accepted rating are integrated by the widget owner as db38c32 on integration/ui-rating (first parent 6507570, second parent eface12) with owner gates and browser suites passing; the prior independent validator died after claiming without evidence, so combined-tree validation is re-dispatched against the exact commit and is running. CI-01 was independently accepted and checkpointed by the coordinator as 84bfd72 on tooling/ci-autonomy; hosted execution deferred as CI-02 by user decision; AUTO-01's dependency is satisfied and dispatches after the integration push. Admin design is reviewed and accepted with one KDF modification (Argon2id); contract-revision round running. After combined independent acceptance standing user approval authorizes feature-branch fast-forward/push only; no main merge, deployment or hosted CI execution is authorized. RATE-02 storage-test and RATE-03 trusted-input/actual-backend browser follow-ups are delivered but not yet accepted.

Current coordination goal (resumed 2026-11-02): finish evidence-backed acceptance of recovered guest conversation rating and independently review CI-01 before AUTO-01. RATE-02 storage-test and RATE-03 trusted-input/actual-backend browser follow-ups are delivered but not yet accepted. RATE-04 independent acceptance passed on recovered rating plus the two test patches, including all gates/full fuzz and real browser regressions. Accepted test/evidence batch is committed locally as eface12 on validation/rating-acceptance in assistant-rating-acceptance. User approved proceeding and validation/rating-acceptance is now pushed; remote head verified at eface12. No merge performed: checked-out feat/conversation-rating has overlapping uncommitted work, including incomplete UX-02; RATE-02/03 feature integration remains separate and must preserve that work. CI-01 independent review found a reproducible browser-adapter hang and ownership-documentation drift; bounded foundation repair is dispatched in assistant-ci. Workers must return handoffs for coordinator review; running sessions are not completion evidence. Preserve main-tree Escape/UI changes. No commits, pushes, merges, deployment or remote writes are authorized.

Staff dashboard contract/design is approved work, not deferred: ADMIN-01 has delivered a draft handoff in `/home/me/Documents/agent-worktrees/assistant-admin/.scratch/admin-dashboard/handoff.md`; review and security/policy decisions remain before any implementation. No staff runtime exists. Live email and WhatsApp remain deferred; public release remains no-go. Next decision: review bounded rating/CI follow-up handoffs, then independently validate an identified integrated revision. AUTO-01 waits for CI acceptance. Historical slice/preview details below are not claims that old processes or URLs are currently running.

Status: slices 1–2 and the provider endpoint seam are committed and pushed (slice-2 896ed7a/8915abc, seam e4bdf39/190ffe4); live composition was verified once locally on 2026-10-20; Milestone 3 and 4 report figures are complete; nothing is deployed. UI-01 local visual pass remains uncommitted. User selected a production-ready visitor-support release; PROD-01 audit completed with a public-release no-go.

## Goal

Build an AI-powered web support assistant. Slice 1 (complete, pushed 69ca0f0/ac7bdb6): AI-first approved-knowledge chat for guests with isolated persistence and safe failure handling. Slice 2 (complete, pushed 896ed7a with closure 8915abc): the guaranteed visitor email-escalation path, additively, per `.scratch/support-escalation/spec.md` — implemented (ESC-01), exposed in the widget (ESC-02), and independently verified end to end (ESC-03).

The canonical support actor is **User**. Guest and signed-in access are separate access states, not separate actors. The planned stack is recorded in `README.md`.

## Success criteria

- [x] User can ask a free-text support question through a framework-neutral web widget; AI chat is the primary interaction, with optional minimal context.
- [x] Answers are grounded in approved support knowledge; uncertainty and unavailable knowledge produce a safe response rather than invented guidance.
- [x] Conversations persist, and access boundaries prevent one User or session from reading another's conversation.
- [x] Provider failures fail safely without exposing secrets or private conversation data.
- [x] Owned contracts and real validation adapters are configured before implementation; applicable application checks then provide evidence for this slice.
- [x] User can request human support with an explicit consent and validated email; the request is stored as a ticket record with a truthful delivery outcome (slice 2).

## Current boundary

Both planned slices and the provider endpoint seam are complete and pushed. Live composition was verified once, locally, against a hosted OpenAI-compatible endpoint with an operator-held credential; that verification is an evaluation result, not a deployment or production claim. Remaining work requires separate user direction and accounts: Turso/libSQL hosting, real SMTP transport, deployment, the conditional WhatsApp channel, and any staff-facing interface. Academic report files remain outside the repository.

Testing approach is decided: tests-alongside development, not strict TDD (ADR 0001). Tests and reproducible validation remain required for every change.

## Next decision

User approved an accessible Request human support menu with By email available, By WhatsApp explicitly unavailable, and a compact More information disclosure. UX-01G is complete and previewed at plain local port 18100; await user feedback before further UI/UX work. PROD-01 no-go blockers remain unresolved; no production claim, commit or push without explicit user authorization.
