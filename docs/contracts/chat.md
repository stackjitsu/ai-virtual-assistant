# Chat and embedding contract — v1

Status: SETUP-01 baseline for review; not evidence of an implemented feature. Backend and widget owners coordinate changes through the orchestrator. JSON names below are normative; no streaming in this slice.

## HTTP

Same-origin iframe → backend requests under `/api/v1`. JSON UTF-8 only; reject unknown fields, trailing JSON, invalid UTF-8, wrong types and unsupported media types. Bodies at most 16 KiB (creation 2 KiB). No cookies or URL credentials. All API responses use `application/json`, `Cache-Control: no-store`, `X-Content-Type-Options: nosniff`; text is never HTML. IDs are opaque lowercase hex: conversation 16 random bytes (32 chars), capability 32 random bytes (64 chars). Randomness is `crypto/rand`, never a deterministic fallback. UUID or `[A-Za-z0-9_-]{8,64}` request IDs are client-generated; the widget uses `crypto.randomUUID()`.

| Method/path | Request | Success |
|---|---|---|
| POST `/api/v1/conversations` | `{}` or `{"context":{"page_path":"/help","locale":"en"}}` | 201 `{"conversation_id":"…","capability":"…","created_at":"2026-10-20T12:00:00Z"}` |
| GET `/api/v1/conversations/{id}/messages?limit=20&before=41` | `Authorization: Bearer <capability>`; no body | 200 history envelope below |
| POST `/api/v1/conversations/{id}/messages` | bearer; `{"request_id":"…","text":"How do I start?"}` | 200 completed-turn envelope below, including identical replay |

Creation returns capability **once**; only its SHA-256 digest is stored. Creation is not replayable: lost creation response means start a fresh conversation. ID alone is not authorization. Authenticate before history access, replay lookup, retrieval or paid inference. Missing, malformed, unknown-ID and wrong capability all return identical 401 `unauthorized` (no existence oracle). Compare fixed-length digests in constant time. Capability grants access to that conversation, not identity or staff privileges. No sign-in/dashboard/escalation routes.

### Records and bounds

```json
{"messages":[
  {"seq":1,"request_id":"req_12345678","role":"user","text":"How do I start?","created_at":"2026-10-20T12:00:01Z","status":"submitted","sources":[]},
  {"seq":2,"request_id":"req_12345678","role":"assistant","text":"…","created_at":"2026-10-20T12:00:02Z","status":"answered","sources":[{"id":"demo-start","title":"Demo getting started","url":"https://example.org/help/start"}]}
],"next_before":null}
```

- History: `limit` integer 1–100, default 20, `before` positive message sequence (exclusive); reject duplicate/unknown query keys. Select newest eligible `limit` messages, return **ascending** `seq`; `next_before` is the first returned sequence only if older messages exist, otherwise null. No timestamps as cursors. Empty history is `[]`. Pair storage must still honor odd limits/cursors. Sequence starts at 1; a turn has user `2*n-1`, assistant `2*n`. Maximum 50 completed turns/100 messages per conversation; refuse the next new turn with 409 `conversation_full`, while old replays still work.
- POST text: trimmed, nonempty, at most 2,000 Unicode code points and 8,000 UTF-8 bytes; reject NUL/control characters except tab/newline/CR. No HTML interpretation, no command syntax. Provider answer at most 4,000 code points and 16,000 bytes. Timestamps UTC RFC3339, created server-side; ordering relies on sequence, not time.
- `context` is optional and creation-only; no arbitrary nested metadata. Only `page_path` (absolute-path string ≤256 bytes, no query/fragment, `//`, backslash or controls) and `locale` (`[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8}){0,2}`, ≤24 bytes). Neither is trusted evidence, identity or retrieval policy. Reject other fields, full URLs, raw DOM, user/account IDs, credentials and free-form page text. Backend stores bounded context and forwards only these fields. The host deliberately supplies a safe path, not automatic `location.href` collection.
- Sources: max 3, approved retrieved IDs only (`[a-z0-9][a-z0-9_-]{0,63}`), nonempty title ≤160 code points, optional URL ≤2,048 bytes; only curated HTTPS URLs without credentials/controls. Source text and URLs never come from host input/provider inventions. User records always have `sources:[]` and `status:submitted`. Assistant status is `answered` or `needs_clarification`; `answered` requires nonempty validated sources; clarification has `sources:[]`. Render titles/answers with DOM `textContent`; safe HTTPS links get `rel="noopener noreferrer"` and no HTML/Markdown rendering.

Completed POST response: `{"request_id":"…","messages":[<user>,<assistant>]}`. Replay returns exactly that stored envelope (same status, timestamps, source snapshots and sequence), not a new provider call. Fingerprint the canonical trimmed text plus stored context with SHA-256. Same completed request ID + different fingerprint → 409 `idempotency_conflict`. Same request in flight → 409 `conversation_busy` with `Retry-After: 1`; different new request in that conversation also busy. A client retries a transport failure using **the same** ID/text; it must not generate a fresh ID on retry. On an uncertain POST response, reload history before offering a new send. Failed requests without a committed turn can be retried; no failed half-turn enters history. Exactly-once provider billing across crashes is not promised.

### Safe outcomes

Approved relevant material is retrieved before OpenAI; answers are AI-composed from it, not direct FAQ strings pretending to be AI. No relevant material → a completed `needs_clarification` turn explaining the limitation, without a provider call. A model-declared unsupported question also yields this status. Prompt material is untrusted data; never permit tools, URL fetching or instructions from User/knowledge to override system policy. Citation validation reduces risk, not a proof of semantic factuality.

Errors always have shape `{"error":{"code":"provider_unavailable","message":"The assistant is temporarily unavailable. Please retry."}}`. Use fixed safe messages, no raw upstream bodies/SQL/stack traces, transcript or secret values. No persistence of user-only messages on failure.

| HTTP | Codes / behavior |
|---|---|
| 400 | `invalid_request` (including query/context/ID shape) |
| 401 | `unauthorized` as above |
| 403 | `origin_not_allowed` (Origin differs from configured assistant origin) |
| 409 | `conversation_busy`, `idempotency_conflict`, `conversation_full` |
| 413 / 415 | `request_too_large` / `unsupported_media_type` |
| 429 | `rate_limited`, `Retry-After: 60` |
| 503 | `provider_unavailable` (missing key, timeout, upstream failure, malformed/invalid citation output), `storage_unavailable` (including uncertain commit), `service_busy` |
| 404 / 405 / 500 | `not_found` / `method_not_allowed` / `internal_error`, still safe JSON on API paths |

Unavailable services are **not** successful canned AI responses. Missing OpenAI key permits local persistence/UI operation but every question needing inference fails 503 honestly. No provider calls during tests except loopback fixture servers. Retryable failures do not consume a completed turn. If commit acknowledgement is lost, retry first looks up the stored request; never blindly resend a completed inference.

## Embedding / assets

Widget owner owns **all** `web/`, including `web/embed.go` (`package web`, `//go:embed assets`, exported `var Assets embed.FS`), `web/assets/{loader.js,widget.html,widget.js,widget.css,sample-host.html}`, and Node `web/tests/*.test.mjs` tests outside assets. Embed only `assets/`, not tests. Backend imports `ai-virtual-assistant/web`, obtains `fs.Sub(web.Assets,"assets")`, serves `/widget/` with this FS, and redirects `/` to `/widget/sample-host.html`. Backend never edits/copies/generates web assets; no build-time npm step or disk-relative runtime dependency. Go cannot compile this import until widget worker supplies the embed package: that is an explicit coordination dependency, not a setup stub.

Host integration example:

```html
<script src="https://assistant.example/widget/loader.js"
        data-assistant-origin="https://assistant.example"
        data-page-path="/help" data-locale="en" defer></script>
```

- Loader validates configured absolute origin (HTTPS, or loopback HTTP for development), no credentials/path/query/fragment; derive default from its own script URL, not parent URL. It mounts an accessible launcher and one iframe at `<assistant-origin>/widget/widget.html#parent_origin=<encoded location.origin>`. No context/capability in URL. Iframe provides actual chat controls, AI/disclosure label, loading, retry/error, clarification, source and history rendering; screenshots alone are not integration.
- Iframe sandbox: `allow-scripts allow-same-origin`, no top navigation, forms, downloads/popups. Loader sends only `{type:"assistant:init",version:1,context:{…}}` after receiving `{type:"assistant:ready",version:1}`. Iframe accepts init only from `window.parent`, exact validated fragment parent origin, and only once. Loader accepts ready/resize only from its iframe `contentWindow` and exact assistant origin. Every `postMessage` uses a concrete target origin (never `*`); validate shape/version, cap resize height to 300–720 px. No transcript, capability, request text, provider key or response contents ever cross postMessage. Parent origin is an untrusted routing hint, **not authentication**. Bad/missing origin → no handshake/no conversation.
- Widget fetches only its own origin using bearer headers and `credentials:"omit"`. Keep conversation ID/capability in iframe memory only, not host DOM, URL, logs, local/session storage or parent events. Reload loses capability and creates a fresh conversation; disclose this rather than promise account history. Disable parallel sends, preserve request ID/text for retry; capability never returned again via GET.
- Sample host must demonstrate the loader, not directly mount application markup; also test a different-origin host/iframe with exact-origin/source rejection. Same-origin sample is useful but does not prove cross-origin embedding. Backend allows **no API CORS**: host never calls chat directly. API requests carrying an Origin other than configured `ASSISTANT_ORIGIN` are rejected (including `null`); missing Origin is allowed for authorized CLI/tests, not a separate authentication mechanism. Compare parsed exact origins, not suffixes or untrusted forwarded headers. No credentialed wildcard CORS.
- Serve widget CSP `default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors <configured WIDGET_PARENT_ORIGINS>`. Production uses explicit space-separated origins. Local default permits only `http://localhost:8080` and `http://127.0.0.1:8080`; a second-origin browser fixture explicitly configures its port. Do not send X-Frame-Options DENY/SAMEORIGIN on widget.html. Parent origins/allowlists are public anti-misembedding controls, not secrets or User authentication. Backend and loader must agree on optional context validation.

## Resource/threat limits

First slice: one backend process, no horizontal scaling claim. Limit global in-flight inference to 8, one per conversation. Provider timeout 15 s, handler deadline 20 s, storage HTTP timeout 3 s, bounded provider/storage response bodies 256 KiB/8 MiB (storage must fit bounded document/history rows including JSON escaping). No automatic paid-inference retries. Process-local per-direct-peer-IP fixed window: creation 10/minute, message attempts 30/minute; bound limiter registry to 10,000 entries with expiry/eviction. Do not trust X-Forwarded-For without a separately approved proxy policy. Count rejected authenticated attempts too; unauthorized requests must never reach provider. Body caps and deadlines apply before expensive work.

Public widgets can be automated; bearer theft grants history access; an embedding host can replace/overlay its own UI; origin checks cannot stop non-browser abuse, malicious extensions or same-origin compromise. In-memory limits reset on restart and do not provide production DDoS or spending protection. No promise of PII filtering, automatic hallucination prevention, encryption at rest, deletion/export workflows or production readiness. Show User disclosure: AI may be wrong; questions/history go to the configured server and bounded excerpts go to OpenAI; use no sensitive data; the demo knowledge is fictional, not organization policy. No transcript/token logging by default.

## Escalation — slice 2 additive section

This section is **additive**: every route, field name, bound and behavior above is unchanged byte-for-byte. It specifies the guaranteed visitor email-escalation path (no staff interface, no WhatsApp, no auth changes). Real outgoing SMTP transport remains out of scope entirely; the local delivery adapter only records outcomes with zero network I/O. Ticket records append immutably to the conversation record set; no staff actor sees them in this slice.

### `POST /api/v1/conversations/{id}/escalation`

Same machinery as the message POST unless stated otherwise: capability `Authorization: Bearer <capability>` authorization reuses the identical seam and `unauthorized` semantics (missing, malformed, unknown-ID and wrong capability → identical 401 body, no existence oracle, digest-only constant-time comparison, ID alone is not authorization); identical origin rules (no API CORS, exact configured origin, rejected `Origin` values → 403); no query string; JSON UTF-8 only; strict single object with exact field names — unknown fields, duplicate keys (including escaped-equivalent repeats), trailing JSON, invalid UTF-8, JSON `null` leaves and wrong types are rejected as `invalid_request`. Request body at most 4 KiB. Timestamps UTC RFC3339, server-generated. Ticket IDs are opaque 32-char lowercase hex from `crypto/rand`; only the capability digest is stored, never the raw capability.

Request (exact JSON names):

| Field | Rule |
|---|---|
| `email` | required string. Trimmed and lowercased once, then validated in that canonical form, which is the form stored. Bounded: ≤254 bytes total; local part ≤64 bytes (`[a-z0-9]` edges, separators `._%+-` only between alphanumeric pieces); one `@`; domain ≤253 bytes of DNS labels (`[a-z0-9]` edges, `-` inside, each ≤63) followed by a final purely alphabetic TLD of 2–63 letters. No spaces, controls, multibyte runes, userinfo or quoted forms fail validation. |
| `question_note` | optional string. Absent, empty or whitespace-only means no note (stored as empty string). Otherwise trimmed, then bounded to ≤1,000 Unicode code points and ≤2,000 UTF-8 bytes; reject NUL and control characters except tab/newline/CR. Never HTML/Markdown interpreted or composed into anything user-facing unsafely. |
| `consent` | required and must be the literal JSON `true`. Absent, `false`, `null`, `"true"`, `1` or any other value fails closed: nothing is recorded and the response is 400 `invalid_request`. Consent is recorded with the ticket. |

Success: 201 with exactly `{"ticket_id":"…","recorded_at":"2026-10-20T12:00:00Z","delivery_outcome":"delivery_not_wired","message":"Your request has been recorded. Email delivery is not configured, so no email has been sent."}`. The `message` is the one fixed truthful visitor sentence for this slice's outcome.

Delivery honesty: this slice defines exactly one value of `delivery_outcome`: `delivery_not_wired`. It always means the ticket is saved to the conversation's record set and **no email has been sent or will be sent**, because no outgoing transport is configured anywhere in this codebase. A saved ticket with this outcome must never be worded as "email sent", and no response or record claims delivery. If a wired transport ever exists it is a separate contract change; until then the recording adapter performs no network I/O.

Duplicates and appends: an exact repeat of the same canonical (`email`, `question_note`) pair for the same conversation replays the stored 201 envelope byte-for-byte (same `ticket_id`, same `recorded_at`) instead of appending a second ticket — a retry after transport loss is safe with the same body. Any new canonical pair appends a new immutable ticket record. Ticket appending never consumes turn sequences, never creates, edits or deletes messages, and never touches the conversation context.

| HTTP | Codes / behavior |
|---|---|
| 400 | `invalid_request` (consent/email/note/body/ID-shape) |
| 401 | `unauthorized`, identical to the message routes |
| 409 | `conflict` — the per-conversation bound of 5 ticket records is reached; matching replays still return their stored ticket |
| 429 | `rate_limited`, `Retry-After: 60` |
| 503 | `storage_unavailable` |

All other statuses (403/404/405/413/415/500) and the error envelope shape, fixed safe messages and no-store headers are exactly the v1 HTTP behavior. Escalation rate limiting is one more process-local per-direct-peer-IP fixed window: **10 escalation attempts/minute**, counting rejected and unauthorized attempts, sharing the bounded 10,000-entry registry with expiry/eviction as in v1. The 5-tickets-per-conversation bound is enforced atomically in storage (a step-number uniqueness plus range check), so concurrent racing inserts cannot exceed it; at the bound only already-stored identical repeats replay. Tickets persist across process restarts through the additive migration `2` alongside the v1 schema, which is untouched: columns are the conversation reference, ticket id, canonical visitor-provided email (lowercase-trimmed), note, `recorded_at`, the honest `delivery_outcome` and the bounded step number. No raw capabilities, secrets, staff views, provider contents or host-page data are stored or logged; the email is stored only as visitor-provided after bounds, for staff review in a later slice.

## Conversation rating — visitor slice, additive contract

This section adds one route without changing creation, messages, escalation, embedding, CSP, provider, import or disclosure behavior. A rating belongs to the **conversation**, never to a particular answer. The widget may offer it only after a completed turn (assistant status `answered` or `needs_clarification`); server-side storage enforces the same prerequisite. There is no public ratings read/list endpoint, staff access, provider call, email action or new guest identity. A reload still loses the in-memory capability and cannot recover or revise the prior conversation's rating.

### `POST /api/v1/conversations/{id}/rating`

No query string. Require `Authorization: Bearer <capability>` and the exact existing assistant-origin policy: no API CORS, a mismatched or `null` Origin is 403, and absent Origin is allowed for authorized non-browser clients. The conversation ID has the v1 lowercase 32-hex shape. Missing/malformed bearer, unknown conversation and wrong capability all produce the **same** 401 `unauthorized`; the ID alone grants no access. Compare stored capability digests in constant time and authorize **before any rating read, replay decision or write**. All API responses retain JSON/no-store/nosniff and fixed safe error envelopes from v1; neither rating nor comment is placed in an error or log. The rating is not sent to the host via `postMessage`.

Request is exactly `{"rating":5,"comment":"Helpful"}`; `comment` may be omitted. Strict UTF-8 JSON **single object**, at most **4 KiB of request-body bytes**: reject unknown fields, duplicate names (including escaped-equivalent names), trailing data, invalid UTF-8, `null` for either field, wrong types and unsupported media types. `rating` is required and must be a JSON integer literal in **1 through 5** (not `1.0`, `1e0`, a string, a boolean or `null`). `comment` is an optional string: absent, empty or whitespace-only canonicalizes to the empty string. Otherwise trim leading/trailing Unicode whitespace once (no case folding or Unicode normalization); the canonical result must have at most **1,000 Unicode code points AND 2,000 UTF-8 bytes**, with no NUL or Unicode control characters except tab, newline and CR. Count both bounds **after** trimming; validate the decoded canonical value, store that value, and compare that value for replay. Invalid input never changes the stored rating.

Success for first write, identical canonical replay and changed replacement is **200** with exactly `{"rating":5,"comment":"Helpful","rated_at":"2026-10-20T12:00:00Z"}`. `comment` is always present, as `""` when absent; `rated_at` is a server-generated UTC RFC3339 timestamp from the **stored row**, not a client-supplied time. The success response does not say the rating was reviewed, sent, or delivered. There is at most one durable row per conversation: `(rating, canonical comment)` equality returns the existing row, including the **same timestamp**; differing rating and/or canonical comment atomically replaces both fields and assigns a **new, distinct timestamp**. A replacement must not reuse the old timestamp even when writes fall within the same clock tick. Concurrent attempts serialize at storage: each success reflects a whole committed row, never mixed fields; an identical retry following an uncertain commit retrieves the stored result rather than overwriting its timestamp. Distinct racing submissions may commit in either order; the last committed replacement wins. Replays and replacements are allowed even after 50 turns and do not create a turn or ticket.

| HTTP | Codes / behavior |
|---|---|
| 400 | `invalid_request` (invalid ID shape, query, JSON, rating or comment) |
| 401 | `unauthorized` (uniform capability/unknown-ID response) |
| 403 | `origin_not_allowed` |
| 409 | `rating_requires_turn` when no completed turn exists at the atomic write decision; no rating is saved |
| 413 / 415 | `request_too_large` / `unsupported_media_type` |
| 429 | `rate_limited`, `Retry-After: 60` |
| 503 | `storage_unavailable` (including an unresolved uncertain commit) |
| 404 / 405 / 500 | `not_found` / `method_not_allowed` / `internal_error` for generic routing/internal failures, not for an unknown conversation ID |

Rating attempts have their own **10 requests per minute per direct peer IP** process-local fixed window; count rejected and unauthorized attempts as well as replays and replacements. Share the existing bounded 10,000-entry limiter registry with expiry/eviction; do not trust forwarding headers, promise cross-process coordination or claim production abuse protection. Rate limiting and basic request validation may precede authorization, but **no rating lookup or mutation** may precede it. Storage failures must not be represented as successful feedback. There is no arbitrary per-conversation rating-count cap: revisions replace the single row. The widget submits from its iframe with `credentials:"omit"` and its in-memory bearer, uses text-only DOM rendering for any comment, and presents only an honest recorded/updated success or safe failure; it never claims staff review. Existing pre-send privacy/data-sharing and guest-reload disclosures remain visible, and the recorded-not-sent email and disabled WhatsApp outcomes remain unchanged.
