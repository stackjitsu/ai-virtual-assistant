# RATE-03: accessible guest rating controls

Status: ready-for-agent
Blocked by: 02
Owner: chat-widget worker. Allowed paths: `web/**`, this issue. Depends on RATE-01 contract (issue 02, not backend implementation); coordinate parallel backend work through published contract.

Show 1–5 accessible choices and optional comment only after at least one completed turn; let guest submit and revise, render waiting/success/error honestly, keep private capability in iframe memory, validate responses and avoid duplicate clicks. Preserve chat history/retry, handoff consent, disclosures, CSP, sandbox, host protocol and mobile pinned composer. Add Node unit/contract and real-browser interaction checks. No provider/mail/staff calls. Call tdd skill where useful. Checks: focused Node tests, `make format lint static unit contract`, browser evidence, `git diff --check`.
