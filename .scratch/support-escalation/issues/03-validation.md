# ESC-03: independent escalation verification

Status: completed

## Completion record (2026-10-20)

Independent end-to-end verification ran against the real tree at HEAD `ac7bdb6cb27ef2461eb3c0ce0471564be4b61755`: **all issue scenarios PASS with real trusted input; no blockers recorded; no implementation edits made.**

### Evidence

- Acceptance document (per-scenario PASS/FAIL, observed values, exact reproduce commands): `/tmp/opencode/ai-escalation-verification/independent-acceptance.md`
- Machine-checkable raw record with per-check pass flags: `/tmp/opencode/ai-escalation-verification/results.json`
- Driver (evidence tooling only, outside the repo): `/tmp/opencode/ai-escalation-verification/verify.mjs`
- Screenshots: `s1-chat-regression.png`, `s2-panel-open.png`, `s3-no-consent-block.png`, `s4-invalid-email-block.png`, `s5-201-outcome.png`, `s6-cap-409.png`, `s9-429-cooldown.png`, `s9-after-cooldown.png` (same directory)
- Byte evidence: `dup1.body` = `dup2.body` (byte-for-byte replay), `pre-restart-ticket.body` (restart persistence byte-comparison)

### Method summary

Real Go backend built from this tree and served on loopback with `STORE_MODE=sqlite` and `OPENAI_API_KEY=''` (never a real value; no live OpenAI/Turso/SMTP/network beyond loopback; browser egress blocked via a dead proxy). Real headless Chromium (Chrome/154) driven via CDP through the actual backend-served sample host page. All visitor actions were CDP `Input.dispatchMouseEvent`/`Input.dispatchKeyEvent` real input; a per-document recorder captured 996 control events across 7 browser scenarios — every one `isTrusted: true`, zero submit events (sandbox forms-disabled held; no synthetic DOM submit/click used as final evidence). SQLite verified read-only via `sqlite3 -readonly`; direct-API checks via real `curl`. Capabilities never recorded anywhere in evidence.

### Per-scenario results

| Scenario | Result | Headline observed values |
|---|---|---|
| Chat send regression (real input) | PASS | exactly 1 POST `/messages` → 200, 2-message clarification pair rendered |
| Click-gated affordance (never auto-opens) | PASS | panel closed across creation, 503 provider failure and reconcile; 0 `/escalation` requests pre-click; opens/closes only on real toggle clicks |
| No-consent block | PASS | exact wording; 0 `/escalation` requests, 0 additional `/api/v1` requests |
| Invalid-email block | PASS | exact wording; 0 network requests |
| Valid escalation 201 + truthful outcome + server-side row | PASS | wire body with raw typed values → 201; truthful sentence verbatim + UTC time + reference in inert text; sqlite row canonical (`visitor@example.com`, trimmed note, `delivery_not_wired`, ticket_number 1) |
| Duplicate replay (UI + direct API) | PASS | same stored ticket + "previously stored request, replayed unchanged"; canonical-variant replay same ticket; direct API ×2 byte-for-byte identical 201 bodies (sha256 equal), still 1 row |
| Cap: 5 distinct → 409, replay works at cap | PASS | five 201s (ticket_number 1..5, all `delivery_not_wired`); 6th distinct → server 409 + exact bound wording; replay at cap → 201 with ticket #1 reference; still 5 rows (also survive restart) |
| Capability isolation across conversations | PASS | wrong capability / malformed capability / unknown id → three byte-identical 401 bodies (no existence oracle); own capability → 201 byte-identical stored ticket; nothing stored by denied attempts; message history isolation 401 |
| Persistence across restart | PASS | SIGTERM restart on same sqlite: history GET 200 with identical pair; ticket replay 201 byte-for-byte identical to pre-restart body (sha256 equal); row + turns survive; fresh page chat + escalation work after restart; 5 cap tickets survive restart #2 |
| 429 rate limit + Retry-After honored by widget | PASS | 10 attempts/min: attempts 1–10 → 201 with one stable reference (1 sqlite row); 11th → 429 with `Retry-After: 60` + exact wording; send button disabled during cooldown; chat send still works during escalation cooldown (separate windows); button re-enabled after ~62 s and next click → 201 replay |

### Exact server command used

```
go build -o /tmp/opencode/ai-escalation-verification/assistant /home/me/Documents/formartting/Working_Milestone_Files/ai-virtual-assistant/cmd/assistant
env STORE_MODE=sqlite SQLITE_PATH=/tmp/opencode/ai-escalation-verification/assistant.sqlite OPENAI_API_KEY='' TURSO_DATABASE_URL='' TURSO_AUTH_TOKEN='' ASSISTANT_ORIGIN=http://127.0.0.1:35953 LISTEN_ADDR=127.0.0.1:35953 WIDGET_PARENT_ORIGINS=http://127.0.0.1:35953 /tmp/opencode/ai-escalation-verification/assistant serve --seed-demo
```

(port reserved dynamically per run; the driver performed three SIGTERM restarts against the same SQLite file and finished with zero leftover owned processes)

### Notes for the orchestrator

- No blockers for ESC-01 or ESC-02; observed behavior matched both completion records in every case, including exact wordings, byte-for-byte replay, cap semantics, isolation 401 identity, restart persistence and cooldown behavior.
- `.harness/*` untouched per invariant; tasks.json ESC-03 row left for the orchestrator to close.
- Honest limitations (documented in the acceptance file): empty-key chat verification used the provider-independent clarification path and the designed 503 wording; `delivery_not_wired` is the contract-honest single outcome; rate-limit windows are process-local and reset on restart.

Owner: independent validation worker. Context: coordination. Pre-read: `AGENTS.md`, `.harness/goal.md`, `.harness/tasks.json` (ESC-03), `.scratch/support-escalation/spec.md`, `docs/contracts/chat.md` (v1 + escalation sections), `web/README.md` (startup), `.scratch/ai-chat/acceptance.md` (previous validation method), `.scratch/support-escalation/issues/{01-backend.md,02-widget.md}` (what was built and why). Allowed edits: scratch evidence under `/tmp/opencode/ai-escalation-verification/**` and this issue only. No implementation edits anywhere; if a defect is found, record it as a blocker for the owning context and stop.

## Task

1. Claim this file (Status: claimed) before writing any evidence.
2. Build and run the real backend locally (sqlite store, loopback only, no live services, no credentials) following `web/README.md`. Then drive a real Chromium through the actual sample host page with REAL trusted input (CDP `Input.dispatchKeyEvent`/`dispatchMouseEvent`, `isTrusted` verified — no synthetic DOM submit), exercising:
   - Chat send still works end to end after the widget changes (regression guard)
   - Escalation affordance is click-gated only; it never opens by itself
   - Submitting without consent → client-side block, zero network request, truthful wording
   - Invalid email → client-side block, zero network request
   - Valid escalation → 201 with the server's truthful `delivery_not_wired` sentence, timestamp and ticket reference; a ticket row exists server-side
   - Duplicate resubmission of the same email+note → replayed stored ticket, byte-for-byte outcome, no new ticket row
   - Cap behavior: record distinct valid escalations up to the 5-ticket bound, then a 6th distinct request → 409 with the bound wording; replay of an existing pair still works at the cap
   - Server-side isolation: the escalation is capability-bound; a different conversation's capability cannot access it (direct API check is acceptable here in addition to UI)
   - Persistence: restart the server process; the ticket survives (fresh page, history intact, escalation state server-side)
   - Rate limiting: rapid escalation attempts → 429 with Retry-After honored by the widget
3. Capture evidence as machine-checkable output: per-scenario PASS/FAIL with observed values (status codes, ticket ids, DB rows via sqlite query, screenshots where useful) into `/tmp/opencode/ai-escalation-verification/independent-acceptance.md`. Include the exact commands to reproduce.
4. Update this issue with Status, evidence path, and any blockers with evidence.

## Acceptance

- Every scenario above has a real-input PASS with observed evidence, or a blocker recorded for the owning context
- No synthetic-submit final evidence; no claims without observation
- No implementation edits, no commits, no live services
