# WIDGET-01: widget

Status: implementation ready for integration review (claimed by WIDGET-01 widget worker, 2026-10-20); full-project gates/browser evidence pending

Owner: widget worker. Context: chat-widget.

The reviewed v1 baseline is docs/contracts/chat.md. Task fields and validation expectations: .harness/tasks.json WIDGET-01 and .scratch/ai-chat/spec.md. Own all web/ including embedded asset package and Node tests. Claim before edits; ask orchestrator for shared-interface changes. Backend composes web.Assets. Do not alter cmd/, internal/, dependencies or academic reports. Report actual Node/browser evidence separately from mocked fixtures and live inference.

Implementation brief: deliver the framework-neutral loader, iframe chat and sample host under web/**, consuming the unchanged v1 HTTP/embedding contract. Capability stays in iframe memory, all rendering is text-only, origins/schemas are validated before fetch, and uncertain submissions require history reconciliation before same-ID retry. Acceptance includes accessible controls/disclosures, honest unavailable/clarification outcomes, bounded requests, safe curated sources, and Node behavioral plus contract tests. Required checks: format/lint/static/unit/contract; run widget-only checks first while backend integration is pending. No live APIs, credentials, dependency edits or backend edits.

Evidence (Node v26.1.0 / Go go1.26.3):
- Actual vanilla loader + iframe + sample host and exported embedded web.Assets implemented without npm, frameworks or backend changes.
- Node behavioral checks: 24 ordinary tests; 12 contract tests; 36 total passing. Includes 128 deterministic malformed-input property cases; these are fixtures/property checks, not native fuzz or real-browser evidence.
- go test -race ./web: PASS (one asset-packaging contract test); go vet ./web: PASS; owned web formatting and seven JS/module syntax checks: PASS. make format: PASS at integration attempt.
- Whole-repository make lint/static/unit/contract/fuzz-smoke each exceeded a 120-second attempt during backend integration. These are unverified, not passes. The unit attempt did pass 10 check-adapter tests before timing out. Owner for final integrated gates: orchestrator/backend and validation owners.
- No live backend/provider/storage calls, credentials, commits or pushes. No contract, dependency, backend or .harness edits.

Construction/public controls, local commands and browser run hints are in web/README.md. The loader allows only explicit origin/page-path/locale attributes; no host API or arbitrary context/DOM forwarding. All API calls originate in iframe memory with omitted credentials and bounded deadlines/body reads. Ambiguous POSTs reconcile authenticated history before explicit same-ID retry. Missing provider service is visibly unavailable, never fake AI success. Sandbox forbids popups; source links use same-frame navigation with a copy-link/access-loss notice.

Remaining dependency: independent integrated Chromium proof, including a truly different-origin host, CSP/module loading, keyboard/focus and source navigation. No real-browser proof is claimed by this worker. Full-project gates must be rerun after backend readiness; do not mark the task fully completed solely from widget fixture checks.
