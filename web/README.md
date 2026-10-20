# Build-free chat widget

`web.Assets` embeds only `assets/`. The Go server serves its `assets` sub-filesystem at `/widget/`; Node tests are never public assets. No npm install, bundler or framework is used.

## Host integration

```html
<script src="https://assistant.example/widget/loader.js"
        data-assistant-origin="https://assistant.example"
        data-page-path="/help" data-locale="en" defer></script>
```

The assistant origin defaults to the script's own origin. Explicit origins must be canonical HTTPS origins (or loopback HTTP for development), with no path, trailing slash, credentials, query or fragment. Configure the backend's `WIDGET_PARENT_ORIGINS` to allow the host origin. No API CORS is needed: only the iframe calls the API.

If the host has its own CSP, allow the assistant origin for this external script, stylesheet and iframe. The widget itself uses only its local external script/module/style files under the backend's restrictive widget CSP.

`data-page-path` and `data-locale` are optional creation-only context. The loader does not scrape the page or automatically collect its URL. A path must be absolute, at most 256 UTF-8 bytes, and contain no query, fragment, doubled slash, backslash or controls. Locale uses the contract's bounded language-tag syntax. Other attributes are never copied to the frame or API.

The launcher lazily mounts one sandboxed iframe and toggles it without creating another conversation. The iframe owns the labelled message form, Send button, disclosure, conversation, source links and explicit retry/history controls. Conversation capability, question IDs and transcript stay in iframe memory; only versioned init/ready/resize messages cross the frame boundary. The backend still persists completed turns: reload loses guest access, not stored data.

Click **Send question**, or Tab to it and press Enter/Space. In the textarea, **Ctrl+Enter** or **⌘+Enter** sends; plain Enter adds a newline. These actions share validation and the send lock without native form submission, which the intentionally forms-disabled sandbox blocks before a submit listener can run. Sandbox and CSP permissions remain restricted.

The required sandbox does not allow popups or top navigation. Source links therefore use ordinary same-frame navigation, with a visible warning to copy the link address instead if the User wants to retain guest chat access. A source site can itself refuse framing; the widget does not bypass that site's policy.

## Email escalation (slice 2)

Once a conversation exists, an explicit **Request human support by email** affordance appears below the chat form; it is never opened by a provider failure or any other server event — the visitor must click it. It toggles a compact panel with:

- **Your email** — required; the typed value is validated client-side against the server's canonical rules (trim once, lowercase once, at most 254 bytes, simple local/domain shape). The server stays authoritative; a mirror mismatch can only fail closed before any request is sent.
- **Optional note** — bounded to 1,000 characters (2,000 bytes) after trim; NUL and control characters other than tab, newline and carriage return are rejected locally. The typed value is sent; the server stores its canonical trimmed form.
- A standalone **consent checkbox** with its own wording: "I agree to store this request and note for human support by email. No email is sent while delivery is not configured." Consent fails closed: the checkbox must be literally checked (not merely truthy) before anything is sent. No pre-checked or bundled consent exists.

Sending is an explicit **Send request** click that POSTs exactly `{email, question_note?, consent:true}` to `POST /api/v1/conversations/{id}/escalation` with the same iframe-owned bearer capability and fetch discipline as messages (same-origin, `credentials:"omit"`, `no-store`, bounded aborts). Nothing else leaves the iframe, and the escalation flow never disables or mutates the normal send path, history or disclosures — chat and escalation requests may run concurrently; pagehide aborts both.

Response handling:

- **201** shows the server's truthful outcome sentence verbatim ("Your request has been recorded. Email delivery is not configured, so no email has been sent.") plus `recorded_at` in a visitor-sane UTC form and the opaque ticket reference, all as inert text. A recorded request is never worded as an email that was sent; the envelope is validated against the one contract outcome (`delivery_not_wired`) and anything else fails closed.
- **Duplicate replay is a feature**: resubmitting the same canonical email/note pair returns the stored ticket and displays it again with a "previously stored request, replayed unchanged" note, not an error.
- **400** uses local safe validation wording; raw server messages are never rendered, and no automatic retry follows.
- **409** explains the per-conversation limit of 5 requests and that a matching repeat returns the stored ticket.
- **429** honors the bounded `Retry-After` parsing on an escalation-only cooldown so the chat send path stays untouched.
- **401** (and 403) follow the existing terminal handling: the whole widget reports lost access and reload is required.
- **503** and transport failures show an explicit "use Send request to try again" wording; there are no automatic retries, and duplicate replay protects an actual repeat.

## Failure behavior

- Invalid origin/context/handshake: visible configuration error, no API fetch.
- Missing AI key/provider failure: unavailable, never a canned successful AI answer.
- Ambiguous send: authenticated GET history before any same-ID retry; no automatic inference retry and no new sends while unresolved.
- Busy/rate-limited: bounded retry cooldown, explicit click required. Retries preserve the original trimmed text and cryptographic UUID.
- Invalid response/source/media type, oversized JSON or unsafe URL: fail closed before answer rendering. Every server string is data, never HTML/Markdown; errors use local fixed wording.
- Requests use same-origin mode, omitted credentials, no-store, rejected redirects and a 25-second abort deadline. The handshake has a 10-second deadline.

## Checks from repository root

```sh
node --test web/tests/widget.test.mjs
node --test web/tests/escalation.test.mjs
node --test web/tests/embedding.contract.test.mjs
node --check web/assets/loader.js
node --check web/assets/widget.js
node --check web/assets/widget-core.mjs
node --check web/assets/protocol.mjs
gofmt -l web/embed.go web/embed_test.go
go test -race ./web
make format
make lint
make static
make unit
make contract
```

The Node suites execute real loader/controller/validator logic in small DOM/message/fetch fixtures. They are not real-browser, live inference, CSP or live storage evidence. The deterministic malformed-input property corpus is not native fuzzing. Whole-repository Make checks also require the parallel backend implementation.

Opt-in actual-input regression (Node with built-in WebSocket and Chromium installed; no npm dependencies):

```sh
go build -o /tmp/opencode/ai-chat-send-fix/assistant ./cmd/assistant
node web/tests/sandbox-send.browser.mjs /tmp/opencode/ai-chat-send-fix/assistant /tmp/opencode/ai-chat-send-fix/browser-run
```

Use a fresh evidence directory for each run; `CHROMIUM` can select the Chromium executable. The script starts and cleans up its own loopback Go/SQLite, different-origin host and sandboxed headless Chromium. It uses real CDP mouse/keyboard input (never a synthetic submit) to check one POST per Send click, button Enter/Space, and textarea Ctrl/Meta+Enter; newline behavior, unavailable/history/same-ID retry and persisted clarification are also checked. Credentials are explicitly empty, external browser traffic is blocked via a closed proxy, and results/screenshots omit capabilities. This separate script is not part of every Node unit run. It does not yet drive the escalation panel; real-input escalation evidence is delegated to the independent escalation validation task.

After backend integration, use `make run`, open `http://127.0.0.1:8080/widget/sample-host.html`, and open the launcher. No provider key is required to inspect unavailable/clarification behavior. Do not use real sensitive questions. A **separate-origin** browser test should serve a small host on another loopback port with the script above pointing to `http://127.0.0.1:8080` and explicitly add that host origin to `WIDGET_PARENT_ORIGINS`. Verify actual module/CSP loading, keyboard/focus behavior, spoof rejection and private iframe API requests with Chromium before claiming browser integration. No such browser proof is implied by Node checks.
