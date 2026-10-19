# Pinned Goal

Status: pinned; application implementation not started

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

This bootstrap contains application identity and development workflow metadata only. Application code, dependencies, provider setup, and an autonomous product runtime are outside this task.

## Next decision

Assign the first slice's owners and contracts, configure the validation adapters, and decide whether to use TDD. No test-first commitment has been made.
