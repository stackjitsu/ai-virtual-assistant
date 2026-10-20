# Feature: visitor email escalation

## Scope

Second feature slice after the approved-knowledge chat. Adds the visitor-visible path from an unresolved chat conversation to a human-support request by email, without any staff interface.

## Product decisions carried from supervision

- Email escalation is a guaranteed live path of the product, distinct from the conditional/unconfigured WhatsApp channel
- The visitor supplies the contact email; the assistant never asks for credentials, payment or identity documents
- Consent must be explicit before an escalation request is stored or composed
- The Chat remains chat-context; an escalation never mutates or deletes conversation messages
- Honesty rules apply: if no outgoing-email delivery is configured at runtime, the escalation still records the request as saved with an explicit, truthful delivery outcome rather than pretending an email was sent
- Staff review, ticket queues and dashboards remain later slices; no staff actor is trained or exposed here

## Non-goals

- No WhatsApp, no SMS, no staff dashboard, no auth changes
- No live outbound email in tests; a local adapter records outcomes
- No live OpenAI/Turso calls; all tests stay loopback/local

## Data invariants

- Escalation is capability-authorized like messages (bearer capability only, hash stored server-side)
- Email is stored only as provided by the visitor, bounded, validated, lowercase-trimmed
- Consent is an explicit true boolean recorded with the ticket; absent/unknown consent fails closed
- Ticket records append immutably to the conversation record set; no edits or deletions
- No raw capabilities, no secrets, no host-page content beyond the already disclosed minimal context
