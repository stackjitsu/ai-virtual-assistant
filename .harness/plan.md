# Active Plan

## CI-01 closed on local evidence; CI-02 created; AUTO-01 unblocked — 2026-11-03

User approved the recommended sequence. The coordinator checkpointed the accepted CI implementation/repair as commit `84bfd72` ('Add deterministic validation CI with bounded browser containment') on `tooling/ci-autonomy` in assistant-ci: ~55 tracked files — source (workflow, browser_check + tests, tasks-schema tests, docs, manifest) plus briefs/handoffs/JSON-text evidence; binaries, browser profiles, screenshots, sqlite and 16MiB executables remain deliberately untracked. CI-01 is marked completed: acceptance criteria are met on local gates plus independent review; explicitly NOT claimed: hosted execution, an independent happy-path browser rerun, or a green static on the worktree's historical ledger. CI-02 (hosted deterministic PR validation follow-up, pending user scheduling) was added; AUTO-01's CI-01 dependency is satisfied and its dispatch follows the integration push. The CI tracker row was removed from blockers.md as resolved.

## CI-01 repair independently accepted — 2026-11-03

Assistant-ci-review completed its re-review and recommends ACCEPT: P2 containment closed (finite stage bounds, owned process-group cleanup, deterministic deadline ordering; three independent executions of the process suite passed) and P3 ownership closed (workflow ownership synchronized in context map and manifest). No new blocking defect; happy-path independent browser success was NOT demonstrated in the review environment and is not claimed; static remains blocked by exactly the four historical empty-evidence ledger entries on the CI worktree's base. Repair acceptance is not yet full CI-01 completion: remaining gates are (1) user authorization to commit the uncommitted repair on tooling/ci-autonomy, (2) user decision on hosted GitHub execution, and (3) ledger truth-up already tracked at INTEG-01 acceptance. AUTO-01 stays gated until CI-01 completes.

## Admin design review complete — KDF upgrade dispatched 2026-11-03

User delegated the ADMIN-01 design decision ("security should be there", benchmark other ticketing systems, cherry-pick). Coordinator reviewed the draft contract against OWASP Password Storage and Zammad/Linear patterns. Verdict: accepted with one modification — primary password KDF upgraded to Argon2id (OWASP m=19456/t=2/p=1) via approved `golang.org/x/crypto` dependency, with the draft's stdlib PBKDF2-600k kept as the documented fallback; all session/CSRF/throttle/CAS/audit/privacy properties retained as drafted; all-staff visibility accepted for the invite-only v1 with roles recorded as future work. Dispatched admin contract worker on bounded revision brief `.scratch/admin-dashboard/issues/05-kdf-revision.md` in assistant-admin (docs-only change, gates must be rerun after the earlier quota-blocked runs). No admin runtime exists; implementation briefs 02/03/04 remain undispached until this revision hands off.

## Resume — 2026-11-03: integration delivered; validation and CI re-review re-dispatched

- INTEG-01 owner handoff verified: merge commit `db38c32` on `integration/ui-rating` with first parent `6507570` (approved UI checkpoint) and second parent `eface12` (accepted rating source). Conflicts were resolved without ours/theirs replacement (`.harness/` byte-identical to the first parent; web/contract changes combined additively); one supplemental trusted-input regression covers close/reopen around an unsaved rating. Owner gates format/lint/static/unit/contract plus fuzz-smoke/fuzz-full and browser suites rating 61/0, sandbox-send 48/0, visual 208/0 all passed; handoff `.scratch/automation/ui-rating-integration/handoff.md` in the integration worktree. Owner results are not acceptance.
- The independent combined-tree validator claimed the brief on 2026-11-02 14:35 and died with the tmux server leaving no evidence beyond its claim line and the copied owner handoff. A fresh independent validator is re-dispatched to re-claim and validate exactly `db38c32` against the same brief in assistant-ui-rating-validation.
- CI-01 bounded repair was delivered after two implementation rounds (assistant-ci worktree, uncommitted at 1253287): format/lint/unit/contract/fuzz-smoke plus targeted tooling and real-browser/crash-fixture checks passed, with the recorded static blocker limited to the four historical empty-evidence ledger entries at that base. Repair handoff `.scratch/automation/ci-repair/handoff.md`. An independent re-review is dispatched with the new brief `.scratch/automation/ci-re-review-brief.md`.
- Worker model note: `openai/gpt-6.1-sol` auth check returned invalid on 2026-11-03; `openai/gpt-6-astra` is verified ready (oauth) and is used for both dispatches, matching earlier successful owned workers.
- Still not authorized: main merge, deployment, hosted GitHub execution, admin runtime implementation, staff email/WhatsApp transport, public release. After combined-tree independent acceptance, the only authorized advancement by this step is fast-forwarding and pushing the feature branch.

## UI approved and checkpointed; combined integration next

User approved the running UI preview and requested proceeding. Reviewed UX-02 handoff and source diff; committed approved UI/Escape implementation, contract and selected durable evidence as `191910a` on feat/conversation-rating. UX-02 completed with explicit user acceptance and known compact/landscape limits retained. Do not mix CI/admin drafts into UI integration. Checkpoint coordination records separately, then delegate merge of accepted rating branch into isolated integration branch; widget owner resolves application/test conflicts within exact web/contract scope, no blind ours/theirs. Independent combined-tree gates/browser/full fuzz required before feature fast-forward/push. No main merge or deployment.

## Current UI completion — user visual acceptance

User requested finishing the UI and will validate it personally. UX-02 dispatched to `assistant-widget-finish` in the main feature worktree: sole web writer, preserve existing edits, complete visible close wiring/styles/tests and handoff labels; no rating import, commit, push, merge or deployment. Brief `.scratch/widget-ux/issues/08-close-button-and-labels.md` now includes explicit evidence scope and preview authorization. Worker must pass technical gates/trusted-browser checks and leave a credential-free loopback preview at unused port 18102 (or documented alternative) in `assistant-widget-preview`. URL is not claimed running until verified. Handoff `.scratch/widget-ux/handoff-ux-02.md`. User visual approval gates next UI commit/integration. Other workstreams remain separate; rating branch already pushed, CI repair still needs independent re-review.

## Rating branch pushed; merge held — 2026-11-02

User green-lit proceeding. Pushed `validation/rating-acceptance` and verified remote head equals `eface12be8e508a31efa6a8d1ceb7bd2daf07be5`; upstream tracking configured. Remote feature branch `feat/conversation-rating` does not exist; remote main currently points to 40773a3e09e83c584baadc6dae18a3482991ac67 (not inspected/accepted for integration). Local feature branch is an ancestor of the rating branch (0 local-only / 11 rating-only commits), but its checked-out worktree has overlapping uncommitted harness/UI edits including incomplete UX-02. No stash, reset, overwrite, feature-branch push or merge performed. Next: obtain explicit scope for checkpointing/finishing main-tree work before feature integration; do not merge into remote main blindly. CI remains pending independent re-review.

## Earlier local rating acceptance committed — 2026-11-02

Reviewed final RATE-04 handoff and accepted its bounded recovered-rating scope. Created local branch `validation/rating-acceptance` in assistant-rating-acceptance and committed `eface12` (`Strengthen rating durability and trusted-browser acceptance`). Commit includes only the two reviewed test files plus selected durable acceptance handoff/logs/JSON results (no executable, SQLite database, browser profile or screenshots). All source checks were run before this evidence-only commit; no test source changed after validation. RATE-04 independent acceptance is completed; RATE-02/03 integration into main feature tree remains pending separately. Main feature branch and its existing application/UI edits were untouched. No push or merge performed. Low-severity runner-profile cleanup is not an acceptance blocker. CI repair remains unaccepted until independent re-review. Next user decision: whether/how to push and integrate the local rating branch; do not assume permission.

## Earlier local commit permission — 2026-11-02

User authorized committing only if safe; push and merge are separate future decisions. Independent rating execution logs show all common gates/full fuzz passed (seven mutation campaigns), repeated rating race tests passed ten times, browser rating 52/52, sandbox-send 22/22, visual 113/113. Validator reports no acceptance blocker and is finalizing handoff; review before committing only the test/evidence batch on isolated local branch `validation/rating-acceptance`. CI remains unaccepted pending repair/re-review. Main-tree UX-02 is incomplete: header close-button markup exists but click wiring/styles/tests remain pending, so do not sweep main-tree changes into a commit. Preserve all uncommitted work.

## Earlier acceptance/repair dispatch — 2026-11-02

Reviewed the backend test diff and complete browser regression plus all three handoffs. RATE-02/03 workers delivered test-only changes; reported passes are not independent acceptance. Dispatched `assistant-rate-acceptance` in new isolated `/home/me/Documents/agent-worktrees/assistant-rating-acceptance` at 1253287: mechanically assemble only supplied test patches, independently review/rerun common gates, full fuzz, trusted-input rating plus sandbox-send/visual regressions. Backend patch SHA256 `98bc40d3700374850f3c8bdc29a2122a418d2c37c2bad6c2fd9625912adb1ad6`; browser patch SHA256 `dab81ae7b84b3517a48eadc3ed8fa4278df50eb008a634ddb910086ae4b3efbc`. Main-tree Escape work is excluded and not implicitly accepted. Brief `07-rating-independent-acceptance.md` under `.scratch/automation/issues/`.

CI review recommended hold: P2 adapter can hang when Chromium dies by signal; P3 workflow ownership missing from context map. Dispatched `assistant-ci-repair` in existing assistant-ci worktree with allowed foundation adapter/tests/docs paths only, including context map; no web changes. Brief `06-ci-failure-containment.md`. Must demonstrate timeout/process-group cleanup without browser sandbox bypass; independent re-review follows.

Central ledger repair restored genuine DISC-01/RATE-01 evidence pointers from existing issue handoffs; RATE-02/03 remain claimed pending acceptance. CI worktree's historical ledger remains unchanged and can still fail static; no fabricated evidence or schema weakening. Next: review acceptance and CI repair handoffs; do not start AUTO-01 or admin runtime yet. No commit/push/merge/deploy authorized.

## Earlier resume — 2026-11-02

User authorized proceeding with the status report's next work. Started three interactive tmux workers using explicit openai/gpt-6.1-sol; each has executed its first brief-read tool call (not merely opened a session):

| Task | Session | Worktree / allowed scope | Handoff |
|---|---|---|---|
| RATE-02 follow-up | assistant-rate-backend | assistant-rating-review at 1253287; backend tests only | `.scratch/automation/backend-followup/handoff.md` |
| RATE-03 follow-up | assistant-rate-browser | new isolated assistant-rating-browser at 1253287; rating browser/tests only | `.scratch/automation/browser-followup/handoff.md` |
| CI-01 independent review | assistant-ci-review | assistant-ci at 1253287 plus existing CI diff; no implementation writes | `.scratch/automation/ci-review/handoff.md` |

Briefs are `.scratch/automation/issues/03-rating-backend-followup.md`, `04-rating-browser-followup.md`, and `05-ci-independent-review.md`. Each worker has a two-repair limit and must stop at handoff/blocker. Main-tree Escape/UI changes remain untouched. No commit/push/merge/deploy or remote-write permission granted. No polling/heartbeat daemon added.

Recovered ADMIN-01 handoff exists; draft contract/design delivered, not accepted. Static/unit/contract attempts were blocked by historical disk-quota errors; current filesystem check shows 13GB available on home and 3.8GB on /tmp, which does not itself establish quota remediation or passing checks. Review security/policy decisions before any admin implementation dispatch.

Next: inspect completed follow-up handoffs, assign any demonstrated defects to owners, and perform independent RATE-04 validation only on the exact combined reviewed tree. CI must retain honest ledger-schema failures until real evidence/status corrections exist. AUTO-01 remains pending CI acceptance. Goal now acknowledges approved admin design scope; older plan/preview sections below are historical.

## Staff dashboard — approved new feature

User approved separate staff login and ticket management dashboard and requested available UI/UX skills. ADMIN-01 contract/design worker dispatched on feat/admin-contract at `/home/me/Documents/agent-worktrees/assistant-admin`, base 1253287. Brief: `.scratch/admin-dashboard/issues/01-contract-design.md`. Apply UI/UX Pro Max, Design Taste app-screen workflow, and Tailwind v4 tokens. Contract gate precedes backend/UI implementation. Ticket lifecycle, priority, assignment, internal notes and audit included; outbound replies and public signup excluded. CI/rating blockers remain open; no main-tree application changes or commit/push/deploy authorized by this dispatch.

## Approved automation — current decision

Latest resume: tmux server/sessions are absent, but both worker handoffs survived and were copied to `.scratch/automation/handoffs/`. CI-01 implementation remains uncommitted in its isolated worktree; local gates passed except new full task-schema validation, which correctly rejects four completed rating/disclosure records with empty evidence on base 1253287. GitHub execution and independent CI patch review remain outstanding. Rating recovery passed existing local gates and browser scripts, but acceptance is BLOCKED: rating browser tests use synthetic input against a mock API; storage restart/concurrent-write/uncertain-commit assertions need strengthening. Full fuzz not independently rerun. Next dispatch: bounded widget browser and backend test follow-ups, plus independent CI review; do not reimplement rating or declare completion. AUTO-01 remains pending CI acceptance. Original Escape/UI work remains untouched.

User approved CI-01 and AUTO-01. CI-01 foundation implementation dispatched in isolated worktree `/home/me/Documents/agent-worktrees/assistant-ci` on `tooling/ci-autonomy`, based on recovered rating integration `1253287`. Brief: `.scratch/automation/issues/01-ci.md`. No commit/push/merge/deploy authorized by this dispatch. Independent validation is required before acceptance. Startup correction: both initial worker calls failed before implementation because default gpt-5.5 was rejected by the account. Explicit openai/gpt-5.4 was also rejected. Restarted the two owned tmux workers with openai/gpt-6-astra, matching the current coordinator's PI_MODEL/PI_PROVIDER; success must be confirmed from tool activity/handoffs, not session existence. Rating recovery brief: `.scratch/automation/issues/02-rating-recovery.md`; isolated review tree: `/home/me/Documents/agent-worktrees/assistant-rating-review`.

Reconciliation: rating backend/widget commits already exist, and `rating-integration-clean` ends at `1253287`; previous temporary worktrees are absent/prunable and no matching tmux workers remain. Old claimed task statuses below are historical, not proof of running sessions. Preserve the main tree's uncommitted Escape/UI work. Next: review CI handoff, independently validate recovered rating integration, then implement bounded AUTO-01 runtime/skills with two-repair limit and explicit permission gates. No autonomous production work.

## Widget Escape close — completed locally

WIDGET-ESC-01 and WIDGET-ESC-02 are complete: Escape inside the iframe (including the handoff form) or on the focused launcher closes the window, focuses the launcher, and leaves the guest chat/draft intact for reopening. Exact-origin/source message guards and embedding contract were reviewed. Widget owner passed format, lint, static, unit, contract, fuzz-smoke and trusted Chromium checks (31/31); foundation owner passed format, lint, static, unit, contract. Current isolated preview: `http://127.0.0.1:18101/widget/sample-host.html` (local SQLite, no external AI credentials). Existing 18100 preview was left untouched. No commit/push or deployment. Next decision: user review of Escape behavior; rating slice below remains independent.

## Rating slice — integration branch feat/conversation-rating

| Step | Owner | State | Depends on |
|---|---|---|---|
| DISC-01 pre-send disclosure verification | independent validator | completed; wording gap assigned RATE-03 | baseline |
| RATE-01 additive rating contract | foundation | completed | baseline |
| RATE-02 SQLite/Turso rating API | backend | delegated | RATE-01 |
| RATE-03 accessible rating widget | widget | delegated | RATE-01 |
| RATE-04 integrated browser/gate review | independent validator | pending | RATE-02, RATE-03 |

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
