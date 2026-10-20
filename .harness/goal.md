# Pinned Goal

Status: slice 2 (visitor email escalation) implemented and independently verified (15/15); awaiting explicit user authorization for commit/push; live OpenAI/Turso report figure still pending user credential

## Goal

Build an AI-powered web support assistant. Slice 1 (complete, pushed 69ca0f0/ac7bdb6): AI-first approved-knowledge chat for guests with isolated persistence and safe failure handling. Slice 2 (complete, uncommitted): the guaranteed visitor email-escalation path, additively, per `.scratch/support-escalation/spec.md` — implemented (ESC-01), exposed in the widget (ESC-02), and independently verified end to end (ESC-03).

The canonical support actor is **User**. Guest and signed-in access are separate access states, not separate actors. The planned stack is recorded in `README.md`.

## Success criteria

- [x] User can ask a free-text support question through a framework-neutral web widget; AI chat is the primary interaction, with optional minimal context.
- [x] Answers are grounded in approved support knowledge; uncertainty and unavailable knowledge produce a safe response rather than invented guidance.
- [x] Conversations persist, and access boundaries prevent one User or session from reading another's conversation.
- [x] Provider failures fail safely without exposing secrets or private conversation data.
- [x] Owned contracts and real validation adapters are configured before implementation; applicable application checks then provide evidence for this slice.
- [ ] User can request human support with an explicit consent and validated email; the request is stored as a ticket record with a truthful delivery outcome (slice 2).

## Current boundary

The current task is the visitor email-escalation slice: additive contract, storage migration, capability-authorized ticket records with a local email-outcome adapter. No staff interface, WhatsApp, or auth changes in this slice. Live OpenAI/Turso, real SMTP transport, infrastructure provisioning and deployment require separate setup/consent. Academic report files remain untouched.

Testing approach is decided: tests-alongside development, not strict TDD (ADR 0001). Tests and reproducible validation remain required for every change.

## Next decision

Dispatch ESC-01 (backend + additive contract), then ESC-02 (widget affordance), then independent validation (ESC-03). No new commit or push without explicit user authorization. The live AI-answer screenshot and live smoke checks for the academic report remain pending the user's separately held provider credential.
