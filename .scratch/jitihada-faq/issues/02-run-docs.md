# JITI-02 — Accurate local Jitihada FAQ documentation

Status: completed; project-foundation documentation worker updated README local demo guidance and validated it.

Owner: project-foundation worker. Allowed paths: `README.md` and this issue only. No widget, backend, shared contract, harness, academic, or unrelated file edits. No live credentials/calls, database migrations/imports, commits or pushes.

Goal: update existing local-run README wording for the reviewed JITI-01 backend behavior without implying staging or a finished mock university website.

Refs: `.scratch/jitihada-faq/issues/01-backend-content.md`, `internal/chat/store.go` (`SeedDemo`), `internal/chat/http.go`, `internal/chat/provider.go`, `docs/contracts/backend.md`, `docs/architecture/context-map.md`.

Invariants: current published HTTP shapes unchanged; explicit local-only seed with 10 fictional Jitihada FAQs; repeated seeding preserves withdrawn approval flags and does not purge unrelated records; no live Turso schema or knowledge; source URLs are reserved example.org placeholders until real mock FAQ pages exist, never imply citations currently resolve. Missing API key still 503 for relevant questions, unrelated questions still honest clarification. Do not claim AI output verified from live corpus. No changes to current operator configuration or service startup instructions beyond correcting demo copy/sample questions.

Acceptance: README seed table and walkthrough match exact currently seeded content (no real support hours claim). Clearly distinguish local demo content vs unseeded assistant-staging, provisional links, missing mock site, and separate future import. Run `make format`, `make lint`, `make static`, `make unit`, `make contract` (read-only; avoid concurrent source edits) and report evidence; record blockers when checks are affected by other contexts. Dependency: JITI-01 review.

## Worker handoff (2026-10-20)

Changed paths: `README.md` and this issue only. The existing local-run command and unrelated README sections remain unchanged. The table now names all ten current `SeedDemo` IDs/titles with grounded example details and question prompts, including that the IT-support record offers no support hours. Local SQLite seeds on explicit invocation only: fresh databases receive ten approved records, while reseeding refreshes matching content without changing existing approval flags or deleting unrelated records. The README distinguishes this from unseeded assistant-staging/Turso, and says `https://example.org/jitihada/help/` URLs are reserved placeholders, not usable FAQ citations; the mock site and pages are absent. Remote import with real, resolving URLs remains a separately authorized future operation. No live AI answer, deployment or staging import was claimed.

Evidence: `make format` PASS; `make lint` PASS; `make static` PASS; `make unit` PASS (Go tests and 48 Node unit tests); `make contract` PASS (Go and 13 Node contract tests); `git diff --check` clean. Checks ran sequentially without source writes. Blockers for this documentation task: none. Future staging import still blocked on reviewed, resolving HTTPS FAQ pages and separate authorization (orchestrator/widget owner to coordinate). No credentials, live service calls, remote migration/import, commits or pushes.
