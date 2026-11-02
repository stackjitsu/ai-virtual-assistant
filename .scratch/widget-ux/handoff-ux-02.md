# UX-02 handoff — implementation delivered; user visual acceptance pending

## Scope and preservation

- Delegated widget implementation owner; no nested agents, commits, pushes, merges, deployments, dependency changes or remote writes.
- Main tree: `feat/conversation-rating`, HEAD `8fb8e2acd29c3fc2a9747ba3bc6bcae4efa5ec86` (unchanged). **This HEAD predates recovered rating. No rating implementation imported or validated.**
- Before editing: claimed `.scratch/widget-ux/issues/08-close-button-and-labels.md`; saved full worktree diff, web diff, complete web tar snapshot and initial web SHA-256 manifest under `.scratch/widget-ux/evidence/ux-02/initial/`.
- Worker-only delta: `.scratch/widget-ux/evidence/ux-02/worker-only.patch`; hashes: `final-web.sha256`; preservation inventory: `preservation.json`. All tracked non-web initial diffs are unchanged (`concurrent-nonweb.json`). No reset/stash/restore used.
- The pre-existing `web/assets/widget.html` close markup and new labels, `web/assets/loader.js` Escape/close handling, and `web/tests/escalation.test.mjs` Escape tests are **byte-identical to the initial snapshot**. All other unrelated web files also match the snapshot.

### Changed web paths vs initial snapshot

1. `web/assets/widget-core.mjs`: add `chat-close` fixture lookup; share `closeChat()` between native button click and Escape. Guard initialization/stopped state; reset only handoff disclosures; keep chat, pending question and support drafts untouched. Exact existing `{type:"assistant:close",version:1}` with concrete validated parent origin.
2. `web/assets/widget.css`: flex header with a nonshrinking 44×44 CSS-pixel tertiary close target, 8px minimum separation, wrapping heading, neutral hover/press and existing visible focus ring. No composer styles changed.
3. `web/tests/harness.mjs`: native-button fixture for `chat-close`.
4. `web/tests/widget.test.mjs`: exact close message, no pre-init/terminal/pagehide close; transcript/question/email/note/consent preservation.
5. `web/tests/embedding.contract.test.mjs`: requested label/header markup and shared close-path packaging assertions; existing exact origin/source/shape rejection retained.
6. `web/tests/sandbox-send.browser.mjs`: trusted pointer/keyboard close activation, exact host-received close message, launcher focus, same iframe/conversation/drafts, visible keyboard focus. Existing Send and Escape regressions retained.
7. `web/tests/widget-visual.browser.mjs`: correct stale menu-only Escape expectation to existing whole-window contract; close geometry/labels; more viewports; explicitly record inherited landscape host clipping rather than claim full fit there.
8. `web/README.md`: requested channel wording and accurate close/Escape/outside-click/reopening behavior.

Only additional writes are this handoff, the assigned issue and its authorized evidence directory. No harness/contracts/backend/CI/academic files edited.

## Technical results

Evidence root: `.scratch/widget-ux/evidence/ux-02/`.

| Command | Exact final summary | Full evidence |
| --- | --- | --- |
| `make format lint static unit contract fuzz-smoke` | `format: PASS`, `lint: PASS`, `static: PASS`, `unit: PASS`, `contract: PASS`, `fuzz-smoke: PASS`, `EXIT_CODE=0` | `checks/required-final.log` |
| Node ordinary tests within unit | `tests 59`, `pass 59`, `fail 0` | same log; Go race suite and 10 adapter tests also pass |
| Node contract tests | `tests 15`, `pass 15`, `fail 0` | same log; backend contract race tests pass |
| `node --test web/tests/widget.test.mjs web/tests/escalation.test.mjs web/tests/embedding.contract.test.mjs` | `tests 74`, `pass 74`, `fail 0` | `checks/focused.log` |
| `timeout 150s node web/tests/sandbox-send.browser.mjs "$PWD/.scratch/widget-ux/evidence/ux-02/assistant" "$PWD/.scratch/widget-ux/evidence/ux-02/sandbox-round2"` | `{"passed":48,"failed":0}`, `EXIT_CODE=0` | `checks/sandbox-round2.log`, `sandbox-round2/results.json` |
| `timeout 150s node web/tests/widget-visual.browser.mjs "$PWD/.scratch/widget-ux/evidence/ux-02/assistant" "$PWD/.scratch/widget-ux/evidence/ux-02/visual-final"` | `{"passed":208,"failed":0}`, `EXIT_CODE=0` | `checks/visual-final.log`, `visual-final/results.json` |
| `node .scratch/widget-ux/evidence/ux-02/inspect-component.mjs <same binary> <absolute component-inspection directory>` | `{"passed":208,"failed":0}`, `EXIT_CODE=0` | `checks/component-inspection.log`, `component-inspection/*-taste-probe.json`, `*-component.json` |
| `node /home/me/.agents/skills/design-taste-frontend/scripts/lint.mjs --app web/assets/widget.html web/assets/widget.css` | `RESULT: 0 FAIL, 7 WARN`, `EXIT_CODE=0` | `checks/taste-lint.log` |
| `timeout 70s node .scratch/widget-ux/evidence/ux-02/preview-smoke.mjs` | `Preview smoke: 3/3 viewports initialized and trusted pointer close restored launcher focus` | `checks/preview-smoke.log`, `preview/browser.json` |

Environment: Node `v26.1.0`; Chromium `154.0.8037.97` (Arch Linux); Go `go1.26.3 linux/amd64`. Built from this working tree using `env OPENAI_API_KEY= TURSO_DATABASE_URL= TURSO_AUTH_TOKEN= go build -o .scratch/widget-ux/evidence/ux-02/assistant ./cmd/assistant`.

Fuzz-smoke ran all six available targets with `-fuzztime=2s -parallel=2`. Five budgets mostly exercised their cached baseline corpora; do not interpret this as full mutation coverage. Fuzz-full was not required/run: this is a low-risk additional trigger for an existing close path, not a new parser/protocol/release. Node deterministic boundary/property sweep also passes.

Two bounded repair rounds, both browser-test-only: corrected CDP Shift modifier (`8`, not Alt `1`) and refined the landscape measurement to account for the actual inherited host top offset plus iframe border. Retained unsuccessful logs/results in `sandbox-round1`, `visual-round1`, `visual-round2`. No application repair after the initial implementation; no repair beyond the allowed two rounds.

### Trusted-browser assertions

- Actual Go/SQLite backend, not mocked API; different-origin embedding fixtures; sandbox exactly `allow-scripts allow-same-origin`, restrictive CSP retained. No browser sandbox bypass.
- Browser-generated pointer click, keyboard Tab/Shift+Tab traversal, Enter and Space produce three trusted button clicks. Each activation yields exactly one host-received `{type:"assistant:close",version:1}` from the mounted iframe at its exact origin.
- Host hides the frame and restores launcher focus/closed state. Reopening retains the same live iframe execution context with no new conversation creation.
- Completed clarification transcript, unsent question, email, note and consent checkbox survive all three close activations. Disclosures reset closed. No close-triggered chat/escalation POST.
- Trusted Escape from chat and open email form still closes the whole window and restores launcher focus. Launcher Escape still works; other keys do not close.
- Unit tests additionally prove no close message before initialization, after terminal access rejection or after pagehide.
- All seven measured viewports have a 44×44 close target and no heading collision. Narrowest 146px iframe has exactly 8px heading gap and two-line title; normal viewports have one-line title. White header/conversation/composer/form surfaces remain white under dark preference. Existing pinned-composer and independent-scroll assertions remain in place.
- Labels render exactly `Request via email` and `Request via WhatsApp`; WhatsApp remains disabled/no phone collection. Existing email consent gate and recorded-not-sent outcome pass.

## Named UI skills and visual review

Applied UI/UX Pro Max's targeted visible-focus and touch-target guidance (`focus-guidance.txt`, `touch-guidance.txt`), including the distinction between WCAG's 24px web minimum and this task's stricter 44px requirement. Native semantic button, accessible name, visible 3px focus ring, no overlap and no layout-shifting press state.

Applied Design Taste's **single-component** exception: no direction draw, no redesign, keep existing white palette/type/layout. Scoped real-browser inspection uses the skill's actual `probe()` inside the initialized iframe via `inspect-component.mjs`; this is **not a claim that the full standalone-page `inspect.mjs` CLI ran**. The iframe-aware regression supplies the scoped interaction/layout checks that a sample-host-only inspector would miss.

Lint's seven warnings are inherited: four existing uppercase eyebrow references (preserve existing header/brand scope), two input focus shadows (focus feedback, not decorative resting shadows), and one pinned composer shadow (actual floating boundary). No new warned device added; no suppression or unrelated redesign.

Reviewed all four visual states (ready, synthetic long history, handoff menu, consent error) at 1440×900, 768×900, 390×640, 375×640, 360×640/dark, 640×360 landscape, and 180×320 compact. Read every final state's crop through `review/*-states.jpg`; read the two full-resolution keyboard-focus screenshots and `visual-final/sample-host-dark.png`. Also read actual running-preview screenshots at 1440/768/390. Full PNGs remain beside results; synthetic long-history screenshots are layout fixtures, not persisted chat evidence. Duplicate scoped-probe run repeats the same final render; no separate approval inferred.

### Scoped critique — final round

- Seen in one second: an ordinary close cross at the top right of a white support window; primary Send remains distinct below.
- Idea: add the familiar close escape route without competing with the existing chat or changing its visual language.
- Live thing: close and reopen an ongoing guest conversation without losing drafts.
- Layout: header=heading + fixed utility / conversation=scroll pane / composer=pinned.
- Scores (component-only judgment, not user approval): design 7 · usability 8 · creativity 6 · content 8 → overall **7.2**. Conventional component by design; no award-level redesign claim.
- Caps: page imagery/motion floors do not apply to a single control/app fragment. Inherited compact/landscape limits below prevent a flawless usability claim.
- Top remaining observations: (1) at 640×360 the existing host frame clips approximately 6px at the very top; close target remains on screen, but its focus ring may clip there; (2) at the 180×320 approximation the pre-existing single-line escalation Send label can clip, although the button remains reachable and composer stays pinned; (3) the disabled WhatsApp option can need pane scrolling to read its full reason on small screens. No additional scope authorized to redesign these. The host/sample page is naturally overlaid by its corner widget; that is not header-title overlap inside the widget.

## Preview left running

**URL: http://127.0.0.1:18102/widget/sample-host.html**

- Named tmux session: **`assistant-widget-preview`**, running the owned `assistant` process. Port 18102 was unused before startup; no unrelated listener stopped.
- Start command from repository root:
  ```sh
  tmux new-session -d -s assistant-widget-preview "bash '$PWD/.scratch/widget-ux/evidence/ux-02/start-preview.sh'"
  ```
- Exact executable/configuration is in `evidence/ux-02/start-preview.sh`: `env -i`, local SQLite at `evidence/ux-02/preview/assistant.sqlite`, loopback listen `127.0.0.1:18102`, assistant origin and permitted parent both `http://127.0.0.1:18102`, demo seed only. No inherited credentials, live provider, Turso or outgoing email transport.
- Verified sample-host HTTP **200** after startup and again before handoff. All **8 served assets byte-match the working-tree assets**, with SHA-256 recorded in `preview/assets.json`. Actual preview initialized and real-pointer close restored launcher focus at 1440/768/390 (390 also dark/reduced-motion).
- Inspect logs: `.scratch/widget-ux/evidence/ux-02/preview/backend.log`; attach with `tmux attach -t assistant-widget-preview`.
- Cleanup **when visual acceptance is finished**: `tmux kill-session -t assistant-widget-preview` (only this owned session). Disposable executable, SQLite, profiles and screenshots remain under the authorized evidence root; do not publish them as web assets.
- Honest preview limits: relevant FAQ questions requiring inference show unavailable with the empty key; an unsupported question such as “Can you explain quantum entanglement?” produces a persisted clarification without provider access. Email requests can be recorded, but **no email is sent**. WhatsApp is unavailable. No rating UI on this revision.

## Next decision / gaps

**Ready for the user's bounded visual review, not a central completion or visual approval.** No known defect remains in the requested label/close behavior. User acceptance is outstanding. Existing 640×360 host top clip / compact escalation text limits are recorded for chat-widget owner and orchestrator triage; broad responsive/launcher changes were deliberately not made. Real screen-reader testing, real mobile hardware/soft keyboard, actual browser 200% zoom, and other browser engines are not claimed. Stop at this handoff; leave files uncommitted and preview running.
