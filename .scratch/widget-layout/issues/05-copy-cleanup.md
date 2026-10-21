# UX-01E: concise disclosures and no demo branding

Status: resolved

Owner: chat-widget worker. Claim before editing; read AGENTS.md, pinned goal, chat contract section on visitor disclosure, relevant widget/sample-host markup, core, tests, and current git diff. Allowed edits: `web/**` and this issue only. No backend/contracts/harness/academic edits, live services, commits or pushes.

## Goal

User finds the current three-paragraph notice too long and asks to remove demo branding. Make visitor-facing chat copy concise, professional and easy to scan; remove "DEMO / AI SUPPORT", "fictional demo" in heading/subheading, and sample-host demo branding. Do not remove factual safety information or falsely imply the current local fictional seed is official organization policy or that real email delivery exists.

## Content invariants

- Before first send, the UI must still plainly say AI can be wrong, avoid sensitive data, current sample knowledge is *not official policy*, questions/recent context reach the app server, bounded/approved excerpts may go to the configured AI provider, and reloading loses guest access but does not delete stored history. Wording may compress these into two short sentences/lines; avoid advertising a hard-coded OpenAI provider when the server supports alternatives. No secrets/raw capability/entire-page contents in display/logs.
- Do not suppress fallback truth: currently server-side AI key may be absent, local email records requests without sending, and sample knowledge is fictional. Keep essential preview/test-site disclaimer on the sample host in short plain language rather than pretending this is a real staffed service.
- Retain exact IDs, regions/labels, consent text/delivery status, DOM inert-text rendering, published embedding behavior and stable CSS layout. Update focused tests currently tied to older literal wording, while asserting essential meanings. Be aware this is a shared worktree with concurrent backend/documentation activity; preserve it and touch only owned paths.

## Validation

Run `make format lint static unit contract`, `git diff --check`, and offline actual-browser layout/interaction checks on white 360x640/light+dark-preference. Capture safe screenshots showing shorter copy and the fixed Send button; no live provider/DB. Report changed paths and evidence. Do not restart the preview server; orchestrator will deploy a fresh local preview URL without query parameters.

## Result

Shortened the pre-send disclosure while retaining AI fallibility, sensitive-data warning, fictional/non-official knowledge, actual app-server origin, possible configured-provider excerpts, and guest-access versus server-history distinction. Removed visible demo branding from the widget and sample host; the sample host still states that it is a fictional preview without staffed support, that AI needs a server-side key, and that support requests do not send email. Consent and delivery wording are unchanged.

Checks: `make format lint static unit contract` and `git diff --check` passed. Offline Chromium visual/interaction check passed 75/75 including 360×640 white light and dark preference. Screenshots: `/tmp/opencode/ux-01e-browser-20261020/mobile-ready.png`, `/tmp/opencode/ux-01e-browser-20261020/mobile-dark-ready.png`, `/tmp/opencode/ux-01e-browser-20261020/sample-host-dark.png`. The browser script launched and cleaned up its own isolated loopback server; no existing preview server was restarted.

Orchestrator reviewed screenshots, built `/tmp/opencode/ai-widget-preview/assistant-copy-18098`, stopped only the 18097 preview, and launched a fresh credential-free local preview on port 18098. Verified HTTP 200 and served widget HTML checksum matches the working tree. Waiting for user copy approval; no commit/push.
