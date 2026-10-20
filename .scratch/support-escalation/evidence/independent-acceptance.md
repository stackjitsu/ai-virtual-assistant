# ESC-03 — independent escalation acceptance (visitor email escalation)

Date: 2026-10-20
Validator: ESC-03 independent validation worker (not an implementer; no implementation files touched)
Tree: `ai-virtual-assistant` at HEAD `ac7bdb6cb27ef2461eb3c0ce0471564be4b61755` (working tree contained the implementation-owner scratch/harness edits recorded in issues 01/02; none created by this task)
Raw machine-readable evidence: `/tmp/opencode/ai-escalation-verification/results.json` (one JSON record per scenario with per-check pass flags and observed details)
Driver: `/tmp/opencode/ai-escalation-verification/verify.mjs` (evidence tooling only; lives outside the repo)
Overall: **15/15 scenario records PASS, 0 failed checks, no blockers.** No implementation changes were made by this validation.

## Method

- Real Go backend built from this tree (`cmd/assistant`) and served on loopback with `STORE_MODE=sqlite` and an empty `OPENAI_API_KEY` (no live OpenAI/Turso/SMTP/network beyond loopback; browser external egress blocked via dead proxy `127.0.0.1:9` with loopback bypass).
- Real Chromium (Chrome/154.0.8037.97 headless) driven over CDP through the **actual sample host page** (`/widget/sample-host.html`, served by the backend itself) with the real loader → launcher click → sandboxed iframe.
- Every visitor action was real trusted input: `Input.dispatchMouseEvent` for clicks and `Input.dispatchKeyEvent` for typing (per-character key events; Ctrl+A/Backspace clearing also via real key events). A per-document recorder captured `click`/`keydown`/`input`/`submit` events on chat and escalation controls; **996 recorded control events across 7 browser scenarios, every one `isTrusted: true`**, and **zero submit events** (the `allow-scripts allow-same-origin` forms-disabled sandbox held; no synthetic DOM submit/click used as final evidence).
- All final UI evidence (wording, states, references) was read from the real rendered DOM of the iframe; server-side evidence from SQLite (`sqlite3 -readonly`) and real `curl` calls; browser network observations from CDP `Network.*` (Authorization headers/capabilities deliberately never recorded; no capability value appears anywhere in evidence).
- Chat-regression questions used the `needs_clarification` path (200 completed turn, no provider required). The intentionally failing provider path (503, empty key) was exercised in S2 as the server event that must not open the escalation panel. Unknown-origin API 403/logout behavior not exercised here beyond contract-level checks already covered by owner suites.

## Server command used (exact)

```
go build -o /tmp/opencode/ai-escalation-verification/assistant \
  /home/me/Documents/formartting/Working_Milestone_Files/ai-virtual-assistant/cmd/assistant

env STORE_MODE=sqlite \
    SQLITE_PATH=/tmp/opencode/ai-escalation-verification/assistant.sqlite \
    OPENAI_API_KEY='' TURSO_DATABASE_URL='' TURSO_AUTH_TOKEN='' \
    ASSISTANT_ORIGIN=http://127.0.0.1:35953 \
    LISTEN_ADDR=127.0.0.1:35953 \
    WIDGET_PARENT_ORIGINS=http://127.0.0.1:35953 \
    /tmp/opencode/ai-escalation-verification/assistant serve --seed-demo
```

(port was reserved dynamically per run; `verify.mjs` starts/stops/restarts the process itself — three SIGTERM restarts were performed against the same SQLite file)

## Scenario matrix

| ID | Scenario (issue 03) | Result | Key observed evidence |
|---|---|---|---|
| S1 | Chat send regression (end-to-end, real input) | **PASS** | exactly 1 `POST /api/v1/conversations/860136d4…/messages` → **200**; 2-message rendered pair; status "Clarification received: approved knowledge is limited."; 0 `/escalation` requests |
| S2 | Escalation affordance click-gated only; never auto-opens | **PASS** | panel `aria-expanded=false` + form hidden after conversation creation, after a **503 provider failure** ("The AI service is unavailable…"), and after reconcile; **0 `/escalation` requests** before any toggle click; real toggle click opens (`aria-expanded=true`), second click closes |
| S3 | No consent → client-side block, zero network | **PASS** | error exactly "Tick the consent box first. Nothing is recorded for you without your consent."; **0 `/escalation` requests, 0 additional `/api/v1` requests of any kind** (only the page-baseline creation + history GET) |
| S4 | Invalid email → client-side block, zero network | **PASS** | real-clicked consent checked; error exactly "Enter one valid email address of up to 254 bytes, like you@example.org."; **0 `/escalation` requests, 0 additional `/api/v1` requests** |
| S5 | Valid escalation → 201 truthful outcome + server-side row | **PASS** | wire body `{"email":"Visitor@Example.COM","consent":true,"question_note":" Please ring me at three "}` → **201**; UI shows the fixed truthful sentence verbatim + "Recorded Tue, 20 Oct 2026 12:10:58 GMT." + "Request reference: 24696783b4dad4fa36bd58f7beaf25c7."; sqlite row: `email=visitor@example.com, question_note='Please ring me at three', delivery_outcome='delivery_not_wired', ticket_number=1` (conversation `c74d689b…`); rendered ref === stored ticket_id |
| S5b | Duplicate replay: UI identical pair + canonical variant | **PASS** | identical resubmission → 201 replay, same reference, +"This is your previously stored request, replayed unchanged."; retyped `visitor@example.com` / trimmed note (canonical variant) → same stored ticket again; **still exactly 1 ticket row** for that conversation after all three submissions |
| S5c | Duplicate replay byte-for-byte, no new row (direct API) | **PASS** | two identical `POST /escalation` via curl → both **201**, bodies **byte-for-byte identical** (sha256 `83bb0e10b050dede0515c396b89c6f402b81245103dc6cfb6f8cd5eed957daea` for both, files `dup1.body`/`dup2.body`); exactly **1** ticket row created for that (fresh direct-API) conversation |
| S6 | Cap: 5 distinct tickets, 6th distinct → 409, replay works at cap | **PASS** | visitor1…visitor5 → five **201**s, refs `055a7b2c…`, `dc008504…`, `a22c8c40…`, `b5170c68…`, `d8f78963…`; sqlite: 5 rows, `ticket_number` 1..5, all `delivery_not_wired`; 6th distinct → server **409** + exact UI wording "This conversation has recorded its limit of 5 support requests. …"; replay of visitor1 at cap → **201** with ticket #1's stored reference + replay note; still exactly 5 rows |
| S7 | Capability isolation across conversations (direct API) | **PASS** | conversation B's capability on the ticket-bearing conversation → **401**; malformed capability → **401**; valid capability + unknown id → **401**; all three 401 bodies **byte-identical** (sha256 `9c8e7bba…` — no existence oracle); own capability on the same conversation → **201** replay byte-identical to the stored ticket envelope; denied attempts stored nothing (row count unchanged); message history GET with the wrong capability → **401** |
| S8 | Persistence across server restart | **PASS** | SIGTERM restart #1 on the same sqlite file: history GET with the pre-restart capability → **200**, 2 messages, same `request_id`, identical user/assistant texts; escalation meanwhile-ticket `18fbf83dad022ee12b55cb70ea4c92d9` replay → **201** body **byte-for-byte identical to the pre-restart body** (sha256 `3536d3dcf9b2101c22f8194e13c9ed80ecffcf602b3b50d59a05bf2a996884a4` both, file `pre-restart-ticket.body`); sqlite row + `turns` row survive; fresh page + fresh conversation after restart chat-sends and escalates normally (S6's page); restart #2 → the 5 cap tickets still present (row count 5) |
| S9 | Rate limit: burst → 429 + Retry-After honored by widget | **PASS** | 10 rapid same-pair attempts → ten **201**s, single stable reference `3f6cb5020312c8d50f30405ec06b9b92` (sqlite: exactly **1** row); 11th → **429 `rate_limited`** with **`Retry-After: 60`**; UI wording exactly "Too many support requests. Please wait a moment, then send again."; **send button disabled** during cooldown; **chat send still works during the escalation cooldown** (separate windows, 200 completed pair rendered); after ~62 s real time the button re-enables and a real click returns **201** replay with the same reference; still 1 row |

Supporting records: ENV (backend alive/safe-JSON 404, restricted CSP byte-checked: `default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors http://127.0.0.1:35953`), R1/R2/R3 (three clean SIGTERM restarts), X1 (input-authenticity and safety summary).

## Per-scenario details (observed values)

### S1 — chat send regression
- Real mouse click into `#question`, per-character real key events typing "Can you explain quantum entanglement?", real mouse click on `#send`.
- Observed network: exactly **one** POST to `/api/v1/conversations/860136d408a0d758ec0cbfcd072b2b04/messages`, body `{"request_id":"dccbd3e3-1613-4fd2-8641-aebf5bd96d39","text":"Can you explain quantum entanglement?"}`, response **200** (`no-store`, `nosniff`, `application/json`).
- Rendered: 2 messages ("You" + "Clarification · limited knowledge"), no error, no escalation traffic.
- Screenshot: `s1-chat-regression.png`.

### S2 — click-gated affordance
- State right after conversation exists: `toggleHidden=false`, `toggleDisabled=false`, `aria-expanded="false"`, form hidden, email/note empty, consent unchecked. The affordance exists only once a conversation exists (checked via pre-JS DOM vs module state in earlier runs of the driver; the passing contract checks above are the final evidence).
- Provider-failure chat send ("How do I start the demo?", empty API key by design) → `POST /messages` → **503**, UI "The AI service is unavailable. Your question has not been answered. Please retry later.", reconcile via GET history finished ("No saved reply found. Retry uses the same question ID.").
- The escalation panel stayed **closed (`aria-expanded="false"`, form hidden) at every checkpoint** across creation, 503 failure and reconcile; **zero `/escalation` network requests** during the entire scenario.
- Open only after the first real toggle click (`aria-expanded="true"`); closed again after the second real click. Screenshot `s2-panel-open.png`.

### S3 — no-consent block (zero network)
- Real toggle click → typed valid email `visitor-probe@example.com` → real click on `#escalate-send` **without** touching the consent checkbox.
- UI error verbatim: "Tick the consent box first. Nothing is recorded for you without your consent."
- Network: `/escalation` requests **0** before and after; total `/api/v1` requests unchanged from the page baseline (only creation POST + history GET). Screenshot `s3-no-consent-block.png`.

### S4 — invalid-email block (zero network)
- Consent ticked by real checkbox click (verified `checked===true`), typed `invalid@@example.com`, real click on `#escalate-send`.
- UI error verbatim: "Enter one valid email address of up to 254 bytes, like you@example.org."
- Network: `/escalation` requests **0**; no additional `/api/v1` requests. Screenshot `s4-invalid-email-block.png`.

### S5 — valid escalation, truthful outcome, replay (UI + wire + sqlite)
- Wire body observed on the network (raw typed values go to the server): `{"email":"Visitor@Example.COM","consent":true,"question_note":" Please ring me at three "}`.
- Server: **201**, `Content-Type application/json`, `Cache-Control: no-store`, `X-Content-Type-Options: nosniff`.
- UI outcome (inert text, verbatim server sentence + widget-added time/reference):
  `Your request has been recorded. Email delivery is not configured, so no email has been sent.` / `Recorded Tue, 20 Oct 2026 12:10:58 GMT.` / `Request reference: 24696783b4dad4fa36bd58f7beaf25c7.`
- SQLite (`escalation_tickets`, conversation `c74d689b3be6b1995147b85f170d1f5c`): `ticket_id=24696783b4dad4fa36bd58f7beaf25c7, email='visitor@example.com', question_note='Please ring me at three', delivery_outcome='delivery_not_wired', ticket_number=1, recorded_at='2026-10-20T12:10:58.768438594Z'` — canonical (trimmed/lowercased) form stored; rendered reference equals stored ticket_id.
- Duplicate replay (identical typed pair): 201, same ticket reference, replay note shown; canonical-variant replay (`visitor@example.com`, trimmed note): same stored ticket again; ticket row count stayed **1** through all three submissions. Screenshot `s5-201-outcome.png`.
- Direct-API byte-for-byte duplicate (fresh direct-API conversation, capability legitimately obtained by creating it):
  - `POST {…}/escalation` ×2 with identical body `{"email":"visitor@example.com","question_note":"Please ring me at three","consent":true}` → both **201**; bodies identical byte-for-byte: sha256 `83bb0e10b050dede0515c396b89c6f402b81245103dc6cfb6f8cd5eed957daea` (both), saved as `dup1.body` / `dup2.body`.
  - Exactly **1** ticket row for that conversation; row `ticket_id/recorded_at` match the 201 envelope exactly; envelope is the truthful shape (`delivery_not_wired` + the fixed sentence).

### S6 — 5-ticket cap, 409 wording, replay at cap
- Five distinct valid escalations (consent checked once by real click; each email typed via real keys): five 201s with references `055a7b2c101c1daefb4511cb16aed30f`, `dc008504c82ad79da6d78ce98fda5bcc`, `a22c8c401c4c13bccb826ec3a9eeb8af`, `b5170c68f02147b8eac8ec9314dd9b06`, `d8f7896384eeac4d93c768f218e820e2`.
- SQLite for the cap conversation: 5 rows, `ticket_number` = 1,2,3,4,5, all `delivery_not_wired`.
- 6th distinct (`visitor6@example.com`): server **409 `conflict`**; UI wording verbatim: "This conversation has recorded its limit of 5 support requests. Sending the same email and note again shows your stored request; a new request needs a fresh chat."
- Replay of visitor1 **at the cap**: **201** with ticket #1's stored reference + "This is your previously stored request, replayed unchanged."; row count still exactly 5. Screenshot `s6-cap-409.png`.

### S7 — capability isolation (direct API)
- Wrong capability (conversation B) on the ticket-bearing conversation: **401** `{"error":{"code":"unauthorized","message":"Conversation authorization is required."}}`.
- Malformed capability (`zz`): **401**, identical body. Valid capability + unknown id (`deadbeef`×4): **401**, identical body. All three bodies **byte-identical** (sha256 `9c8e7bbafea74188bb858f00489cfa4ba7166f1caf62c89968dcc950c8934427`) — no existence oracle.
- Own capability on the same conversation: **201** replay, body **byte-identical** to the stored ticket envelope (`identical: true`).
- Denied attempts stored nothing: row count unchanged. History GET with the wrong capability: **401**.

### S8 — persistence across restart
- Restart #1 (SIGTERM → clean exit → same `SQLITE_PATH` reopened):
  - `GET /api/v1/conversations/<p>/messages?limit=20` with the pre-restart capability → **200**, 2 messages, same `request_id` (`75a20acf-fae7-46f7-b8e9-e08eaa6519d0`), identical user/assistant texts (server-side history intact).
  - Escalation replay `persistence-check@example.com` / "Restart persistence probe" → **201**, body byte-for-byte equal to the pre-restart 201 (sha256 `3536d3dc…` = `3536d3dc…`; ticket `18fbf83dad022ee12b55cb70ea4c92d9`, `recorded_at` unchanged `2026-10-20T12:11:02.247646105Z`).
  - SQLite: ticket row and `turns` row survive with identical fields.
  - Fresh page after restart: new conversation created, chat send produces the 200 clarification pair and escalations work (observed at the start of S6 and independently by S6's five 201s).
- Restart #2: the 5 cap tickets survive (row count 5 observed post-restart).

### S9 — rate limit with Retry-After honored by the widget UI
- Burst of 11 rapid same-pair real clicks on `#escalate-send` (each waiting for the previous response; all inside one 60 s window):
  - Attempts 1–10: **201** each, single stable reference `3f6cb5020312c8d50f30405ec06b9b92` (first click created, subsequent clicks replayed; sqlite row count stayed **1**).
  - Attempt 11: **429** response with headers `retry-after: 60`, `no-store`, `nosniff`; UI wording verbatim: "Too many support requests. Please wait a moment, then send again."; **`#escalate-send` disabled** (cooldown honored).
- Chat send during the escalation cooldown (real input): **200** completed pair rendered — the escalation-only cooldown left the chat path untouched (separate rate windows).
- After ~62 s real time: send button re-enabled; a real click returned **201** replay with the same reference; still exactly 1 ticket row. Screenshots `s9-429-cooldown.png`, `s9-after-cooldown.png`.

### X1 — input authenticity + page safety
- Recorded control events: S1 79, S2 55, S3 55, S4 47, S5 192, S6 396, S9 172 — **996 total, all `isTrusted: true`**, zero untrusted.
- Submit events: none (native submission blocked by the forms-disabled sandbox; no synthetic submit was used as final evidence anywhere).
- No browser JS exceptions; browser security log had only `warning`-level informational entries about the contract-mandated `allow-scripts allow-same-origin` sandbox itself; **zero error-level entries**.
- Escalation HTTP across the whole run: 22 requests → **20×201, 1×409, 1×429** (each 409/429 asserted against the exact expected scenario).
- Sandbox attribute byte-checked: `allow-scripts allow-same-origin`; served CSP byte-checked (above); no capability value appears in any evidence artifact.

## Exact reproduce commands

```sh
# build the backend from the repo (read-only build; no repo writes)
go build -o /tmp/opencode/ai-escalation-verification/assistant \
  /home/me/Documents/formartting/Working_Milestone_Files/ai-virtual-assistant/cmd/assistant

# full run (starts/stops/restarts its own backend + headless Chromium on a reserved loopback port;
# requires a fresh evidence directory with no assistant.sqlite present)
node /tmp/opencode/ai-escalation-verification/verify.mjs
```

Direct-API checks (any fresh local conversation; `$CAP` is its one-time creation-response capability, `$CID` its id; the assistant must be running with the env block above):

```sh
CONV=$(curl -s -X POST http://127.0.0.1:35953/api/v1/conversations \
  -H 'Content-Type: application/json' -d '{}')
CID=$(printf '%s' "$CONV" | python3 -c 'import json,sys;print(json.load(sys.stdin)["conversation_id"])')
CAP=$(printf '%s' "$CONV" | python3 -c 'import json,sys;print(json.load(sys.stdin)["capability"])')

# byte-for-byte duplicate replay: two identical calls
curl -s -i -X POST "http://127.0.0.1:35953/api/v1/conversations/$CID/escalation" \
  -H "Authorization: Bearer $CAP" -H 'Content-Type: application/json' \
  -d '{"email":"visitor@example.com","question_note":"Please ring me at three","consent":true}'
curl -s -i -X POST "http://127.0.0.1:35953/api/v1/conversations/$CID/escalation" \
  -H "Authorization: Bearer $CAP" -H 'Content-Type: application/json' \
  -d '{"email":"visitor@example.com","question_note":"Please ring me at three","consent":true}'

# isolation (expect identical 401): wrong capability / malformed capability / unknown id
curl -s -i -X POST "http://127.0.0.1:35953/api/v1/conversations/$CID/escalation" \
  -H 'Authorization: Bearer 00' -H 'Content-Type: application/json' \
  -d '{"email":"intruder@example.com","consent":true}'
curl -s -i -X POST "http://127.0.0.1:35953/api/v1/conversations/$(printf 'deadbeef%.0s' 1 2 3 4)/escalation" \
  -H "Authorization: Bearer $CAP" -H 'Content-Type: application/json' \
  -d '{"email":"intruder@example.com","consent":true}'

# sqlite (read-only)
sqlite3 -json -readonly /tmp/opencode/ai-escalation-verification/assistant.sqlite \
  'SELECT ticket_id, email, question_note, delivery_outcome, ticket_number, recorded_at FROM escalation_tickets'
```

(Capabilities are per-run bearer secrets and are never recorded anywhere in the evidence; each run creates its own conversations.)

## Artifacts

- `results.json` — per-scenario records with per-check pass flags and all observed values above
- `independent-acceptance.md` — this document
- `server-command.txt` — the exact build/serve commands
- screenshots: `s1-chat-regression.png`, `s2-panel-open.png`, `s3-no-consent-block.png`, `s4-invalid-email-block.png`, `s5-201-outcome.png`, `s6-cap-409.png`, `s9-429-cooldown.png`, `s9-after-cooldown.png`
- byte-evidence: `dup1.body` = `dup2.body` (byte-for-byte replay), `pre-restart-ticket.body` (persistence byte-comparison)
- `verify.mjs` — the driver; `backend.log`, `chromium.log` — process logs
- Zero leftover owned processes after the run (verified by `pgrep`).

## Honest limitations

- No live OpenAI/Turso/SMTP/network services were used. `OPENAI_API_KEY` was left empty by design, so: chat verification used the provider-independent `needs_clarification` completed-turn path; the 503 `provider_unavailable` wording was observed as designed (S2 uses it as the server event that must not open the escalation panel).
- `delivery_not_wired` is the only outcome this slice defines; all tickets observed carry it and the schema/contract pin it. Nothing in this evidence claims an email was sent, and no production-readiness or live-delivery claim follows.
- Rate-limit windows are process-local per direct-peer IP (observed: registry reset across restarts behaved as the contract discloses); concurrency beyond one browser and CLI was out of scope.
- Chromium's informational sandbox warning (allow-scripts + allow-same-origin) reflects the widget's deliberate, documented sandbox combination recorded in the contract; it is not a new finding of this validation.

## Blockers

None. All issue-03 scenarios passed with real-input, observed evidence. ESC-01 (backend) and ESC-02 (widget) behaviors matched their completion records in every observed case, including exact wordings, byte-for-byte replay, cap semantics, isolation 401 identity, restart persistence and cooldown behavior.
