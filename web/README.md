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

Use a fresh evidence directory for each run; `CHROMIUM` can select the Chromium executable. The script starts and cleans up its own loopback Go/SQLite, different-origin host and sandboxed headless Chromium. It uses real CDP mouse/keyboard input (never a synthetic submit) to check one POST per Send click, button Enter/Space, and textarea Ctrl/Meta+Enter; newline behavior, unavailable/history/same-ID retry and persisted clarification are also checked. Credentials are explicitly empty, external browser traffic is blocked via a closed proxy, and results/screenshots omit capabilities. This separate script is not part of every Node unit run.

After backend integration, use `make run`, open `http://127.0.0.1:8080/widget/sample-host.html`, and open the launcher. No provider key is required to inspect unavailable/clarification behavior. Do not use real sensitive questions. A **separate-origin** browser test should serve a small host on another loopback port with the script above pointing to `http://127.0.0.1:8080` and explicitly add that host origin to `WIDGET_PARENT_ORIGINS`. Verify actual module/CSP loading, keyboard/focus behavior, spoof rejection and private iframe API requests with Chromium before claiming browser integration. No such browser proof is implied by Node checks.
