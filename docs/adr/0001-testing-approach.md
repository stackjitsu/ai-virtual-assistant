# ADR 0001: Testing approach is tests-alongside, not strict TDD

Date: 2026-10-20
Status: accepted
Deciders: product owner (user) via orchestrator; backend/widget/validation owners affected

## Context

At project start, test-driven development was recorded as undecided while tests remained required (task briefs and goal state). The first slice was implemented with tests written alongside the code, gate-enforced by `make format lint static unit contract fuzz-smoke`, and independently verified. The second slice (email escalation) required this choice to be resolved before its tasks are shaped.

## Decision

Continue with tests-alongside development. Strict red-to-green TDD is not adopted for the escalation slice. Tests remain mandatory, and the existing validation gates are the accepted bar:

- Every change passes all required checks before acceptance
- Durable browser regressions are re-run for user-visible changes
- Fuzz campaigns cover boundary changes
- Race detection runs over server code

## Consequences

- Task briefs do not require failing-test-first artifacts or red/green evidence for ordinary changes
- The coverage bar is unchanged; passing the same gates remains the definition of complete
- A feature-specific brief may still demand a reproduction test for a diagnosed defect, as happened for the sandbox-send fix
- This decision can be revisited per slice; a future slice may adopt strict TDD by a new ADR rather than silent practice change
