CREATE TABLE IF NOT EXISTS ratings (
 conversation_id TEXT PRIMARY KEY REFERENCES conversations(id),
 rating INTEGER NOT NULL CHECK(rating BETWEEN 1 AND 5),
 comment TEXT NOT NULL CHECK(length(CAST(comment AS BLOB)) <= 2000),
 rated_at TEXT NOT NULL
);
INSERT OR IGNORE INTO schema_versions(version) VALUES(3);
