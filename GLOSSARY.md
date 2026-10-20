# Glossary

## Support domain

**User**: The person seeking support from the assistant. Guest and signed-in are access states of the same actor, not separate actors.
_Avoid_: Visitor, guest actor, registered-user actor.

**Conversation**: An isolated support exchange between a User and the assistant.

**Completed turn**: A User question and the assistant's corresponding answer or explicit clarification, retained together as one exchange.

**Approved knowledge**: Support material explicitly authorized for assistant answers. Withdrawn material is not approved knowledge.

**Clarification**: An honest request for more information or statement that approved knowledge cannot support an answer; it is not a successful grounded answer.

## Development coordination

| Term | Definition |
|---|---|
| Context | A boundary of owned data and behavior with a named owner and explicit contracts to collaborators. |
| Context owner | The person or agent accountable for a context's implementation, invariants, and published contract. |
| Contract | The explicit data shape and behavior agreed across a context boundary. |
| Validation adapter | Project-specific commands that implement the harness's common check interface. |
| Fuzz target | A function or boundary exercised with generated or mutated inputs to test stated invariants. |
| Observation | A compact, independently meaningful fact recorded from interaction or execution history. |
| Orchestrator | The agent responsible for goal, plan, task coordination, blockers, review, and memory maintenance. |
| Worker | A delegated agent responsible for a bounded task within named contexts and paths. |
