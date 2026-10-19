# Domain Docs

How engineering skills consume this project's domain documentation.

## Before exploring

- Read `GLOSSARY.md` at the project root. If `GLOSSARY-MAP.md` exists, read the glossary for each context relevant to the task.
- Read ADRs in `docs/adr/` that touch the area. In multi-context projects, also read context-scoped ADRs under the relevant source context.
- If a glossary or ADR directory is absent, proceed silently. Create or update domain docs when terms or decisions are actually resolved.

## Layout

This starter uses a single-context documentation layout:

```text
/
├── GLOSSARY.md
└── docs/
    └── adr/
```

If the adopted project has multiple bounded contexts, add a root `GLOSSARY-MAP.md` linking each context's glossary. Keep system-wide decisions in `docs/adr/` and context-specific decisions beside their owning context.

## Vocabulary and decisions

- Use the glossary's terms in issue titles, hypotheses, refactor proposals, and test names. Avoid synonyms the glossary explicitly rejects.
- If a needed term is missing, note the vocabulary gap for domain modeling rather than silently inventing a competing term.
- If proposed work conflicts with an ADR, identify the ADR and explain why it may need to be reopened. Do not silently override it.
