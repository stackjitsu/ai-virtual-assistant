# Conversation rating — visitor slice

Status: approved scope for implementation on feat/conversation-rating

## Goal

After a completed chat turn, a guest may rate the conversation 1–5 and optionally leave a comment. The guest can revise the rating during the same accessible session. Ratings are stored for a future staff dashboard; this slice does not claim staff can view them. Verify the existing pre-send privacy/data-sharing and guest-reload disclosures; leave the recorded-not-sent email outcome and disabled WhatsApp unchanged.

## Decisions and invariants

- Rating belongs to a conversation, not an individual answer. At least one completed turn (answered or needs_clarification) is required. One durable row per conversation; an identical canonical repeat returns the existing result (same timestamp), a changed submission replaces it atomically (new timestamp).
- Input: integer rating 1–5, optional trimmed comment up to 1,000 Unicode code points AND 2,000 UTF-8 bytes; empty means no comment; no NUL or controls except tab/newline/CR. JSON strict; 4 KiB body. Response repeats validated rating, canonical comment and server timestamp; no invented review/delivery claims.
- Capability authorization and exact origin rules mirror chat messages; no guest identity, cross-conversation access, new browser storage, staff access, public ratings list, provider call or live remote/service access.
- Additive versioned schema/contract; existing conversation, turn, escalation, sandbox, CSP, provider and import behavior remain unchanged. Auth precedes rating lookups/writes. Storage changes must be atomic and race-safe in SQLite and Turso.
- Keep tests credential-free and local; no deployment or production-readiness claim.

## Task graph

DISC-01 and RATE-01 are independent; RATE-02 and RATE-03 require RATE-01; RATE-04 requires RATE-02 and RATE-03. Each implementer works in its own branch/worktree; merged to this integration branch and reviewed before resolving. See numbered issues.
