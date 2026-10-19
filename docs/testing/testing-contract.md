# Testing Contract

Each project maps its native tools to the common checks in `harness.json`. The adapter owns command syntax; the harness owns the meaning of each check. An active manifest configures a command for every `requiredChecks` entry; this makes each command available and does not require running every command for every task.

| Check | Required evidence |
|---|---|
| `format` | Formatting check for changed files |
| `lint` | Static style and misuse checks |
| `static` | Available type, schema, or compile-time checks |
| `unit` | Focused tests for local behavior and invariants |
| `contract` | Compatibility tests for published context boundaries |
| `fuzz-smoke` | Bounded fuzz/property run for affected fuzzable targets |
| `fuzz-full` | Extended run used before release or after high-risk boundary changes |

## Completion gate

1. Run `format`, `lint`, `static`, and `unit` on each change.
2. Run `contract` for published-contract changes and affected consumers.
3. Run `fuzz-smoke` for changes affecting fuzzable boundaries or state transitions.
4. Run `fuzz-full` before release or after high-risk boundary changes.
5. Treat missing commands, missing relevant coverage, and failing applicable checks as blockers. Any exception requires an ADR with an owner, reason, and expiry or review condition.

CI for an adopting project should run the applicable configured checks on each change and enforce these completion gates.
