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
