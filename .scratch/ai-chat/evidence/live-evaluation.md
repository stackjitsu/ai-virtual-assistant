# Live provider evaluation record

Date: 2026-10-20

Mode: live hosted provider with a local loopback server and local SQLite store; no remote storage, email transport, or deployment.

## Setup

- Revision: 190ffe4 — `go build -o /tmp/opencode/live3/assistant ./cmd/assistant`
- Server: `LISTEN_ADDR=127.0.0.1:18093`, fresh SQLite database, `serve --seed-demo`
- Provider: hosted OpenAI-compatible chat-completions endpoint (`generativelanguage.googleapis.com/v1beta/openai/chat/completions`), model `gemini-2.5-flash`
- Credential: operator-held file outside the repository (`~/.config/assistant/openai_key`, mode 600); never printed, never placed in browser assets or commands, and not recorded here
- Input: "How do I start the demo?" typed with browser-trusted input (CDP mouse click plus `Input.insertText`) into the real widget

## Result

| Field | Value |
|---|---|
| Expected | Concise AI-composed guidance grounded in the approved onboarding entry, with a matching approved source |
| Actual | Turn stored with `status=answered`; widget showed the "AI answer" label and the text "To start the demo, you need to open the sample support widget and ask an onboarding question."; approved source "Sample organization: getting started" was listed |
| Stored turn | One conversation, one turn, 93-character assistant text |
| Result | PASS for this question |

## Evidence (outside the repository, transient)

- `/tmp/opencode/live3/evidence/chat-live-answer.png`, sha256 `34696f58001c46a675830948a6f5c08d572b9ac488d7bea1a2127b4d7258e420`
- `/tmp/opencode/live3/db/assistant.db`, `/tmp/opencode/live3/answer.txt`, `/tmp/opencode/live3/server.log`
- The same PNG is embedded byte-identically in the Chapter 3 report deliverable, which remains outside this repository.

## Limits and honest caveats

- One question, one provider, one run. A grounding/citation check does not establish factual correctness of the fictional seed material.
- The widget's pre-use disclosure keeps the product's default "OpenAI" wording; this evaluation exercised the unchanged composition protocol against an OpenAI-compatible endpoint.
- No Turso/libSQL service, real email delivery, deployment, or production configuration was involved. No production security or readiness claim follows from this record.
