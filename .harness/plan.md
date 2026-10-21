# Active Plan

## Rating slice — integration branch feat/conversation-rating

| Step | Owner | State | Depends on |
|---|---|---|---|
| DISC-01 pre-send disclosure verification | independent validator | completed; wording gap assigned RATE-03 | baseline |
| RATE-01 additive rating contract | foundation | completed | baseline |
| RATE-02 SQLite/Turso rating API | backend | completed; merged, concurrent identical/differing tested | RATE-01 |
| RATE-03 accessible rating widget | widget | completed; merged, timer fix tested | RATE-01 |
| RATE-04 integrated browser/gate review | independent validator | claimed | RATE-02, RATE-03 |

Baseline local work was committed separately as `611d3fa`, with baseline gates passing. Next decision is contract review, then parallel context-owner dispatch. No provider, mail, Turso or production deployment is authorized.

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
| 13 | Refine the visitor corner widget UI and verify responsive/browser states | chat-widget owner; orchestrator review | completed visual pass (UI-01); awaiting user visual feedback, mobile composer not sticky | 05, 09 |
| 14 | Audit production release scope and blockers for visitor support | independent validation owner; orchestrator review | completed (PROD-01); public release is no-go | 13 |
| 15 | Redesign corner chat layout with composer in view at mobile size | chat-widget owner; orchestrator review | completed (UX-01); live local preview on 8080 awaiting user feedback | 13 |
| 16 | Refine corner-widget surfaces to user's preferred white formal UI | chat-widget owner; orchestrator review | completed (UX-01B); updated preview on 8080 for user feedback | 15 |
| 17 | Remove OS dark-mode override so requested white UI stays white | chat-widget owner; orchestrator review | completed (UX-01C), white dark-preference preview on 18096 | 16 |
| 18 | Develop fictional Jitihada University approved FAQ and grounded AI wording locally | chat-backend owner; orchestrator review | completed (JITI-01); 10 local FAQs, no staging import | 04 |
| 19 | Document current local FAQ behavior and provisional citations | project-foundation owner; orchestrator review | completed (JITI-02); mock pages and staging import remain blocked | 18 |
| 20 | Increase white corner-widget typography one step without losing mobile Send | chat-widget owner; orchestrator review | completed (UX-01D), plain preview on 18097 awaiting visual feedback | 17 |
| 21 | Shorten pre-send disclosure and remove demo branding without hiding sample status | chat-widget owner; orchestrator review | completed (UX-01E), preview on 18098 for user feedback | 20 |
| 22 | Remove technical localhost/port URL from visitor disclosure | chat-widget owner; orchestrator review | completed (UX-01F), plain preview on 18099 awaiting visual feedback | 21 |
| 23 | Build controlled preview-first FAQ import path | chat-backend owner; orchestrator review | completed (JITI-03); staging still empty, no remote write | 18, 19 |
| 24 | Add accessible human-support action menu and concise disclosure | chat-widget owner; orchestrator review | completed (UX-01G), plain preview on 18100 awaiting user feedback | 22 |

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

UX-01C reproduced the dark-preference bug, made white surfaces unconditional, and serves the fixed build at local port 18096; older preview ports are closed. Ask user to confirm the white display before any next UI/UX step. Chat-state/email-form UX later; PROD-01 release blockers unchanged. No commit or push without explicit authorization.

UX-01D increased white-widget base type to 16px and small supporting copy proportionally; actual browser checks pass and a fresh plain preview URL is running on port 18097. Await user visual feedback before further UI/UX changes.

UX-01E condensed visible copy, removed demo-facing branding, and kept truthful safety, transmission, guest persistence and fictional knowledge warnings. Current plain local preview on port 18098; await user wording approval. Do not imply real knowledge or email delivery.

User asked to remove the displayed origin URL/port from the disclosure; UX-01F changes visitor copy only and retains actual origin checks and data-sharing semantics.

JITI-01 was confined to backend-owned paths; review its provisional source routes before a separate controlled Turso staging import or widget mock-site task. The existing UX-01C and uncommitted widget work remain untouched.

JITI-01 local backend changes have now passed all required gates; JITI-02 corrects README. Staging remains empty, so source pages and a safe import are future reviewed tasks rather than a performed migration.
