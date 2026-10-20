# Active Plan

Status: slices 1–2 and the provider endpoint seam complete and pushed; live composition verified locally; no deployment

| Step | Outcome | Owner | Status | Depends on |
|---|---|---|---|---|
| 01 | Adopt reviewed starter and pin application identity and scope | repository-bootstrap worker | completed | |
| 02 | Define first-slice ownership, contracts, and data invariants | contract/tooling worker; orchestrator review | completed | 01 |
| 03 | Configure real required validation adapters | contract/tooling worker | completed | 02 |
| 04 | Implement backend persistence, grounded AI and HTTP behavior | backend owner | completed; local fixture/race/full-fuzz evidence reviewed | 03 |
| 05 | Implement and test the embeddable widget/sample host | widget owner | completed; sandbox-send fix independently validated | 03 |
| 06 | Independently validate contracts, isolation, failure paths and browser behavior | validation owner; orchestrator review | completed; independent actual-input acceptance passed; primary gates rerun | 04, 05 |
| 07 | Document exact local startup and honest service limitations | foundation documentation owner | completed | 04, 05 |
| 08 | Implement visitor email escalation backend and additive contract | backend owner | completed; ESC-01 accepted after primary diff + gate review | 07 |
| 09 | Expose escalation affordance, consent and email capture in widget | widget owner | completed; ESC-02 accepted after primary diff + gate review | 08 |
| 10 | Independently validate escalation flow end to end | validation owner | completed; ESC-03 15/15 scenarios PASS accepted after primary evidence review | 08, 09 |
| 11 | Add provider endpoint configuration seam for zero-cost live evaluation | backend owner | completed; LLM-01 accepted after primary diff + gate review | 08 |
| 12 | Verify live composition against a compatible hosted endpoint at zero cost | orchestrator / validation evidence | completed; one grounded answered turn with matching approved source recorded in `.scratch/ai-chat/evidence/live-evaluation.md` | 11 |

## Decision log

- The first-slice ownership, exact data/HTTP/provider/store/embedding contracts and real validation commands have been reviewed; manifest is active. This gate is not evidence that app tests pass before implementations exist.
- Bootstrap file checks do not satisfy the future application's validation gates.
- TDD remains undecided. Durable architectural decisions belong in `docs/adr/` once resolved with the affected owners.
- Canonical actor: User. Access may be guest or signed-in; this slice uses guest conversation-scoped access.
- AI-first answers use approved knowledge. Direct FAQ strings must not masquerade as generated AI answers. Missing knowledge/provider failure is a clearly labelled limitation, not success.
- Local SQLite may support offline development of the Turso/libSQL-compatible schema; live Turso and OpenAI tests require explicit credentials and are not implied by local test passes.
- Resume check: independently ran make format, lint, static, unit, contract and fuzz-smoke against persisted implementation; all passed. This does not close pending backend review/browser checks or establish live-service integration.
- Independent acceptance: actual Chromium mouse/keyboard input submitted exactly one POST per send against the actual Go/SQLite backend with sandbox/CSP unchanged and byte-identical served assets; primary reran all integrated gates on the post-fix tree.
- First-slice commit 69ca0f0 (54 files, 5872 insertions) pushed to private origin/main after explicit user authorization; remote ref and privacy verified after push.
- Testing approach resolved as tests-alongside, not strict TDD, for current and near-term slices; recorded in `docs/adr/0001-testing-approach.md`.
- Slice-2 commits 896ed7a and closure 8915abc pushed to private origin/main after explicit user authorization; escalation records, evidence and harness state included.
- Provider seam commits e4bdf39 and closure 190ffe4 pushed after explicit user authorization; the default provider endpoint stays byte-for-byte unchanged.
- Live composition evaluated once on 2026-10-20 against the Gemini OpenAI-compatible endpoint with an operator-held credential: browser-trusted input produced an answered turn grounded in the seed entry with its approved source. This is a single-question evaluation, not deployment readiness (see `.scratch/ai-chat/evidence/live-evaluation.md`).

## Next decision

Both slices, the endpoint seam and the live evaluation are closed; no task is dispatched. The user's next decisions are deployment (hosted Turso/libSQL plus production provider configuration), real email delivery, and the conditional WhatsApp channel, each requiring separate setup, credentials and authorization.
