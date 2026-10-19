# Architecture Rules

These rules define the architecture gate for an adopting project. Record project-specific decisions and exceptions in its context map and ADRs.

## Conway-aligned boundaries

1. Map the people or agents that own work and the communication paths they use before defining contexts.
2. Give each context one accountable owner, owned paths, a contract, and named collaborators.
3. Make every cross-context dependency visible in the context map. Cross-context collaboration happens through contracts and their owners, not private implementation access.
4. Keep boundaries aligned with real coordination: when ownership or communication topology changes, review the context map and affected contracts.
5. Resolve shared changes by naming a coordinating owner and the affected context owners before implementation.

## Data-oriented design

1. Define canonical data shapes, units, identity, validity rules, and compatibility expectations before behavior that consumes them.
2. Keep domain state explicit and inspectable. Prefer data records and transformations over hidden mutable object state.
3. Separate validation at boundaries, pure transformations, state transitions, and side effects. Put external I/O behind adapters.
4. State invariants next to the owning data contract and test them at the boundary where invalid data could enter.
5. Version or migrate externally consumed data deliberately; update contract tests and affected owners together.

## Change gate

A change is ready for implementation only when its owning context, allowed paths, data invariants, contract impact, and validation plan are known. A cross-context change additionally records affected owners and the coordination path.
