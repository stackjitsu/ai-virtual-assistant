# Feature: provider endpoint configuration seam

## Scope

One small additive backend configuration change so the approved-knowledge path can be evaluated against any OpenAI-compatible chat-completions endpoint, in preparation for a zero-cost live verification with a customer-held free-tier credential (e.g. an OpenAI-compatible Gemini endpoint).

## Decisions carried from supervision

- The default endpoint remains `https://api.openai.com/v1/chat/completions` byte-for-byte; an unset variable changes absolutely nothing
- The configuration variable names the complete chat-completions URL (the provider POSTs exactly there), because compatible hosts differ in both host and path (`/v1beta/openai/chat/completions`, `/openai/v1/chat/completions`)
- HTTPS is required except for loopback hosts (`127.0.0.1`, `localhost`), which local tests and local evaluation use
- Malformed or non-https-non-loopback values fail fast at startup in the same honest style as other environment validation
- No model-name, retry, or payload changes; provider behavior, timeouts, bounds, and strict decode discipline are untouched

## Non-goals

- No live provider calls in tests (loopback httptest only), no credential handling changes, no widget or storage changes, no deployment work
