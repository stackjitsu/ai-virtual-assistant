# Pinned Goal

Status: first-slice implementation authorized; contract and validation setup in progress

## Goal

Build an AI-powered web support assistant whose first slice answers free-text questions from approved knowledge, persists isolated conversations, and fails safely.

The canonical support actor is **User**. Guest and signed-in access are separate access states, not separate actors. The planned stack is recorded in `README.md`.

## Success criteria

- [ ] User can ask a free-text support question through a framework-neutral web widget; AI chat is the primary interaction, with optional minimal context.
- [ ] Answers are grounded in approved support knowledge; uncertainty and unavailable knowledge produce a safe response rather than invented guidance.
- [ ] Conversations persist, and access boundaries prevent one User or session from reading another's conversation.
- [ ] Provider failures fail safely without exposing secrets or private conversation data.
- [ ] Owned contracts and real validation adapters are configured before implementation; applicable application checks then provide evidence for this slice.

## Current boundary

The current task is the first AI-first chat slice: a framework-neutral widget, Go/Chi backend, approved-knowledge retrieval, conversation persistence/isolation and safe failure handling. OpenAI/Turso adapters may be implemented and tested with local fixtures; live credentialed requests, infrastructure provisioning and deployment require separate setup/consent. Academic report files remain untouched. Staff administration, live escalation and WhatsApp are later slices.

TDD remains an open workflow choice. Tests and reproducible validation are required now; this is not a claim that test-first development was selected.

## Next decision

Review the first-slice contract/validation baseline, implement within assigned ownership, then review evidence and outstanding live-service blockers. No commit, push, provider charge or production deployment is authorized by this development task.
