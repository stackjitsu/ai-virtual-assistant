# CI-01 — deterministic local/PR validation

Status: claimed by foundation worker; orchestrator acceptance pending.

## Goal
Implement deterministic GitHub Actions checks around the existing validation adapters without touching application changes. User approved CI-01 and AUTO-01; no commit, push, merge or deploy is authorized in this dispatch.

## Ownership and paths
Foundation owner. Allowed: .github/workflows/**, scripts/**, Makefile, docs/testing/project-checks.md, harness.json, .scratch/automation/**. Do not edit application code, shared goal/task ledger, or other worktrees. Request coordination for any other path.

## Acceptance
- PR and manual workflow: format, lint, static, unit, contract, fuzz-smoke and actual existing trusted Chromium browser check against local fixtures.
- Explicit manual full-fuzz option; required jobs fail on missing dependencies, tests or browser.
- Full tasks JSON Schema validation (not just targeted checks), including negative regression tests; expose existing invalid ledger entries as blockers, never weaken schema to pass.
- Read-only GitHub token, no secrets or live service calls, no pull_request_target, bounded job timeouts, artifacts with tested SHA and logs even on failure, pinned actions/tool versions where practical.
- Preserve existing read-only format check; inspect actual tool versions and browser script before selecting CI dependencies.
- Document what was verified locally versus not run on GitHub.

## Required reading and checks
Read AGENTS.md, docs/testing/testing-contract.md, docs/testing/fuzzing.md, docs/architecture/rules.md, docs/testing/project-checks.md, harness.json and scripts before changes. Run make format lint static unit contract fuzz-smoke plus added tooling regressions/browser checks. Tests-alongside, not mandatory TDD.

## Worker implementation plan / invariants
- Foundation worker implements within the listed paths; no application or shared ledger edits.
- Validate the entire task ledger against its existing Draft-07 schema with a pinned JSON Schema implementation, including conditional requirements and additional-property rejection. Report all errors; never normalize invalid records.
- Keep formatting read-only; required commands and browser failures propagate nonzero exit status.
- CI uses isolated local fixtures, read-only token, pinned tools/actions, bounded execution and always-uploaded SHA/log evidence. Full fuzz runs only on explicit manual selection.
- Regression tests accompany validator/adapter changes; run all six requested gates and the existing trusted-input browser script locally. GitHub execution remains unverified until an authorized push/run.

## Handoff
Write .scratch/automation/ci-handoff.md with base SHA, diff hash, changed paths, commands/results, blockers and evidence paths. Worker must not mark centrally completed. Stop after two repair rounds or 30 minutes; no nested workers, installs outside local environment, credentials, production calls or git commits/pushes. Public dependency downloads for build/CI are allowed. Never modify the main working tree.
