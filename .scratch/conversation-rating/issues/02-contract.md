# RATE-01: additive rating contract

Status: completed
Owner: project-foundation contract worker. Allowed paths: `docs/contracts/**`, `docs/adr/**`, this issue. Depends on: none.

Specify exact `POST /api/v1/conversations/{id}/rating` request/response, status/errors, auth/origin/rate bounds, at-least-one-turn rule, replay vs replace semantics, canonical validation and version-3 migration invariants from spec. Coordinate backend and widget consumption through the orchestrator. No application changes or credentials. Checks: `make format lint static unit contract`, `git diff --check`. Report contract decisions and paths.

## Handoff

Published additive HTTP/widget behavior in `docs/contracts/chat.md` and storage/migration obligations in `docs/contracts/backend.md`. Backend and widget owners consume these contracts via the orchestrator; no runtime implementation is asserted by this documentation task. Decision points: success 200 for first/replay/revision, zero completed turns 409 `rating_requires_turn`, independent 10-attempt/minute rating window, `comment:""` canonical absence, distinct server timestamp on changed replacement. No ADR needed: these specify the approved slice without changing an existing architecture decision.
