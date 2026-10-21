# RATE-01: additive rating contract

Status: ready-for-agent
Owner: project-foundation contract worker. Allowed paths: `docs/contracts/**`, `docs/adr/**`, this issue. Depends on: none.

Specify exact `POST /api/v1/conversations/{id}/rating` request/response, status/errors, auth/origin/rate bounds, at-least-one-turn rule, replay vs replace semantics, canonical validation and version-3 migration invariants from spec. Coordinate backend and widget consumption through the orchestrator. No application changes or credentials. Checks: `make format lint static unit contract`, `git diff --check`. Report contract decisions and paths.
