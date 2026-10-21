# UX-01B: white corner-widget surfaces

Status: completed by UX-01B chat-widget worker (2026-10-20); awaiting user visual feedback

Owner: chat-widget worker. Context: chat-widget. Claim before editing; read `AGENTS.md`, `.harness/goal.md`, `docs/contracts/chat.md`, `web/assets/widget.css`, `web/assets/loader.css`, `web/assets/widget.html`, existing git diff and UX-01 issue. Allowed edits: `web/**` and this issue. No commit/push, live providers, backend or harness edits.

## Goal

User explicitly rejected the blue/off-white chat-window surfaces and wants **white**. Refine the existing UX-01 layout into a predominantly white, restrained formal interface. Header, conversation, disclosure/card and composer should feel like one white window distinguished by subtle gray borders/shadows/spacing, with blue reserved for primary Send action, links and focus states. Preserve hierarchy and readability without making the UI look flat or losing visible focus/error/success states. The corner launcher may stay blue as the entry point unless its open-state presentation clashes; avoid substantial redesign beyond surface color.

## Invariants / acceptance

- Preserve fixed header + scrollable conversation + persistent composer structure, full pre-send disclosure, text content, email form/consent, chat/security/loader contracts, light/dark support unless the user later explicitly changes dark behavior. Do not add login/dashboard or alter backend. Prior user UI-01/UX-01 changes are uncommitted work; never wipe them.
- At default light theme and real 360×640 + desktop: computed backgrounds of header, conversation and composer are white (#fff / rgb(255,255,255)) or visually white; blue appears as accents, not a large filled header. Include a browser assertion for this exact previous mismatch; updated safe screenshots and ready-state Send visible without scroll. Confirm dark mode still legible with appropriate dark surfaces, not forced white.
- Run `make format lint static unit contract`, browser regression at desktop/mobile/dark with no live credentials, `git diff --check`. Report exact file changes/results and screenshot paths; no server restart (orchestrator manages port 8080).

## Worker result (2026-10-20)

- Changed `web/assets/widget.css` (white light-theme header, conversation, disclosure and composer; neutral message surfaces and retained dark palette), `web/assets/loader.css` (white iframe fallback), `web/tests/widget-visual.browser.mjs` (browser-computed light/dark surface regression), and this brief. Existing UX-01 markup, structure, composer scrolling, chat behavior and disclosures remain intact.
- `make format lint static unit contract` and `git diff --check`: PASS. No fuzz run: no parsing or state transitions changed. Offline separate-origin Chromium/local SQLite, no provider credential or external network: 46/46 checks PASS at desktop, 360×640 mobile, dark mobile and 180×320 compact viewport, including visible composer, disclosure, scrollable transcript, consent gate and desktop real-click clarification/recorded-not-sent outcome.
- Safe screenshots: `/tmp/opencode/ux-01b-white-browser-20261020/{desktop,mobile,mobile-dark,zoom-200}-{ready,long,email-consent}.png`; results: `/tmp/opencode/ux-01b-white-browser-20261020/results.json`. Ready and synthetic-history/consent-error screenshots contain no visitor input or credentials; do not share the browser profile or SQLite database in that directory. Visual review confirms white light-theme surfaces and legible dark mode. The 180×320 viewport approximates 200% zoom rather than testing browser zoom itself.
- Did not restart or alter the orchestrator-owned :8080 preview; its embedded assets will remain stale until the orchestrator rebuilds/restarts it. No live service/deployment, commit or push.

## Orchestrator preview

Reviewed light desktop/mobile screenshots and white surface assertions. Built `/tmp/opencode/ai-widget-preview/assistant-ux01b`, replaced only the existing local 8080 preview, reusing its local SQLite database without reseeding. Verified HTTP 200, port 8080 served by `assistant-ux01b`, and served widget CSS checksum equals `web/assets/widget.css`. Awaiting user visual approval; no production claim or commit/push.
