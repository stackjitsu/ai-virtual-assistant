# DOCS-01: local run documentation

Status: completed

Owner: foundation documentation worker. Context: project-foundation.

Update README.md and stale status wording in docs/testing/project-checks.md/harness.json only to describe the actual implemented local slice accurately. Existing CLI is cmd/assistant/main.go: serve --seed-demo, default SQLite ./data/assistant.db, sample host /widget/sample-host.html, no automatic .env loading. Grounded inference requires an OpenAI key; no live service tests authorized or completed. Actual demo facts/questions must match database seed contents. Keep exact configuration and clear safety/limitations, no feature expansion. Runtime/widget review proceeds in parallel; do not edit those paths or run overlapping full checks. Report read-only format/JSON/link checks and known remaining backend/browser acceptance gate. See .harness/tasks.json DOCS-01.

## Work brief / claim

- Claimed by DOCS-01 foundation documentation worker on 2026-10-20.
- Allowed writes: `README.md`, `docs/testing/project-checks.md`, adapter name only in `harness.json`, and this issue. Runtime, widget, contracts, dependencies, shared harness state and academic report are read-only.
- Acceptance: concise runnable SQLite quickstart, exact fictional seed facts and example questions, configuration/privacy limits, current routes/check commands, and honest local-fixture versus pending browser/live-service evidence.
- Invariants: no credential reads or live calls, no implicit `.env` loading, no canned AI-success claim, no remote migration execution or production-readiness claim.
- Checks: focused read-only text-format, local Markdown-link and JSON syntax checks only; do not run overlapping full Go/Node gates. Primary reports format/lint/static/unit/contract/fuzz-smoke passed; independent backend/browser review remains pending.

## Handoff / evidence

- Changed only `README.md`, `docs/testing/project-checks.md`, the adapter name in `harness.json`, and this issue. No runtime, widget, contract, dependency, test, shared harness-state or academic-report edits; no commit/push.
- README now gives fresh-clone dependency download, explicit key-free SQLite seeded start, sample-host/widget flow, exact fictional onboarding and Monday–Friday seed facts/questions, unseeded `make run`, opt-in storage/migration configuration, disclosure and current limits/routes.
- Read-only Python check passed for UTF-8/final newline/no CRLF/no trailing whitespace on all four owned files, balanced Markdown code fences, all 10 local Markdown links, and JSON syntax/adapter name. No full Go/Node gates rerun and no dependencies downloaded, app started, credentials read or external calls made.
- Reported automated baseline comes from the primary session, not new worker execution: format/lint/static/unit/contract/fuzz-smoke pass, including 36 Node tests, Go race tests and three fuzz-smoke targets. Backend review/browser acceptance remain pending; full fuzz and live-service success are not claimed.
- Next decision: orchestrator reviews documentation and reconciles later independent backend/browser/full-fuzz evidence separately. Documentation completion does not close those acceptance gates.

## Backend-evidence follow-up / claim

- DOCS-01 reclaims the same documentation-only scope after receiving `/tmp/opencode/ai-chat-backend-resume/HANDOFF.md`; no backend checks will be duplicated.
- Acceptance: incorporate reported full-fuzz and actual CLI subprocess evidence, use the handoff's explicit credential-empty local command, and explain reseeding updates/reapproves both fictional documents rather than being normal startup. Browser acceptance stays pending; no live OpenAI/Turso claim.

### Follow-up handoff

- README now uses the backend's fully specified local command with empty OpenAI/Turso credentials and distinguishes explicit reapproval/update seeding from normal unseeded startup. No credentials were inspected or used.
- Backend evidence supersedes the earlier pending-full-fuzz wording: all seven gates reported passing; actual CLI startup/restart/migration guards tested in subprocesses; full-fuzz execution counts provider 119,499, Turso 23,344, message 4,416 at 30 seconds/target, 2 workers. Project checks preserve the provider-smoke cache-loading caveat and temporary evidence locations.
- Integrated review/browser acceptance remains pending. Fixture tests are not real OpenAI/Turso integration. No backend/Go/Node checks duplicated. Focused read-only checks passed: four owned files' UTF-8/newline/whitespace, balanced Markdown fences, 10 local links, JSON syntax/adapter name, and credential-empty quickstart shell tokenization (not executed).
