# UX-01C: white interface even when the OS is in dark mode

Status: resolved

Owner: chat-widget worker; context: chat-widget. Claim before editing. Read `AGENTS.md`, `.harness/goal.md`, relevant web CSS, browser test and existing git diff. Allowed edits: `web/**` and this issue. No backend, academic reports, harness shared state, live services, commits or pushes.

## Reproduction / goal

The user sees a dark widget despite asking for white. `web/assets/widget.css` and `loader.css` still apply `prefers-color-scheme: dark`, proven in `/tmp/opencode/ux-01b-white-browser-20261020/mobile-dark-ready.png`. The prior UX-01B test accepted dark surfaces, so it missed the user's actual requirement. Make the **entire corner widget white in both light and dark browser/OS modes**: header, conversation, disclosure and composer; blue accents remain. Keep the launcher readable/consistent and sample host white. Do not only change the light theme.

## Acceptance

- A real browser with `prefers-color-scheme: dark` and light both computes white backgrounds for header/conversation/composer and white or neutral sample page. Dark browser form controls/text stay legible on white; set explicit `color-scheme: light` on the widget/host styles as needed. The dark theme must not reappear when an OS or browser preference is dark.
- Preserve UX-01 pinned composer, transcript scrolling, disclosures, escalation consent and honest outcomes, source links, accessibility and all embedding security invariants.
- First change browser regression so the exact dark-preference case fails against current styles (red), then fix to green. Run `make format lint static unit contract`, `git diff --check` and actual-browser geometry/interaction test; capture screenshot with dark preference showing WHITE surfaces. No :18095 server restart by worker; orchestrator handles new port for user review.

## Worker brief

Owner: chat-widget; permitted changes only `web/**` and this issue. Preserve iframe capability isolation, validated message routing, untrusted text as data, consent gate, truthful delivery wording and pinned composer/scrolling layout. No API/data-shape changes. Test actual dark and light browser preferences for white host, header, conversation, disclosure and composer, and light form-control scheme; record red and green results and screenshots. Run required checks and diff check; no live services or commits.

## Regression evidence (red, before CSS changes)

`node web/tests/widget-visual.browser.mjs /tmp/opencode/ux-01c-assistant /tmp/opencode/ux-01c-red-dark-20261020` exited 1. `results.json` confirms dark media in host and iframe, then fails `mobile-dark: white window surfaces and light form controls` after verifying frame and composer geometry. Baseline was the pre-fix built-in widget/loader CSS; browser profile and database are isolated in that evidence directory.

## Result / validation

Removed browser-preference dark palettes from `web/assets/widget.css` and `web/assets/loader.css`; set iframe root and loader embed to `color-scheme: light`. Sample host inherits the explicit light scheme and white background from `widget.css`. Browser regression checks white regions and light form controls in both preference modes and opens the actual sample host under a dark preference. No behavioral, disclosure, security or contract changes.

`/tmp/opencode/ux-01c-green-verified-20261020/results.json`: 63 passed, 0 failed, including real click/consent/recorded-not-sent interaction and mobile-dark geometry. Screenshots: `mobile-dark-ready.png` (white header, conversation, disclosure and composer), `sample-host-dark.png` (white actual sample host), plus long-transcript and consent-error screenshots. `make format lint static unit contract`: first invocation passed format/lint/static but unit transiently failed in concurrently edited backend (`TestContractHTTPClarificationWithdrawalAndNoHalfTurn`: expected 503, received 200); rerun `make unit contract` passed. `git diff --check` passed. No live services, commits or pushes.

## Orchestrator preview

Primary reviewed the dark-preference white screenshots. Built `/tmp/opencode/ai-widget-preview/assistant-always-white-18096`, stopped the previous 18095 preview and launched this build on fresh port 18096 with a separate local demo SQLite database and external AI credentials disabled. Verified HTTP 200, only 18096 listens among the three preview ports, and served widget CSS checksum equals the working-tree CSS. Concurrent backend/source changes from other work were observed and not altered by this UI task. User visual confirmation is pending; no production claim or commit/push.
