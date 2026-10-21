# UX-01: visibly redesigned corner-widget layout

Status: completed (first layout checkpoint; awaiting user visual feedback)

Implementation invariant: keep the existing contract, iframe handshake, capability confinement, and chat/email request semantics unchanged. Only split the iframe into a compact header, independently scrolling content (disclosure, status, history, email form), and a persistent composer. The complete disclosure remains visible in the ready state before sending; at narrow sizes it stays readable by scrolling the content pane without moving the composer. Browser regression checks actual geometry after readiness and after long history, not just screenshots.

Owner: chat-widget worker; context: chat-widget. Before edits read `AGENTS.md`, `.harness/goal.md`, `docs/architecture/context-map.md`, `docs/contracts/chat.md`, `web/README.md`, relevant markup/CSS/loader/core/tests and `git diff`. Claim this brief before editing. Allowed paths: `web/**` and this brief only. Never commit/push.

## Goal (step 1 only)

Replace the current long scrolling page inside the corner iframe with a recognizably different chat-window layout: compact fixed-height header, bounded scrollable conversation/disclosure area and an accessible composer that is visible **without scrolling** in the ready state at 360×640 on a real phone-sized browser and on desktop. Keep the existing corner launcher and embed. Preserve the existing uncommitted UI-01 work; do not replace it wholesale. Email escalation **behavior**, chat requests and backend are not part of this step.

## Required layout behavior

- A new structural hierarchy should be obvious in screenshots side by side with UI-01: clear chat header, conversational center, separated composer/action area. Styling-only changes to colors or radii do not satisfy this task. Avoid framework/build deps or external fonts/network assets.
- AI-risk/no-sensitive-data, transmission/provider, fictional-demo and memory-only guest limitations must still be readable **before the first question** (see `docs/contracts/chat.md`). Do not hide required disclosures behind a collapsed widget before first send or claim that an unconfigured provider sent AI/email. Keep status/errors discoverable and announced to assistive tech.
- Desktop and 360×640: the question field AND Send button appear in the iframe viewport after opening with no iframe scroll. When history grows, the conversation scrolls rather than pushing the composer away. At narrow widths, 200% zoom and long unbroken text, controls remain reachable and do not overlap. Email escalation remains available and truthful; if the layout changes its position, keep full keyboard reachability, consent and outcome visibility.
- Retain exact iframe sandbox/CSP/origin/handshake/capability constraints, data as inert text, normal send, retry/history/error behavior and the pre-existing `widget-core.mjs` work. If resize/postMessage needs to change, discuss the contract and coordinate before editing. Prefer CSS/markup adjustments contained to `web/**` with unchanged protocol.

## Checks / evidence

- Implement browser-level regression for the **specific previous failure**: at 360×640 after real open/ready, textarea and Send button both have nonzero bounding boxes inside the visible iframe without any scrolling; after enough history, composer remains visible while transcript scrolls. Capture desktop + phone screenshots of ready and long-conversation states, compare against prior UI-01 screenshots. Use real browser actions; mock provider only in local loopback fixtures, no live credentials/paid services; do not leak capabilities or visitor text in screenshots/artifacts.
- Run `make format`, `make lint`, `make static`, `make unit`, `make contract`; run `make fuzz-smoke` only if fuzzable parsing/state transitions are changed; report exceptions as blockers. Record changed paths, checks, screenshots, limitations, and whether running server 8080 needs rebuilding; do not edit or restart it yourself (orchestrator will coordinate). No academic report edits or live deployment.

## Worker result (2026-10-20)

- Changed `web/assets/widget.html`, `web/assets/widget.css`, `web/assets/loader.css`, `web/tests/embedding.contract.test.mjs`, `web/tests/widget-visual.browser.mjs`, `web/README.md`, and this brief. Existing UI-01 changes and `web/assets/widget-core.mjs` remain intact. Header, scroll pane and pinned composer are structurally distinct; email form remains in the keyboard-accessible scroll pane. No protocol, sandbox, CSP, or server changes.
- `make format`, `make lint`, `make static`, `make unit`, `make contract`: PASS. `make fuzz-smoke` not applicable: no parsing/state-transition change. Offline separate-origin Chromium with local SQLite/mockless clarification and no provider credentials: 42/42 assertions pass in `/tmp/opencode/ux-01/browser-final/results.json` (desktop, 360×640 mobile, dark mobile, and compact 180×320 layout). Real mouse input checks ready state and consent/outcome; synthetic, non-visitor long history checks independent scrolling, unbroken text and composer geometry.
- Safe ready/long/consent screenshots: `/tmp/opencode/ux-01/browser-final/{desktop,mobile,mobile-dark,zoom-200}-{ready,long,email-consent}.png`. Comparison: prior `/tmp/opencode/ui-01-visual-20261020-final/mobile-ready.png` cuts off Send; new `/tmp/opencode/ux-01/browser-final/mobile-ready.png` shows Send and textarea below the complete visible disclosure. Browser profile/SQLite in evidence directory must not be shared; screenshots use no visitor content or credentials.
- Limitations: 180×320 is a CSS viewport approximation of 200% zoom, not an actual browser zoom-setting check. Desktop real-click clarification and email-outcome behavior pass; no live provider, production readiness or exhaustive host CSS claim. The existing :8080 process was neither touched nor restarted; its Go-embedded assets require the orchestrator to rebuild/restart separately before preview there.

## Orchestrator review / current preview

- Primary independently reviewed new desktop/mobile screenshots and 42/42 browser assertions; reran `make format lint static unit contract` and `git diff --check`, all PASS. The 360×640 ready screenshot shows the textarea and Send button with no scroll; disclosure stays readable in the conversation pane.
- Rebuilt current source as `/tmp/opencode/ai-widget-preview/assistant-ux01`; replaced only the orchestrator-owned local preview on port 8080, reusing its existing local demo SQLite without reseeding. Verified port 8080 serves HTTP 200 and the served widget HTML hash equals `web/assets/widget.html`. Live AI remains disabled. User visual approval is the next decision, not production release.
