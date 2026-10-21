# UX-01F: no localhost/port in visitor disclosure

Status: resolved

Owner: chat-widget worker, context: chat-widget. Claim before edits. Read AGENTS.md, pinned goal, contract disclosure requirements, current web core/tests and git diff. Allowed paths: `web/**` and this issue; no backend, foundation contracts, harness, academic files, commits, pushes or live integrations.

## Goal

User says the `http://127.0.0.1:<port>` shown in the Before you ask paragraph is irrelevant. Replace only the visitor-facing transmission copy with concise, accurate provider-neutral language that says questions/recent chat go to the app server and approved/relevant excerpts *may* go to the configured AI provider, without embedding a hostname, localhost, IP or port. Preserve all other disclosure, visual, guest access and consent behavior. The internal origin validation/postMessage/fetch routing must stay unchanged; only the displayed string changes.

## Acceptance / checks

- Browser and unit/contract test assert actual visible notice does not include `location.origin`, localhost, `127.0.0.1` or a port while retaining server and possible provider sharing. Keep the rest of the text honest and current.
- Run `make format lint static unit contract`, `git diff --check`, and offline browser suite; report evidence/changed paths. No restart of existing preview (orchestrator handles it). No remote service calls.

## Worker brief

Owning context: chat-widget. Allowed paths: `web/**` and this issue. Preserve `origin` as the validated iframe fetch/postMessage routing value; change only the rendered transmission sentence, not its source or other disclosure text. Keep the pre-send warning, guest history warning, and consent intact. Unit, embedding-contract and offline browser checks must confirm the displayed notice names the app server and possible configured AI provider without showing the actual origin, loopback address, or port. Required checks: `make format lint static unit contract`, `git diff --check`, and offline browser suite. No live integrations, preview restart, commit or push.

## Answer

The displayed transmission sentence now says questions and recent chat go to the app server and approved excerpts may go to the configured AI provider, without interpolating the origin. The existing origin-dependent routing remains untouched. Focused unit, embedding contract and visual-browser assertions cover the notice; one existing escalation assertion was updated to match the revised text. `make format lint static unit contract` and `git diff --check` pass. Offline visual browser: 75/75 (`/tmp/opencode/ux-01f-visual-20261020/results.json`); offline sandbox send browser: 22/22 (`/tmp/opencode/ux-01f-send-20261020/`). No preview restart or live integrations.
