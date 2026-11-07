# ADMIN-01 — Staff dashboard contract and design

Status: claimed
Owner: delegated foundation/design worker

## Approved scope
Separate staff-facing admin dashboard alongside the existing visitor widget. Invite-only staff login/logout, ticket inbox with search/filter/sort and paging, ticket detail with relevant conversation/consent/contact/delivery status, assignment, priority, internal notes, open/in-progress/resolved and reopen, audit history. Issue means the problem in a support ticket, not a separate issue tracker. No outbound reply, attachment, WhatsApp, public registration or deployment in this slice.

## Task
Produce the contract and UI design direction before runtime implementation. Read repository AGENTS.md and its relevant architecture/testing/delegation references and actual backend/store/widget before specifying interfaces. Identify existing auth dependencies rather than assuming a new framework or provider. Recommend a concrete compatible staff-auth approach and bootstrap/recovery flow; flag any operator decisions. Do not build fake login security.

Read and apply these skills using absolute paths:
- /home/me/.agents/skills/ui-ux-pro-max/SKILL.md: run design-system search, verify relevance; use accessibility guidance.
- /home/me/.agents/skills/design-taste-frontend/SKILL.md and full references/app-screens.md: app draw, not marketing-page workflow.
- /home/me/.agents/skills/tailwind-design-system/SKILL.md and references/details.md as needed: Tailwind v4 CSS-first tokens and compiled local CSS scoped to admin, not a widget migration or CDN runtime.
Follow the actual skills' commands. Read any additional skill if auth/design decisions trigger it. Do not introduce a hosted backend dependency merely because a skill recommends one.

## Deliverables / permitted paths
Allowed writes only .scratch/admin-dashboard/**, docs/contracts/admin.md, design-system/admin/**. Create a spec, contract draft, design direction, and separately numbered ADMIN-02 backend, ADMIN-03 UI, ADMIN-04 independent-validation issue briefs under .scratch/admin-dashboard/issues/. Do not edit central ledger or existing shared contracts; propose their necessary updates in the handoff. Scope of implementation ownership must be proposed with existing context owners.

Contract must cover server-side staff authorization on every endpoint, separate staff sessions (HttpOnly/Secure cookies, CSRF/origin validation, expiry/revocation, login throttling), no guest capability accepted as staff identity, safe password handling if applicable, invite/bootstrap and disabled users, bounded filters/pagination and notes, concurrency/lost-update protection, audit integrity, additive local SQLite/Turso-compatible migration, cache/log/privacy boundaries, safe conversation access, and no internal data in visitor responses. Keep email delivery distinct from workflow status.

Design: formal white interface, restrained blue accents consistent with visitor UI; ticket-first not decorative KPI dashboard. Screen questions: who needs attention, what happened in this request, what staff action is needed. Desktop list/detail, separate mobile detail preserving list state, accessible sorting and keyboard controls, explicit save/error/conflict/session-expired states, loading/empty/no-results, readable contrast/focus and mobile touch targets. No automatic dark override. Specify login/invite states without pretending auth exists. Fictional samples must be labelled, and counts computed from sample rows. No runtime UI required at this contract gate.

## Boundaries and evidence
No application/runtime/schema changes yet. No edits to other worktrees, commits/pushes/merges, credentials, remote databases/services or deploy. Public reference/build-resource downloads allowed; no global installs or nested agents. Base is recovered rating integration 1253287; uncommitted Escape patch is NOT included. CI/rating acceptance blockers remain unresolved and are not this task.

Run applicable nonmutating format/lint/static checks; distinguish pre-existing failures and doc-only evidence. Stop after 30 minutes or two repair rounds. Write .scratch/admin-dashboard/handoff.md with base SHA, actual changed paths, search/draw output pointers, recommendations, unresolved decisions, check results and implementation-ready next steps. Do not mark centrally completed or claim dashboard built. Coordinator reviews before backend/UI dispatch.
