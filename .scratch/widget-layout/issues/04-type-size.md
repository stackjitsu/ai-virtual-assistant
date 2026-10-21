# UX-01D: increase white widget text size one step

Status: resolved — UX-01D chat-widget worker (2026-10-20); awaiting orchestrator review

Owner: chat-widget worker. Read `AGENTS.md`, `.harness/goal.md`, `docs/contracts/chat.md`, existing UI issues and web CSS/browser test. Claim before editing. Allowed implementation paths `web/**` and this issue only; no backend, harness, academic changes, commits/pushes or live services.

## User request

Increase the widget's font size "up a notch" while keeping the all-white default UI. Use CSS px/rem (web equivalent of their 'pt' wording), not print points. Aim for ~16px base from 15px plus proportional improvements to uncomfortably small disclosure, hint, status and link text; retain hierarchy and comfortable legibility, avoid disproportionate enlargement or removal of required copy. Launcher and sample host should remain coherent.

## Acceptance

- White under both light and dark OS preferences, no URL query parameter to enable it.
- Actual browser on 360×640 and desktop shows visibly larger readable text; composer textarea and Send remain visible without iframe scroll, long transcript scrolls; full disclosure can be read before send. No overlapping/overflowing controls at narrow widths or compact viewport; error/handoff labels remain legible.
- Add/update browser font-size regression for smallest essential text and composer geometry; capture safe white screenshots in both browser preference modes. Preserve chat/consent/security behavior, no CSP/loader protocol changes.
- Run `make format lint static unit contract`, `git diff --check` and offline browser suite; report paths/evidence/limitations. Do not restart preview port 18096; orchestrator will build the new preview at an unambiguous plain URL.

## Worker brief

Owning context: chat-widget. Allowed paths: `web/**` and this issue only. Preserve HTTP, consent, capability isolation, CSP, embed protocol, and all required disclosure copy; typography and browser assertions only. Set the iframe's 15px base to 16px and bring small essential labels/disclosure/hints/status up proportionally without hiding content or dislodging the pinned composer. Verify computed font sizes and visibility/scroll geometry in offline Chromium at desktop and mobile in light and dark preference, retain safe screenshots; run all five make checks and `git diff --check`.

## Result / evidence

- Set 16px iframe base; increased disclosure, metadata, status, composer hint and launcher fonts. Compact-only header and escalation-button sizing preserves a scrollable handoff button without reducing base body text.
- Offline Chromium: **75 passed, 0 failed**, including computed font sizes, 360×640 light/dark white surfaces, composer geometry, independent long-history scrolling, compact 180×320 reachability, consent and real-click clarification/recorded-not-sent checks. Evidence: `/tmp/opencode/ux-01d-visual-final/results.json` and safe `mobile-ready.png`, `mobile-dark-ready.png`, `desktop-ready.png`, `zoom-200-email-consent.png` in the same directory. Screenshots use fixture text; actual screenshot review confirmed white panels and visible mobile composer.
- `make format lint static unit contract`: **all PASS** (Node unit 48/48; Node contract 13/13); `git diff --check`: **PASS**. The make checks ran against a shared worktree with concurrent backend edits; this worker changed only the allowed paths. Browser proof is local offline Chromium, not a production/deployment or actual 200%-browser-zoom claim; 180×320 is a viewport approximation. Preview port 18096 was not restarted.

## Orchestrator preview

Reviewed the 360×640 white screenshots and browser evidence. Built `/tmp/opencode/ai-widget-preview/assistant-type-18097`, stopped only prior preview on 18096, and started a fresh local demo SQLite on port 18097 with AI credentials disabled. Verified HTTP 200 and served CSS byte checksum equals working-tree CSS. The plain sample-host URL needs no parameters. No commit/push; awaiting user visual feedback.
