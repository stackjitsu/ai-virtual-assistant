# Pinned Goal

Status: slices 1–2 and the provider endpoint seam are committed and pushed (slice-2 896ed7a/8915abc, seam e4bdf39/190ffe4); live composition was verified once locally on 2026-10-20; Milestone 3 and 4 report figures are complete; nothing is deployed

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

No implementation is dispatched. The next decisions are the user's: whether to prepare deployment (hosted database and production provider configuration under separately authorised accounts), whether to wire real email delivery, and whether the conditional WhatsApp channel proceeds. No new commit or push without explicit user authorization.
