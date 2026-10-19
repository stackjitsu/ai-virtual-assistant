# Observational Memory

Use observations as the durable record of what happened. Keep them factual, atomic, and independently meaningful; distinguish observed fact from inference.

## Observation record

Record each observation with:

- `id`: stable identifier
- `observed_at`: timestamp when useful
- `source`: interaction, tool result, test, or file
- `context`: relevant project context or task
- `fact`: one concise observation
- `evidence`: pointer to the source when available

Append observations to `.harness/observations/INBOX.md`. An observer processes interaction-history chunks of approximately 10,000 tokens and emits atomic observations rather than another narrative summary.

When the observation pool approaches 15,000 tokens, consolidate older observations into topic-specific human-readable Markdown files. Preserve source pointers for claims that could be revisited, then remove only observations whose useful content is represented in the consolidated file.

Long-term memory lives in editable Markdown. Recursive LLM summaries alone are not durable memory.
