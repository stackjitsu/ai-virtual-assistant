# LLM-01: provider endpoint configuration seam

Status: completed

Owner: backend worker. Context: chat-backend. Pre-read: `AGENTS.md`, `.harness/goal.md`, `.harness/tasks.json` (LLM-01), `.scratch/provider-endpoint-seam/spec.md`, this issue, `internal/chat/provider.go`, `cmd/assistant/main.go`, `docs/contracts/backend.md`. Allowed edits: `internal/chat/**`, `cmd/assistant/main.go`, `docs/contracts/backend.md` (additive note only), `README.md` (additive env documentation), this issue, scratch evidence under `/tmp/opencode/ai-endpoint-seam/**`. No `web/**`, no harness shared state, no academic files, no commits or pushes.

## Task

1. Claim this file (Status: claimed) before any other edit.
2. Add an environment-driven endpoint override:
   - Read `OPENAI_BASE_URL` in `cmd/assistant/main.go`; empty/missing ⇒ construct the provider exactly as today (`chat.NewOpenAI`) — byte-for-byte unchanged behavior
   - Add an exported parameterized constructor in `internal/chat` (for example `NewOpenAIEndpoint(key, model, endpoint string)`) reusing all existing fields, timeout, bounds and decode discipline; the v1 `NewOpenAI` constructor keeps its exact signature and behavior
   - Validate the override: must parse as an absolute URL; require `https://` unless the host is `127.0.0.1` or `localhost`; reject userinfo, fragments, query components; empty after trim ⇒ use default. Failure exits at startup with the same plain-honest style as other environment validation
   - Wire in `main.go` so the exact POSTed URL is the configured value; document in `README.md` that the variable names the complete chat-completions URL (give the Gemini OpenAI-compatible path as an example)
3. Add an additive note in `docs/contracts/backend.md`: runtime configuration surface, default unchanged, HTTPS-or-loopback rule.
4. Tests alongside (ADR 0001): loopback `httptest` server asserting the POST lands on the configured path with `Authorization: Bearer` of the key and the existing body bounds; default-constructor endpoint equivalence test (constant endpoint unchanged); startup validation cases (relative URL, https elsewhere rejected, loopback http allowed, userinfo rejected, trailing garbage). Race-clean; all existing suites green.
5. Run `make format lint static unit contract fuzz-smoke`; report outputs.

## Acceptance

- Unset variable reproduces today's behavior exactly (all existing provider tests untouched and green)
- Override validated, wired, documented; loopback test proves the request hits the configured URL
- No existing route/field/schema behavior changes; no live provider calls; no commits or pushes

Report changed paths, gate outputs, and any blocker with evidence rather than widening scope.

## Completion record (2026-10-20)

### Changed paths (exact)

- `internal/chat/provider.go` — additive only: `NewOpenAIEndpoint(key, model, endpoint string)` (same shared `newOpenAI` helper — 15s client timeout, no-redirect rule, every existing field), `loopbackEndpointHost` helper, exported `ValidateOpenAIEndpoint(raw) (string, error)` (trim → blank fallback; reject any ASCII space/control character inside; absolute URL; `https://` unless host is `127.0.0.1` or `localhost`; no userinfo/fragment/query; otherwise `ErrInvalid`). `NewOpenAI`, `newOpenAI`, `Compose`, `providerEndpoint` constant: byte-for-byte unchanged.
- `internal/chat/provider_test.go` — additive only: `TestContractOpenAIConfiguredEndpoint` (loopback httptest on configured complete path asserts POST on that exact path + bearer + Content-Type + bounded strict request shape + strict composition decode; miss-path returns 404 so success proves the configured URL), `TestOpenAIDefaultEndpointEquivalence` (fixed endpoint constant unchanged; equal endpoint ⇒ equal fields/timeout/no-redirect), `TestValidateOpenAIEndpoint` (25 cases incl. the brief's six), `FuzzValidateOpenAIEndpoint` (new fuzzable boundary; auto-discovered by `scripts/check.py`). No existing test touched.
- `cmd/assistant/main.go` — wiring only: reads `OPENAI_BASE_URL`, fails fast with existing plain style, else swaps in `chat.NewOpenAIEndpoint(...)` for the configured complete URL. The default/blank-branch construction is literally today's expression `chat.NewOpenAI(os.Getenv("OPENAI_API_KEY"), model)`.
- `cmd/assistant/main_test.go` — additive only: `TestContractCLIEndpointOverrideGuard` — real subprocess serve with 5 invalid override values (relative, http-elsewhere, userinfo, fragment, query) exits with the honest message before serving.
- `docs/contracts/backend.md` — one additive bullet after the fixed-endpoint sentence (runtime configuration surface, default unchanged, https-or-loopback rule, no behavior change).
- `README.md` — one additive bullet in "Configuration and normal workflow" (complete chat-completions URL contract, Gemini example, default unchanged, startup validation, message text, live-use caveat).
- This issue + scratch evidence under `/tmp/opencode/ai-endpoint-seam/**` (`HANDOFF.md`, gate transcripts).

Not touched: `web/**`, `.harness/*` (pre-existing primary-session edits only — verified by reviewing `git status`; my edits never touched them), `database/**`, academic files, tooling, model/retry/payload logic, route/store/escalation behavior. No commits or pushes. Zero live provider calls — all provider/CLI tests use loopback httptest servers only; no test contacts api.openai.com or any real host.

### Gate outputs (all on the final tree; `scripts/check.py` strips live-service credentials)

- `make format` → `format: PASS`
- `make lint` → `lint: PASS`
- `make static` → `static: PASS`
- `make unit` → `unit: PASS` (10 check-adapter Python tests; `go test -race ./...`: `cmd/assistant` 3.682s ok, `internal/chat` 4.558s ok; Node widget unit tests 48/48)
- `make contract` → `contract: PASS` (`go test -race -run=^TestContract`: `cmd/assistant` 3.815s ok, `internal/chat` 3.864s ok; Node contract tests 13/13)
- `make fuzz-smoke` → `fuzz-smoke: PASS` — 5 targets × 2s (`-parallel=2`): `FuzzDecodeEscalation` 4,306 execs / corpus 63, `FuzzDecodeProvider` 110 / 88, `FuzzDecodeTurso` 1,592 / 73, `FuzzValidateMessage` 2,610 / 67, and new `FuzzValidateOpenAIEndpoint` 6,139 execs / corpus 30 (baseline 10 seeds)
- Extra race evidence: `go test -race -count=2 ./internal/chat ./cmd/assistant` → ok (8.558s / 6.514s)

### Test evidence before/after

- `internal/chat`: 23 → 26 top-level test functions; 4 → 5 fuzz targets; passing `-v` (sub)cases 122 → 161 (new: 1 configured-endpoint contract test, 1 equivalence test, 1+25 validation cases, fuzz function + 10 seeds)
- `cmd/assistant`: 3 → 4 top-level test functions; passing `-v` (sub)cases 8 → 14 (new CLI guard: function + 5 subtests)
- Node widget tests unchanged (no `web/` edits): 48 unit + 13 contract, all green

### Exact validation behavior and error messages

- Startup failure (`log.Fatal`, serve path only — `migrate`/`seed-demo` do not use the endpoint): `2026/… assistant: invalid OpenAI endpoint configuration` (single fixed message; case is exit status 1 before any listen)
- Validator error value: `ErrInvalid` (the package's existing semantic category; self-describing as `invalid record`)
- Precisely rejected: missing scheme/relative (`example.org/…`, `//host/…`), non-`http(s)` schemes, plain `http` to any non-loopback host (incl. `192.168.1.10`, `0.0.0.0`, `[::1]`), userinfo (`user@`, `user:pass@`), fragments, query components, empty host (`https://`, `https://:8080`), ASCII space/control anywhere (so `"https://example.org/v1/chat/completions trailing"` is rejected, including path-trailing garbage that `url.Parse` alone tolerates)
- Precisely accepted: `https://` to any host; `http://` only when host is `127.0.0.1` or `localhost` (case-insensitive); any port; the value is trimmed and POSTed exactly as configured (complete URL including path)
- Missing, empty, or whitespace-only `OPENAI_BASE_URL` → default `https://api.openai.com/v1/chat/completions` byte-for-byte (same construction expression as today)

### Decisions beyond the brief (explicit)

1. The brief's wording "https elsewhere rejected" is implemented as "plain `http` to non-loopback hosts rejected" — the spec (`HTTPS is required except for loopback hosts`) and the Gemini example require `https://` to other hosts to be accepted; an https-elsewhere ban would contradict the feature's own zero-cost evaluation purpose.
2. Loopback host match is case-insensitive for `localhost` (`strings.EqualFold`), exact for `127.0.0.1`; only these two literal hosts count — `http://[::1]`, `http://0.0.0.0`, private-network IP hosts are rejected, pinned by tests.
3. Validation lives as an exported `chat.ValidateOpenAIEndpoint` beside the constructor in `internal/chat` (seam stays cohesive: constructor + validator + loopback rule in one place) — `main.go` only wires it and `log.Fatal`s with the one fixed honest message. Tests can unit-test the pure function; the CLI guard proves actual startup exit.
4. OpenAI API model-name semantics: no name validation beyond the existing rule (`≤128` chars, no whitespace) — endpoint override changes nothing about the request body.
5. `https://` to the default host (`https://api.openai.com/v1/chat/completions`) is accepted if explicitly configured (it is a valid https absolute URL), though unset behavior remains byte-for-byte unchanged; case scheme (`HTTPS://`) is normalized by `url.Parse` and accepted.
6. Accepted values must also contain no inner ASCII space/control characters (RFC 3986: unencoded space is not a URL character); this is the rule that rejects "trailing garbage" including `url.Parse`'s lenient path handling.
7. A subprocess end-to-end test with a *valid* override was deliberately not added: the loopback contract capture test proves the POST lands on the configured complete URL through the same constructor `main.go` uses; adding a full serve flow would duplicate existing integration scaffolding without added proof, and keeps zero ambiguity about no unintended outbound traffic.
8. `.env.example` was not edited (outside this issue's allowed paths); README documents the variable additively. Suggested follow-up for a docs owner: add `OPENAI_BASE_URL=` to `.env.example`.

### Exact env usage example (Gemini-compatible endpoint)

```
### any OpenAI-compatible endpoint; the value is the COMPLETE chat-completions URL
export OPENAI_API_KEY='<your separately held key>'
export OPENAI_MODEL='<model name your endpoint serves, e.g. its non-OpenAI model id>'
export OPENAI_BASE_URL=https://generativelanguage.googleapis.com/v1beta/openai/chat/completions
go run ./cmd/assistant serve
```

Unset/blank any variable → today's default behavior exactly (`https://api.openai.com/v1/chat/completions`, model `gpt-4.1-mini`). The provider POSTs `Authorization: Bearer <key>`, `Content-Type: application/json` and the existing bounded JSON body to the configured URL byte-for-byte.
