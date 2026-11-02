# WIDGET-ESC-02 — Document Escape-close embedding message

- Status: completed; context: project-foundation; owner: contract-tooling worker.
- Goal: describe the newly implemented iframe-to-host close message in the published embedding contract.
- Allowed paths: `docs/contracts/chat.md` and this issue file only.
- Dependencies: WIDGET-ESC-01 implementation and its exact-shape/origin/source tests.
- Invariants: keep existing init/ready/resize semantics, sandbox, confidentiality, and authentication statements unchanged; no application code changes, no secret or guest payload in close message.
- Acceptance: contract explicitly names `{type:"assistant:close",version:1}`, allowed only from initialized widget with validated parent origin, host acceptance only after handshake from exact iframe source and assistant origin, hide/reuse frame, focus launcher, and reject extra keys or wrong versions. Document Escape handling while menu open.
- Required checks: `make format lint static unit contract`; report evidence and blockers. No commit or push.
- Evidence (2026-10-21): `docs/contracts/chat.md` now specifies the exact `{type:"assistant:close",version:1}` iframe-to-host message, Escape with the handoff menu/form open, validated parent-target/init/stopped guards, exact mounted iframe source and assistant origin, initialized handshake, strict two-key/version-1 acceptance, and hide/focus/reuse behavior. Reviewed `web/assets/loader.js` and `web/assets/widget-core.mjs` read-only. `make format lint static unit contract` passed (unit: 57 Node tests; contract: 15 Node tests; Go and tooling tests passed). No blockers; no application code changes, commit or push.
