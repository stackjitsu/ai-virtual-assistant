CREATE TABLE IF NOT EXISTS schema_versions (version INTEGER PRIMARY KEY);
CREATE TABLE IF NOT EXISTS conversations (
 id TEXT PRIMARY KEY CHECK(length(id)=32),
 capability_hash BLOB NOT NULL CHECK(length(capability_hash)=32),
 context_json TEXT NOT NULL,
 created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS turns (
 conversation_id TEXT NOT NULL REFERENCES conversations(id),
 request_id TEXT NOT NULL,
 fingerprint BLOB NOT NULL CHECK(length(fingerprint)=32),
 number INTEGER NOT NULL CHECK(number BETWEEN 1 AND 50),
 user_text TEXT NOT NULL CHECK(length(CAST(user_text AS BLOB)) BETWEEN 1 AND 8000),
 user_time TEXT NOT NULL,
 assistant_text TEXT NOT NULL CHECK(length(CAST(assistant_text AS BLOB)) BETWEEN 1 AND 16000),
 assistant_time TEXT NOT NULL,
 status TEXT NOT NULL CHECK(status IN ('answered','needs_clarification')),
 sources_json TEXT NOT NULL,
 PRIMARY KEY(conversation_id,request_id),
 UNIQUE(conversation_id,number)
);
CREATE TABLE IF NOT EXISTS knowledge (
 id TEXT PRIMARY KEY,
 title TEXT NOT NULL,
 url TEXT NOT NULL,
 text TEXT NOT NULL CHECK(length(CAST(text AS BLOB)) BETWEEN 1 AND 8000),
 approved INTEGER NOT NULL CHECK(approved IN (0,1))
);
INSERT OR IGNORE INTO schema_versions(version) VALUES(1);
