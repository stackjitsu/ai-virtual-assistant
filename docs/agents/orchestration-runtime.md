# Persistent Orchestration Runtime

This contract guides the later runtime layer. The runtime is an adapter over an agent platform; it must preserve the project workflow and state contract without assuming a particular model provider.

## Session lifecycle

- Keep one persistent orchestrator session as the source of current goal and plan state.
- Continue the autonomy loop through the platform's session/turn mechanism; do not use cron or an external heartbeat process to simulate persistence.
- Keep the active goal pinned at the top of the orchestrator's working context and synchronized with `.harness/goal.md`.
- Continue until the goal is complete, the user pauses or changes it, or a blocker requires a user decision. Report the next decision and current state at each handoff.

## State the orchestrator maintains

At all times, keep these fields discoverable in `.harness/`:

- current goal and success criteria;
- active plan and completed steps;
- delegated task IDs, owners, and statuses;
- outstanding blockers and next actions;
- next decision point.

Treat `.harness/tasks.json` as the structured task ledger and validate it against `.harness/tasks.schema.json` when the runtime supports JSON Schema validation.

## Interactive workers

- Spawn workers through long-lived interactive sessions or the platform's equivalent asynchronous agent mechanism.
- Give each worker a bounded task contract, owning context, allowed paths, dependencies, and required evidence.
- Track claims, progress, completion, and blockers centrally. Workers report results; the orchestrator reviews and assigns follow-up work.
- The orchestrator coordinates and reviews; workers implement within owned contexts.

## Observation and consolidation

Use observer agents to process interaction history in approximately 10,000-token chunks and emit atomic observations. When the observation pool approaches 15,000 tokens, consolidate older observations by topic into human-readable Markdown while retaining evidence pointers. The memory rules live in `observational-memory.md`.
