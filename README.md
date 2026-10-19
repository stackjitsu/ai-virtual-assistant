# AI Virtual Assistant

Initial development scaffold for an AI-powered web support assistant. No application features, runtime dependencies, provider configuration, or application tests exist yet.

## Product direction

The [pinned goal](.harness/goal.md) defines the first AI-chat slice, its acceptance criteria, and access model.

Planned stack:

- Framework-neutral JavaScript widget
- Go backend with Chi
- Turso/libSQL for persistence
- OpenAI for AI responses

## Before implementation

1. Define accountable owners, communication paths, and contracts in the [context map](docs/architecture/context-map.md), using `templates/context/`.
2. Configure real commands for every `validation.requiredChecks` entry in `harness.json`.
3. Mark the manifest `active` only after ownership, contracts, and validation adapters are ready.

The manifest remains `template`, with no application contexts and an `unconfigured` adapter. Bootstrap file checks are not application test results. TDD is undecided; see the [active plan](.harness/plan.md).

## Development workflow

The copied her-ness starter supplies development coordination, not an autonomous product runtime. Start with [AGENTS.md](AGENTS.md); `.harness/` holds the goal, plan, tasks, blockers, and observations. Validation expectations live in the [testing contract](docs/testing/testing-contract.md).
