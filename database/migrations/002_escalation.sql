CREATE TABLE IF NOT EXISTS escalation_tickets (
 conversation_id TEXT NOT NULL REFERENCES conversations(id),
 ticket_id TEXT NOT NULL CHECK(length(ticket_id)=32),
 email TEXT NOT NULL CHECK(length(CAST(email AS BLOB)) BETWEEN 6 AND 254),
 question_note TEXT NOT NULL CHECK(length(CAST(question_note AS BLOB)) BETWEEN 0 AND 2000),
 recorded_at TEXT NOT NULL,
 delivery_outcome TEXT NOT NULL CHECK(delivery_outcome IN ('delivery_not_wired')),
 ticket_number INTEGER NOT NULL CHECK(ticket_number BETWEEN 1 AND 5),
 PRIMARY KEY(conversation_id,ticket_id),
 UNIQUE(conversation_id,email,question_note),
 UNIQUE(conversation_id,ticket_number)
);
INSERT OR IGNORE INTO schema_versions(version) VALUES(2);
