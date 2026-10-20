package chat

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ai-virtual-assistant/database"
	_ "modernc.org/sqlite"
)

// SQLStore shares parameterized single-statement operations between SQLite and
// Turso. There is no transaction spanning a model call or multi-statement batch.
type SQLStore struct {
	db     *sql.DB
	remote *tursoHTTP
}
type sqlResult struct {
	columns  []string
	rows     [][]any
	affected int64
}

func OpenSQLite(path string) (*SQLStore, func() error, error) {
	if path == "" || path == ":memory:" {
		return nil, nil, ErrInvalid
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, nil, ErrUnavailable
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, nil, ErrUnavailable
	}
	_ = f.Close()
	if err = os.Chmod(path, 0600); err != nil {
		return nil, nil, ErrUnavailable
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, nil, ErrUnavailable
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	s := &SQLStore{db: db}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for _, statement := range []string{"PRAGMA foreign_keys = ON", "PRAGMA busy_timeout = 2000"} {
		if _, err = s.query(ctx, statement); err != nil {
			_ = db.Close()
			return nil, nil, ErrUnavailable
		}
	}
	return s, db.Close, nil
}

// Migrate is explicit: main invokes it on local start, or on the migrate command.
// Remote construction/start never applies schema changes.
func (s *SQLStore) Migrate(ctx context.Context) error {
	files, err := database.Migrations.ReadDir("migrations")
	if err != nil {
		return ErrUnavailable
	}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}
		b, err := database.Migrations.ReadFile("migrations/" + file.Name())
		if err != nil {
			return ErrUnavailable
		}
		// These repository-owned migrations contain no SQL strings with semicolons.
		// Every schema statement is independently atomic and restart-idempotent.
		for _, statement := range strings.Split(string(b), ";") {
			if strings.TrimSpace(statement) == "" {
				continue
			}
			if _, err = s.query(ctx, statement); err != nil {
				return ErrUnavailable
			}
		}
	}
	return nil
}

func (s *SQLStore) query(ctx context.Context, statement string, args ...any) (sqlResult, error) {
	if s.remote != nil {
		return s.remote.query(ctx, statement, args...)
	}
	// QueryContext supports both DDL and INSERT ... RETURNING with modernc.
	rows, err := s.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return sqlResult{}, ErrUnavailable
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return sqlResult{}, ErrUnavailable
	}
	out := sqlResult{columns: cols, rows: [][]any{}}
	for rows.Next() {
		if len(out.rows) >= 101 {
			return sqlResult{}, ErrUnavailable
		}
		values := make([]any, len(cols))
		dest := make([]any, len(cols))
		for i := range values {
			dest[i] = &values[i]
		}
		if err = rows.Scan(dest...); err != nil {
			return sqlResult{}, ErrUnavailable
		}
		out.rows = append(out.rows, values)
	}
	if rows.Err() != nil {
		return sqlResult{}, ErrUnavailable
	}
	return out, nil
}

func (s *SQLStore) CreateConversation(ctx context.Context, c Conversation) error {
	if !idPattern.MatchString(c.ID) || ValidateContext(c.Context) != nil || c.CreatedAt.IsZero() {
		return ErrInvalid
	}
	b, _ := json.Marshal(c.Context)
	_, err := s.query(ctx, `INSERT INTO conversations(id,capability_hash,context_json,created_at) VALUES(?,?,?,?)`, c.ID, c.CapabilityHash[:], string(b), timestamp(c.CreatedAt))
	return err
}

func timestamp(t time.Time) string  { return t.UTC().Format(time.RFC3339Nano) }
func textCell(v any) (string, bool) { s, ok := v.(string); return s, ok }
func timeCell(v any) (time.Time, bool) {
	s, ok := textCell(v)
	if !ok {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	return t, err == nil && !t.IsZero()
}
func digestCell(v any) ([32]byte, bool) {
	b, ok := v.([]byte)
	var d [32]byte
	if !ok || len(b) != 32 {
		return d, false
	}
	copy(d[:], b)
	return d, true
}
func (s *SQLStore) Conversation(ctx context.Context, id string) (Conversation, error) {
	r, err := s.query(ctx, `SELECT id,capability_hash,context_json,created_at FROM conversations WHERE id=?`, id)
	if err != nil {
		return Conversation{}, err
	}
	if len(r.rows) == 0 {
		return Conversation{}, ErrNotFound
	}
	if len(r.rows) != 1 || len(r.rows[0]) != 4 {
		return Conversation{}, ErrUnavailable
	}
	v := r.rows[0]
	cid, a := textCell(v[0])
	h, b := digestCell(v[1])
	raw, c := textCell(v[2])
	when, d := timeCell(v[3])
	out := Conversation{ID: cid, CapabilityHash: h, CreatedAt: when}
	if !a || !b || !c || !d || !idPattern.MatchString(cid) || strictJSON([]byte(raw), &out.Context) != nil || ValidateContext(out.Context) != nil {
		return Conversation{}, ErrUnavailable
	}
	return out, nil
}

const turnColumns = `conversation_id,request_id,fingerprint,number,user_text,user_time,assistant_text,assistant_time,status,sources_json`

func decodeTurn(v []any) (Turn, error) {
	if len(v) != 10 {
		return Turn{}, ErrUnavailable
	}
	id, a := textCell(v[0])
	req, b := textCell(v[1])
	fp, c := digestCell(v[2])
	n, d := v[3].(int64)
	u, e := textCell(v[4])
	ut, f := timeCell(v[5])
	answer, g := textCell(v[6])
	at, h := timeCell(v[7])
	status, i := textCell(v[8])
	raw, j := textCell(v[9])
	if !a || !b || !c || !d || !e || !f || !g || !h || !i || !j {
		return Turn{}, ErrUnavailable
	}
	sources := []Source{}
	if strictJSON([]byte(raw), &sources) != nil || sources == nil {
		return Turn{}, ErrUnavailable
	}
	t := Turn{ConversationID: id, RequestID: req, Fingerprint: fp, Number: n,
		User:      Message{2*n - 1, req, "user", u, ut, "submitted", []Source{}},
		Assistant: Message{2 * n, req, "assistant", answer, at, status, sources}}
	if validateTurn(t, n-1) != nil {
		return Turn{}, ErrUnavailable
	}
	return t, nil
}

func (s *SQLStore) FindTurn(ctx context.Context, id, req string) (Turn, error) {
	r, err := s.query(ctx, `SELECT `+turnColumns+` FROM turns WHERE conversation_id=? AND request_id=?`, id, req)
	if err != nil {
		return Turn{}, err
	}
	if len(r.rows) == 0 {
		return Turn{}, ErrNotFound
	}
	if len(r.rows) != 1 {
		return Turn{}, ErrUnavailable
	}
	return decodeTurn(r.rows[0])
}
func (s *SQLStore) turns(ctx context.Context, id string) ([]Turn, error) {
	r, err := s.query(ctx, `SELECT `+turnColumns+` FROM turns WHERE conversation_id=? ORDER BY number ASC LIMIT 51`, id)
	if err != nil || len(r.rows) > 50 {
		return nil, ErrUnavailable
	}
	out := []Turn{}
	for index, row := range r.rows {
		t, err := decodeTurn(row)
		if err != nil || t.Number != int64(index+1) {
			return nil, ErrUnavailable
		}
		out = append(out, t)
	}
	return out, nil
}
func (s *SQLStore) History(ctx context.Context, id string, limit int, before *int64) (HistoryPage, error) {
	if limit < 1 || limit > 100 || (before != nil && *before <= 0) {
		return HistoryPage{}, ErrInvalid
	}
	t, err := s.turns(ctx, id)
	if err != nil {
		return HistoryPage{}, err
	}
	return historyPage(t, limit, before)
}

const insertTurnSQL = `INSERT INTO turns (` + turnColumns + `)
 SELECT ?,?,?,?,?,?,?,?,?,? WHERE ?=(SELECT COALESCE(MAX(number),0) FROM turns WHERE conversation_id=?)
 AND ?<50 RETURNING ` + turnColumns

func (s *SQLStore) CompleteTurn(ctx context.Context, t Turn, previous int64) (Turn, error) {
	if err := validateTurn(t, previous); err != nil {
		return Turn{}, err
	}
	// The fingerprint is not trusted to represent different text/context at the store boundary.
	c, err := s.Conversation(ctx, t.ConversationID)
	if err != nil {
		return Turn{}, err
	}
	if Fingerprint(t.User.Text, c.Context) != t.Fingerprint {
		return Turn{}, ErrInvalid
	}
	b, _ := json.Marshal(t.Assistant.Sources)
	r, writeErr := s.query(ctx, insertTurnSQL, t.ConversationID, t.RequestID, t.Fingerprint[:], t.Number, t.User.Text, timestamp(t.User.CreatedAt), t.Assistant.Text, timestamp(t.Assistant.CreatedAt), t.Assistant.Status, string(b), previous, t.ConversationID, previous)
	if writeErr == nil && len(r.rows) == 1 {
		stored, err := decodeTurn(r.rows[0])
		if err == nil {
			return stored, nil
		}
	}
	// Even an HTTP error may mean the atomic insert committed. Read before deciding.
	stored, err := s.FindTurn(ctx, t.ConversationID, t.RequestID)
	if err == nil {
		if stored.Fingerprint == t.Fingerprint {
			return stored, nil
		}
		return Turn{}, ErrIdempotencyConflict
	}
	if !errors.Is(err, ErrNotFound) || writeErr != nil {
		return Turn{}, ErrUnavailable
	}
	turns, err := s.turns(ctx, t.ConversationID)
	if err != nil {
		return Turn{}, ErrUnavailable
	}
	if len(turns) >= 50 {
		return Turn{}, ErrFull
	}
	return Turn{}, ErrConflict
}

func (s *SQLStore) ApprovedKnowledge(ctx context.Context) ([]Knowledge, error) {
	r, err := s.query(ctx, `SELECT id,title,url,text,approved FROM knowledge WHERE approved=1 ORDER BY id LIMIT 101`)
	if err != nil || len(r.rows) > 100 {
		return nil, ErrUnavailable
	}
	docs := []Knowledge{}
	for _, row := range r.rows {
		if len(row) != 5 {
			return nil, ErrUnavailable
		}
		id, a := textCell(row[0])
		title, b := textCell(row[1])
		u, c := textCell(row[2])
		text, d := textCell(row[3])
		approved, e := row[4].(int64)
		if !a || !b || !c || !d || !e || approved != 1 {
			return nil, ErrUnavailable
		}
		docs = append(docs, Knowledge{id, title, u, text, true})
	}
	if validateKnowledge(docs) != nil {
		return nil, ErrUnavailable
	}
	return docs, nil
}

const escalationColumns = `conversation_id,ticket_id,email,question_note,recorded_at,delivery_outcome,ticket_number`

// The guarded insert mirrors the completed-turn CAS: one atomic statement
// claims the next step number, and UNIQUE(conversation_id,ticket_number)
// plus the schema CHECK make multiple converged attempts safe.
const insertEscalationSQL = `INSERT INTO escalation_tickets (` + escalationColumns + `)
 SELECT ?,?,?,?,?,?,(?+1) WHERE ?=(SELECT COUNT(*) FROM escalation_tickets WHERE conversation_id=?)
 RETURNING ` + escalationColumns

func decodeEscalation(v []any) (Escalation, error) {
	if len(v) != 7 {
		return Escalation{}, ErrUnavailable
	}
	cid, a := textCell(v[0])
	tid, b := textCell(v[1])
	email, c := textCell(v[2])
	note, d := textCell(v[3])
	when, e := timeCell(v[4])
	outcome, f := textCell(v[5])
	n, g := v[6].(int64)
	if !a || !b || !c || !d || !e || !f || !g {
		return Escalation{}, ErrUnavailable
	}
	t := Escalation{cid, tid, email, note, when, outcome}
	if validateEscalation(t) != nil || n < 1 || n > maxEscalationsPerConversation {
		return Escalation{}, ErrUnavailable
	}
	return t, nil
}

func (s *SQLStore) FindEscalation(ctx context.Context, conversationID, email, note string) (Escalation, error) {
	if !idPattern.MatchString(conversationID) {
		return Escalation{}, ErrInvalid
	}
	// Bounded lookup: an inconsistent record set can never stream unbounded rows.
	r, err := s.query(ctx, `SELECT `+escalationColumns+` FROM escalation_tickets WHERE conversation_id=? AND email=? AND question_note=? LIMIT 2`, conversationID, email, note)
	if err != nil {
		return Escalation{}, err
	}
	if len(r.rows) == 0 {
		return Escalation{}, ErrNotFound
	}
	if len(r.rows) != 1 || len(r.rows[0]) != 7 {
		return Escalation{}, ErrUnavailable
	}
	return decodeEscalation(r.rows[0])
}

func (s *SQLStore) EscalationCount(ctx context.Context, conversationID string) (int, error) {
	if !idPattern.MatchString(conversationID) {
		return 0, ErrInvalid
	}
	r, err := s.query(ctx, `SELECT COUNT(*) FROM escalation_tickets WHERE conversation_id=?`, conversationID)
	if err != nil {
		return 0, err
	}
	if len(r.rows) != 1 || len(r.rows[0]) != 1 {
		return 0, ErrUnavailable
	}
	n, ok := r.rows[0][0].(int64)
	if !ok || n < 0 || n > maxEscalationsPerConversation {
		return 0, ErrUnavailable
	}
	return int(n), nil
}

// CreateEscalation appends an immutable ticket under the per-conversation cap.
// For every attempt a stored identical repeat wins first (even at the cap),
// so a lost or raced write is safe to retry: reads, never error strings,
// decide the outcome.
func (s *SQLStore) CreateEscalation(ctx context.Context, t Escalation) (Escalation, error) {
	if validateEscalation(t) != nil {
		return Escalation{}, ErrInvalid
	}
	if _, err := s.Conversation(ctx, t.ConversationID); err != nil {
		return Escalation{}, err
	}
	for attempt := 0; attempt < 8; attempt++ {
		existing, dedupErr := s.FindEscalation(ctx, t.ConversationID, t.Email, t.QuestionNote)
		if dedupErr == nil {
			return existing, nil
		}
		if !errors.Is(dedupErr, ErrNotFound) {
			return Escalation{}, dedupErr
		}
		n, err := s.EscalationCount(ctx, t.ConversationID)
		if err != nil {
			return Escalation{}, err
		}
		if n >= maxEscalationsPerConversation {
			return Escalation{}, ErrConflict
		}
		r, writeErr := s.query(ctx, insertEscalationSQL,
			t.ConversationID, t.TicketID, t.Email, t.QuestionNote, timestamp(t.RecordedAt), t.DeliveryOutcome, n, n, t.ConversationID)
		if writeErr == nil && len(r.rows) == 1 {
			stored, err := decodeEscalation(r.rows[0])
			if err != nil {
				return Escalation{}, err
			}
			return stored, nil
		}
		// A failed or empty write may have committed a concurrent duplicate;
		// the next attempt reads the dedup key back before counting it lost.
		// A real storage failure returns from the bounded loop honestly.
		if writeErr != nil && attempt == 7 {
			return Escalation{}, writeErr
		}
	}
	return Escalation{}, ErrUnavailable
}

// SeedDemo is deliberately local-only and explicit; these are fictional policies.
func (s *SQLStore) SeedDemo(ctx context.Context) error {
	if s.remote != nil {
		return ErrInvalid
	}
	docs := []Knowledge{
		{"demo-start", "Sample organization: getting started", "https://example.org/help/start", "Fictional sample organization onboarding: to start the demo, open the sample support widget and ask an onboarding question. This demonstration is not real organization policy.", true},
		{"demo-support", "Sample organization: support hours", "https://example.org/help/support", "Fictional sample organization support hours: this demo describes a sample support team available Monday through Friday. These hours are fictional, not real organization policy.", true},
	}
	if validateKnowledge(docs) != nil {
		return ErrInvalid
	}
	for _, d := range docs {
		if _, err := s.query(ctx, `INSERT INTO knowledge(id,title,url,text,approved) VALUES(?,?,?,?,1) ON CONFLICT(id) DO UPDATE SET title=excluded.title,url=excluded.url,text=excluded.text,approved=1`, d.ID, d.Title, d.URL, d.Text); err != nil {
			return err
		}
	}
	return nil
}
