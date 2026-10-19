# Fuzzing Rules

Fuzzing is part of the test contract, not an optional post-implementation activity. The project-specific adapter chooses the native fuzzer; the following coverage and evidence rules remain constant across languages.

## Targets

Add targets for externally influenced parsers and decoders, serialization round trips, API or file boundaries, and state-transition logic with meaningful input space. Prioritize code that accepts untrusted, malformed, oversized, or combinatorial input.

## Invariants

Each target names the property it checks. Use the properties that fit the boundary:

- arbitrary input does not crash, hang, violate memory safety, or exceed stated resource bounds;
- accepted data satisfies the owning schema and domain invariants;
- encode/decode round trips preserve the defined semantic value;
- state transitions preserve safety, ordering, and conservation invariants;
- equivalent inputs produce equivalent outputs where determinism is part of the contract.

## Reproducibility and gates

- Keep seed inputs and regression cases in the owning context's test corpus.
- On failure, preserve the minimized reproducer, record the invariant violated, and add a regression test before closing the task.
- Run `fuzz-smoke` with a bounded budget for changes affecting fuzzable boundaries or state transitions.
- Run `fuzz-full` before release or after high-risk boundary changes; record tool version, budget, seed/corpus revision, and result.
- A target without a stated invariant is incomplete. Missing fuzz adapter commands block activation of the harness manifest.
