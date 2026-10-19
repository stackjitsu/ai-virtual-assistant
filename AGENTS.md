# Project Agent Rules

This repository is AI Virtual Assistant, an AI-powered web support assistant. The adopted her-ness rules govern development, not an autonomous product runtime. Keep application-specific decisions in the context map, manifest, and ADRs.

## Work loop

1. Read this file, the pinned goal in `.harness/goal.md`, the relevant glossary terms, and ADRs that touch the work.
2. Identify the affected context, its owner, its contract, and its permitted paths in `docs/architecture/context-map.md`.
3. Write or update the task brief before implementation. Include the goal, owning context, allowed paths, data invariants, acceptance criteria, and required checks.
4. Delegate implementation and validation to the context owner. The orchestrator owns planning, coordination, review, and follow-up; it does not implement application changes directly.
5. Make data shapes and invariants explicit before changing behavior. Keep transformations pure where practical and put I/O at adapters.
6. Run `format`, `lint`, `static`, and `unit` on each change; run `contract` for published-contract changes and affected consumers; run `fuzz-smoke` for changes affecting fuzzable boundaries or state transitions; run `fuzz-full` before release or after high-risk boundary changes. An active manifest configures a command for every `requiredChecks` entry; this makes each command available, not required for every task.
7. Review the result against the task, contracts, data invariants, and test evidence. Update task state, blockers, and observations before closing the work.

## Hard boundaries

- A context has one accountable owner and an explicit communication path to its collaborators. Architecture follows the real ownership and communication topology; do not create a boundary without naming who coordinates across it.
- Cross-context communication uses a declared contract. A context does not reach into another context's private data or implementation. Coordinate contract changes with every affected owner.
- The primary agent keeps the goal, plan, delegated tasks, blockers, and next decision visible in `.harness/`. Workers change only their assigned context and allowed paths.
- Application code is not complete until required checks pass or a blocker is recorded with evidence and an owner. Missing adapters or missing fuzz coverage are blockers, not implicit waivers.
- Record durable decisions in ADRs and compact factual observations in `.harness/observations/`; do not use recursive summaries as the only long-term memory.

## Reach these references when relevant

- Read `docs/architecture/rules.md` when setting boundaries, ownership, data shapes, or state transitions.
- Read `docs/testing/testing-contract.md` and `docs/testing/fuzzing.md` when adding or changing behavior, test coverage, or validation adapters.
- Read `docs/agents/delegation-protocol.md` when creating or updating delegated tasks.
- Read `docs/agents/observational-memory.md` when recording or consolidating observations.
- Read `docs/agents/orchestration-runtime.md` when designing or changing the persistent orchestrator or subagent runtime.
- Read `docs/agents/issue-tracker.md`, `docs/agents/triage-labels.md`, and `docs/agents/domain.md` when using the engineering skills.

## Agent skills

### Issue tracker

Issues and specs live as local Markdown under `.scratch/<feature>/`. See `docs/agents/issue-tracker.md`.

### Triage labels

Uses the default labels: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, and `wontfix`. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context docs layout: root `GLOSSARY.md` and decisions in `docs/adr/`. See `docs/agents/domain.md`.
