# Context Map

Derive contexts from actual ownership and communication paths. A directory split alone does not establish a context boundary.

## Ownership map

| Context | Owner | Owned paths | Published contract | Collaborators / communication path |
|---|---|---|---|---|
| Replace with context ID | Replace with person or agent role | `path/to/context/**` | `path/to/contract` | Name who coordinates with whom |

## Relationships

For each dependency, record the upstream context, downstream context, contract exchanged, and coordination owner. Keep this list consistent with the ownership table.

| Upstream | Downstream | Contract/data exchanged | Coordination owner |
|---|---|---|---|

## Boundary review

- Every context has one accountable owner and a non-overlapping set of owned paths.
- Every dependency names a contract and a communication path.
- No context depends on another context's private data or implementation.
- Proposed boundaries match the real team's or agents' collaboration topology.
