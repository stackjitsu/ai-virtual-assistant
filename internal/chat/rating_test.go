package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"ai-virtual-assistant/database"
)

type ratingSpyStore struct {
	Store
	writes atomic.Int64
}

func (s *ratingSpyStore) SaveRating(ctx context.Context, r Rating) (Rating, error) {
	s.writes.Add(1)
	return s.Store.SaveRating(ctx, r)
}

func ratingPath(id string) string { return "/api/v1/conversations/" + id + "/rating" }

func ratingResponse(t *testing.T, w *httptest.ResponseRecorder) Rating {
	t.Helper()
	if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("rating response %d %s", w.Code, w.Body)
	}
	var result Rating
	if strictJSON(w.Body.Bytes(), &result) != nil || result.Value < 1 || result.Value > 5 || result.RatedAt.IsZero() {
		t.Fatalf("rating response invalid: %s", w.Body)
	}
	var keys map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &keys)
	if len(keys) != 3 || keys["comment"] == nil || keys["rated_at"] == nil {
		t.Fatalf("rating envelope %s", w.Body)
	}
	return result
}

func TestContractHTTPRatingValidationAuthorizationAndRevision(t *testing.T) {
	s := testStore(t)
	h := escalationHandler(t, s, nil)
	id, cap := createHTTP(t, h, `{}`)
	path := ratingPath(id)
	assertCode(t, perform(h, "POST", path, cap, `{"rating":5}`), 409, "rating_requires_turn")
	for _, body := range []string{`{}`, `[]`, `null`, `{"rating":null}`, `{"rating":0}`, `{"rating":6}`, `{"rating":1.0}`, `{"rating":1e0}`, `{"rating":"1"}`, `{"rating":true}`, `{"rating":5,"comment":null}`, `{"rating":5,"comment":9}`, `{"rating":5,"comment":"x","comment":"y"}`, `{"rating":5,"comm\u0065nt":"x","comment":"y"}`, `{"rating":5,"other":1}`, `{"Rating":5}`, `{"rating":5} {}`, string([]byte{0xff}), `{"rating":5,"comment":"` + strings.Repeat("a", 1001) + `"}`, `{"rating":5,"comment":"` + strings.Repeat("世", 668) + `"}`, `{"rating":5,"comment":"\u0000"}`} {
		assertCode(t, perform(h, "POST", path, cap, body), 400, "invalid_request")
	}
	assertCode(t, perform(h, "POST", path, cap, strings.Repeat("a", 4097)), 413, "request_too_large")
	assertCode(t, perform(h, "POST", path+"?x=1", cap, `{"rating":5}`), 400, "invalid_request")
	assertCode(t, perform(h, "POST", ratingPath("bad"), cap, `{"rating":5}`), 400, "invalid_request")
	assertCode(t, perform(h, "GET", path, cap, ""), 405, "method_not_allowed")
	for _, token := range []string{"", "bad", strings.Repeat("d", 64)} {
		assertCode(t, perform(h, "POST", path, token, `{"rating":5}`), 401, "unauthorized")
	}
	assertCode(t, perform(h, "POST", ratingPath(strings.Repeat("f", 32)), cap, `{"rating":5}`), 401, "unauthorized")
	r := httptest.NewRequest("POST", path, strings.NewReader(`{"rating":5}`))
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	assertCode(t, w, 415, "unsupported_media_type")
	r = httptest.NewRequest("POST", path, strings.NewReader(`{"rating":5}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+cap)
	r.Header.Set("Origin", "null")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	assertCode(t, w, 403, "origin_not_allowed")
	// Independent request windows: the validation traffic above consumes the
	// default peer's window, not this direct peer's.
	post := func(body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", path, strings.NewReader(body))
		r.RemoteAddr = "10.12.0.1:4567"
		r.Header.Set("Authorization", "Bearer "+cap)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	turn := perform(h, "POST", "/api/v1/conversations/"+id+"/messages", cap, postBody("request_001", "unknown question"))
	if turn.Code != 200 {
		t.Fatalf("turn: %d %s", turn.Code, turn.Body)
	}
	first := ratingResponse(t, post(`{"rating":5,"comment":"  Helpful  "}`))
	if first.Comment != "Helpful" {
		t.Fatal(first)
	}
	replayed := ratingResponse(t, post(`{"rating":5,"comment":"Helpful"}`))
	if !replayed.RatedAt.Equal(first.RatedAt) {
		t.Fatal("replay changed timestamp")
	}
	changed := ratingResponse(t, post(`{"rating":4}`))
	if changed.Comment != "" || !changed.RatedAt.After(first.RatedAt) {
		t.Fatal("replacement not monotonic", changed)
	}
	for n := 0; n < 7; n++ {
		ratingResponse(t, post(`{"rating":4}`))
	}
	assertCode(t, post(`{"rating":4}`), 429, "rate_limited")
	other, otherCap := createHTTP(t, h, `{}`)
	assertCode(t, perform(h, "POST", ratingPath(other), cap, `{"rating":5}`), 401, "unauthorized")
	assertCode(t, perform(h, "POST", ratingPath(id), otherCap, `{"rating":5}`), 401, "unauthorized")
}

// Read the persisted columns directly, independently of SaveRating's RETURNING
// decoder, so durability checks cannot accidentally perform another write.
func persistedRating(t *testing.T, s *SQLStore, id string) Rating {
	t.Helper()
	var r Rating
	var timestamp string
	if err := s.db.QueryRow(`SELECT conversation_id,rating,comment,rated_at FROM ratings WHERE conversation_id=?`, id).
		Scan(&r.ConversationID, &r.Value, &r.Comment, &timestamp); err != nil {
		t.Fatal("read persisted rating", err)
	}
	var err error
	r.RatedAt, err = time.Parse(time.RFC3339Nano, timestamp)
	if err != nil || timestamp != ratingTimestamp(r.RatedAt) {
		t.Fatalf("noncanonical persisted timestamp %q: %v", timestamp, err)
	}
	return r
}

func assertSameRating(t *testing.T, got, want Rating) {
	t.Helper()
	if got.ConversationID != want.ConversationID || got.Value != want.Value || got.Comment != want.Comment ||
		got.RatedAt.IsZero() || !got.RatedAt.Equal(want.RatedAt) {
		t.Fatalf("rating mismatch: got %+v, want %+v", got, want)
	}
}

// Every submission has a unique comment and the same old proposed clock. Thus
// distinct serialized replacements must advance the stored clock, and the
// greatest returned timestamp identifies the last committed row, not merely the
// goroutine whose response happened to arrive last.
func concurrentDistinctRatings(t *testing.T, stores []*SQLStore, initial Rating) Rating {
	t.Helper()
	type outcome struct {
		submitted Rating
		stored    Rating
		err       error
	}
	const writes = 24
	start := make(chan struct{})
	results := make(chan outcome, writes)
	for i := 0; i < writes; i++ {
		proposed := Rating{initial.ConversationID, (i % 5) + 1, fmt.Sprintf("revision %d", i), time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)}
		go func(s *SQLStore, submitted Rating) {
			<-start
			r, err := s.SaveRating(context.Background(), submitted)
			results <- outcome{submitted, r, err}
		}(stores[i%len(stores)], proposed)
	}
	close(start)
	latest := initial
	seen := make(map[string]bool, writes)
	for i := 0; i < writes; i++ {
		result := <-results
		if result.err != nil {
			t.Errorf("concurrent rating %+v: %v", result.submitted, result.err)
			continue
		}
		r := result.stored
		if r.ConversationID != result.submitted.ConversationID || r.Value != result.submitted.Value || r.Comment != result.submitted.Comment {
			t.Errorf("concurrent response mixed/replaced fields: submitted %+v, returned %+v", result.submitted, r)
		}
		stamp := ratingTimestamp(r.RatedAt)
		if !r.RatedAt.After(initial.RatedAt) || seen[stamp] {
			t.Errorf("distinct revision reused/nonmonotonic timestamp: %+v", r)
		}
		seen[stamp] = true
		if r.RatedAt.After(latest.RatedAt) {
			latest = r
		}
	}
	if t.Failed() {
		t.FailNow()
	}
	return latest
}

func TestSQLiteRatingAtomicRestartAndMigration(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "rating.db")
	s, closeStore, err := OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = closeStore() }()
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	c := testConversation(t, s, strings.Repeat("a", 32))
	proposed := Rating{c.ID, 5, "great", time.Now().UTC()}
	if _, err := s.SaveRating(ctx, proposed); !errors.Is(err, ErrRatingRequiresTurn) {
		t.Fatal("zero-turn", err)
	}
	turn := testTurn(c, "request_001", "start", 1)
	turn.Assistant.Status = "needs_clarification"
	turn.Assistant.Sources = []Source{}
	storedTurn, err := s.CompleteTurn(ctx, turn, 0)
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.SaveRating(ctx, proposed)
	if err != nil {
		t.Fatal(err)
	}
	proposed.RatedAt = proposed.RatedAt.Add(-time.Hour)
	if again, err := s.SaveRating(ctx, proposed); err != nil || !again.RatedAt.Equal(first.RatedAt) {
		t.Fatal("replay", again, err)
	}
	proposed.Value = 3
	second, err := s.SaveRating(ctx, proposed)
	if err != nil || !second.RatedAt.After(first.RatedAt) {
		t.Fatal("monotonic replacement", second, err)
	}
	if _, err := s.db.Exec(`CREATE TRIGGER reject_rating BEFORE UPDATE ON ratings BEGIN SELECT RAISE(ABORT,'fixture failure'); END`); err != nil {
		t.Fatal(err)
	}
	proposed.Value = 2
	if _, err := s.SaveRating(ctx, proposed); !errors.Is(err, ErrUnavailable) {
		t.Fatal("failed write", err)
	}
	if _, err := s.db.Exec(`DROP TRIGGER reject_rating`); err != nil {
		t.Fatal(err)
	}
	assertSameRating(t, persistedRating(t, s, c.ID), second)
	otherStore, closeOther, err := OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = closeOther() }()
	lastCommitted := concurrentDistinctRatings(t, []*SQLStore{s, otherStore}, second)
	beforeRestart := persistedRating(t, s, c.ID)
	assertSameRating(t, beforeRestart, lastCommitted)
	if err := closeOther(); err != nil {
		t.Fatal(err)
	}
	if err := closeStore(); err != nil {
		t.Fatal(err)
	}
	s, closeStore, err = OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	assertSameRating(t, persistedRating(t, s, c.ID), beforeRestart)
	replayInput := beforeRestart
	replayInput.RatedAt = time.Now().UTC().Add(time.Hour)
	replayed, err := s.SaveRating(ctx, replayInput)
	if err != nil {
		t.Fatal("post-restart replay", err)
	}
	assertSameRating(t, replayed, beforeRestart)
	assertSameRating(t, persistedRating(t, s, c.ID), beforeRestart)
	reopenedTurn, err := s.FindTurn(ctx, c.ID, turn.RequestID)
	if err != nil || !reflect.DeepEqual(reopenedTurn, storedTurn) {
		t.Fatalf("rating writes changed completed turn: %+v, %v", reopenedTurn, err)
	}
	reopenedConversation, err := s.Conversation(ctx, c.ID)
	if err != nil || !reflect.DeepEqual(reopenedConversation, c) {
		t.Fatalf("rating writes changed conversation: %+v, %v", reopenedConversation, err)
	}
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM ratings WHERE conversation_id=?`, c.ID).Scan(&count); err != nil || count != 1 {
		t.Fatal("one durable row", count, err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM turns WHERE conversation_id=?`, c.ID).Scan(&count); err != nil || count != 1 {
		t.Fatal("turn changed", count, err)
	}
}

func TestSQLiteRatingConcurrentIdenticalWritesReplaySingleTimestamp(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "same-rating.db")
	firstStore, closeFirst, err := OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = closeFirst() }()
	if err := firstStore.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	secondStore, closeSecond, err := OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = closeSecond() }()
	c := testConversation(t, firstStore, strings.Repeat("a", 32))
	if _, err := firstStore.CompleteTurn(ctx, testTurn(c, "request_001", "start", 1), 0); err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 20; attempt++ {
		comment := fmt.Sprintf("same canonical comment %d", attempt)
		start := make(chan struct{})
		type outcome struct {
			rating Rating
			err    error
		}
		results := make(chan outcome, 2)
		for _, store := range []*SQLStore{firstStore, secondStore} {
			go func(s *SQLStore) {
				<-start
				r, err := s.SaveRating(ctx, Rating{c.ID, 5, comment, time.Now().UTC()})
				results <- outcome{r, err}
			}(store)
		}
		close(start)
		left, right := <-results, <-results
		if left.err != nil || right.err != nil {
			t.Fatalf("identical race %d: %v, %v", attempt, left.err, right.err)
		}
		if left.rating.Value != 5 || right.rating.Value != 5 || left.rating.Comment != comment || right.rating.Comment != comment ||
			left.rating.RatedAt.IsZero() || !left.rating.RatedAt.Equal(right.rating.RatedAt) {
			t.Fatalf("identical race %d did not replay one timestamp: %+v %+v", attempt, left.rating, right.rating)
		}
		var count int
		var storedAt string
		if err := firstStore.db.QueryRowContext(ctx, `SELECT COUNT(*), MAX(rated_at) FROM ratings WHERE conversation_id=?`, c.ID).Scan(&count, &storedAt); err != nil || count != 1 || storedAt != ratingTimestamp(left.rating.RatedAt) {
			t.Fatalf("identical race %d did not persist one returned row: count=%d time=%q err=%v", attempt, count, storedAt, err)
		}
	}
}

func TestContractRatingAuthorizationBeforeStorage(t *testing.T) {
	spy := &ratingSpyStore{Store: testStore(t)}
	h := escalationHandler(t, spy, nil)
	id, capability := createHTTP(t, h, `{}`)
	path := ratingPath(id)
	var unauthorized string
	for _, attempt := range []struct{ id, token string }{
		{id, ""}, {id, "invalid"}, {id, strings.Repeat("f", 64)}, {strings.Repeat("d", 32), capability},
	} {
		w := perform(h, "POST", ratingPath(attempt.id), attempt.token, `{"rating":5}`)
		assertCode(t, w, 401, "unauthorized")
		if unauthorized != "" && unauthorized != w.Body.String() {
			t.Fatal("authorization leaks existence")
		}
		unauthorized = w.Body.String()
	}
	if spy.writes.Load() != 0 {
		t.Fatal("rating store touched without authorization")
	}
	assertCode(t, perform(h, "POST", path, capability, `{"rating":5}`), 409, "rating_requires_turn")
	if spy.writes.Load() != 1 {
		t.Fatal("authorized rating did not reach storage")
	}
}

func TestContractHTTPRatingStorageFailureReturnsSafe503(t *testing.T) {
	s := testStore(t)
	h := escalationHandler(t, s, nil)
	id, capability := createHTTP(t, h, `{}`)
	turnPath := "/api/v1/conversations/" + id + "/messages"
	turnResponse := perform(h, "POST", turnPath, capability, postBody("request_001", "unknown question"))
	if turnResponse.Code != 200 {
		t.Fatalf("completed clarification: %d %s", turnResponse.Code, turnResponse.Body)
	}
	path := ratingPath(id)
	ratingResponse(t, perform(h, "POST", path, capability, `{"rating":5,"comment":"original"}`))
	before := persistedRating(t, s, id)
	historyBefore, err := s.History(context.Background(), id, 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Fail an actual authorized SQL write; do not substitute a canned HTTP
	// response. Neither adapter diagnostics nor submitted private text may leak.
	if _, err := s.db.Exec(`CREATE TRIGGER reject_rating BEFORE UPDATE ON ratings BEGIN SELECT RAISE(ABORT,'private SQL fixture-secret stack trace'); END`); err != nil {
		t.Fatal(err)
	}
	w := perform(h, "POST", path, capability, `{"rating":1,"comment":"private rating comment"}`)
	assertCode(t, w, 503, "storage_unavailable")
	const safeBody = `{"error":{"code":"storage_unavailable","message":"The assistant is temporarily unavailable. Please retry."}}`
	if strings.TrimSpace(w.Body.String()) != safeBody {
		t.Fatalf("storage failure body is not the fixed safe envelope: %s", w.Body)
	}
	for _, private := range []string{capability, id, "private rating comment", "original", "private SQL", "fixture-secret", "stack trace"} {
		if strings.Contains(w.Body.String(), private) {
			t.Fatalf("storage error leaked %q", private)
		}
	}
	assertSameRating(t, persistedRating(t, s, id), before)
	historyAfter, err := s.History(context.Background(), id, 100, nil)
	if err != nil || !reflect.DeepEqual(historyAfter, historyBefore) {
		t.Fatalf("rating failure changed chat history: %+v, %v", historyAfter, err)
	}
	if _, err := s.db.Exec(`DROP TRIGGER reject_rating`); err != nil {
		t.Fatal(err)
	}
	ratingResponse(t, perform(h, "POST", path, capability, `{"rating":5,"comment":"original"}`))
	assertSameRating(t, persistedRating(t, s, id), before)
}

func TestContractRatingRejectedAttemptsShareBoundedWindow(t *testing.T) {
	h := escalationHandler(t, testStore(t), nil)
	id, cap := createHTTP(t, h, `{}`)
	for i := 0; i < 10; i++ {
		r := httptest.NewRequest("POST", ratingPath(id), strings.NewReader(`{"rating":5}`))
		r.RemoteAddr = "10.12.0.2:1234"
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Forwarded-For", fmt.Sprintf("192.0.2.%d", i))
		if i%2 == 0 {
			r.Header.Set("Origin", "null")
		} else {
			r.Header.Set("Authorization", "Bearer invalid")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 401 && w.Code != 403 {
			t.Fatal("rejected attempt", w.Code)
		}
	}
	r := httptest.NewRequest("POST", ratingPath(id), strings.NewReader(`{"rating":5}`))
	r.RemoteAddr = "10.12.0.2:9999"
	r.Header.Set("Authorization", "Bearer "+cap)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	assertCode(t, w, 429, "rate_limited")
	if w.Header().Get("Retry-After") != "60" {
		t.Fatal("missing Retry-After")
	}
}

func TestContractRatingTursoLoopbackAndUncertainCommit(t *testing.T) {
	backing := testStore(t)
	var drop atomic.Bool
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/pipeline" || r.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Error("bad Turso request")
			w.WriteHeader(500)
			return
		}
		var body struct {
			Requests []struct {
				Type string `json:"type"`
				Stmt *struct {
					SQL      string      `json:"sql"`
					Args     []tursoCell `json:"args"`
					WantRows bool        `json:"want_rows"`
				} `json:"stmt"`
			} `json:"requests"`
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil || len(body.Requests) != 2 || body.Requests[0].Type != "execute" || body.Requests[1].Type != "close" || body.Requests[0].Stmt == nil || !body.Requests[0].Stmt.WantRows {
			t.Error("not a single execute/close pipeline")
			w.WriteHeader(500)
			return
		}
		stmt := body.Requests[0].Stmt
		args := make([]any, 0, len(stmt.Args))
		for _, cell := range stmt.Args {
			encoded, _ := json.Marshal(cell)
			envelope := []byte(`{"results":[{"type":"ok","response":{"type":"execute","result":{"cols":[{"name":"arg"}],"rows":[[` + string(encoded) + `]],"affected_row_count":0}}},{"type":"ok","response":{"type":"close"}}]}`)
			decoded, err := decodeTurso(envelope)
			if err != nil {
				t.Error(err)
				w.WriteHeader(500)
				return
			}
			args = append(args, decoded.rows[0][0])
		}
		result, err := backing.query(r.Context(), stmt.SQL, args...)
		if err != nil {
			w.WriteHeader(500)
			return
		}
		if drop.Swap(false) {
			w.WriteHeader(500)
			return
		}
		_, _ = w.Write(tursoEnvelope(result))
	}))
	defer fixture.Close()
	s := newTurso(fixture.URL+"/v2/pipeline", "fixture-token", fixture.Client())
	ctx := context.Background()
	if err := s.Migrate(ctx); err != nil {
		t.Fatal("migration fixture", err)
	}
	c := testConversation(t, s, strings.Repeat("a", 32))
	proposed := Rating{c.ID, 5, "first", time.Now().UTC()}
	if _, err := s.SaveRating(ctx, proposed); !errors.Is(err, ErrRatingRequiresTurn) {
		t.Fatal("remote zero-turn", err)
	}
	if _, err := s.CompleteTurn(ctx, testTurn(c, "request_001", "start", 1), 0); err != nil {
		t.Fatal(err)
	}
	first, err := s.SaveRating(ctx, proposed)
	if err != nil {
		t.Fatal(err)
	}
	if replay, err := s.SaveRating(ctx, proposed); err != nil || !replay.RatedAt.Equal(first.RatedAt) {
		t.Fatal("remote replay", replay, err)
	}
	drop.Store(true)
	proposed.Comment = "updated"
	if _, err := s.SaveRating(ctx, proposed); !errors.Is(err, ErrUnavailable) {
		t.Fatal("uncertain commit claimed success", err)
	}
	// Observe the hidden committed row before retrying. A merely successful
	// retry with a later timestamp would not prove replay rather than rewrite.
	uncertain := persistedRating(t, backing, c.ID)
	if uncertain.ConversationID != c.ID || uncertain.Value != proposed.Value || uncertain.Comment != proposed.Comment || !uncertain.RatedAt.After(first.RatedAt) {
		t.Fatalf("uncertain write did not commit the proposed whole row: %+v", uncertain)
	}
	proposed.RatedAt = uncertain.RatedAt.Add(time.Hour)
	confirmed, err := s.SaveRating(ctx, proposed)
	if err != nil {
		t.Fatal("retry did not return stored revision", err)
	}
	assertSameRating(t, confirmed, uncertain)
	assertSameRating(t, persistedRating(t, backing, c.ID), uncertain)

	// Exercise concurrent HTTP pipelines, not just the SQLite adapter. Two
	// clients share only the loopback primary fixture and disposable database.
	other := newTurso(fixture.URL+"/v2/pipeline", "fixture-token", fixture.Client())
	lastCommitted := concurrentDistinctRatings(t, []*SQLStore{s, other}, confirmed)
	assertSameRating(t, persistedRating(t, backing, c.ID), lastCommitted)
	replayInput := lastCommitted
	replayInput.RatedAt = lastCommitted.RatedAt.Add(time.Hour)
	replayed, err := other.SaveRating(ctx, replayInput)
	if err != nil {
		t.Fatal("remote concurrent final replay", err)
	}
	assertSameRating(t, replayed, lastCommitted)
	assertSameRating(t, persistedRating(t, backing, c.ID), lastCommitted)
	var ratings, turns int
	if err := backing.db.QueryRow(`SELECT COUNT(*) FROM ratings WHERE conversation_id=?`, c.ID).Scan(&ratings); err != nil || ratings != 1 {
		t.Fatal("remote single rating row", ratings, err)
	}
	if err := backing.db.QueryRow(`SELECT COUNT(*) FROM turns WHERE conversation_id=?`, c.ID).Scan(&turns); err != nil || turns != 1 {
		t.Fatal("remote rating changed turns", turns, err)
	}
}

func TestSQLiteRatingUpgradeFromV2AndMissingRemoteSchema(t *testing.T) {
	ctx := context.Background()
	s, closeStore, err := OpenSQLite(filepath.Join(t.TempDir(), "old.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = closeStore() }()
	for _, version := range []string{"001_initial.sql", "002_escalation.sql"} {
		migration, err := database.Migrations.ReadFile("migrations/" + version)
		if err != nil {
			t.Fatal(err)
		}
		for _, statement := range strings.Split(string(migration), ";") {
			if strings.TrimSpace(statement) == "" {
				continue
			}
			if _, err := s.db.ExecContext(ctx, statement); err != nil {
				t.Fatal(err)
			}
		}
	}
	c := testConversation(t, s, strings.Repeat("a", 32))
	if _, err := s.CompleteTurn(ctx, testTurn(c, "request_001", "start", 1), 0); err != nil {
		t.Fatal(err)
	}
	proposed := Rating{c.ID, 5, "old database", time.Now().UTC()}
	if _, err := s.SaveRating(ctx, proposed); !errors.Is(err, ErrUnavailable) {
		t.Fatal("unmigrated schema succeeded", err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatal("upgrade", err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatal("repeated upgrade", err)
	}
	if _, err := s.SaveRating(ctx, proposed); err != nil {
		t.Fatal("post-upgrade rating", err)
	}
	var versions int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM schema_versions`).Scan(&versions); err != nil || versions != 3 {
		t.Fatal("migration versions", versions, err)
	}
}

// Property: accepted rating input is canonical, bounded, stable under replay
// parsing, and arbitrary malformed/Unicode/oversized JSON cannot panic.
func FuzzDecodeRating(f *testing.F) {
	for _, s := range []string{`{"rating":5}`, `{"rating":1.0}`, `{"rating":5,"comment":"  hi \t"}`, `{"rating":5,"comment":null}`, `{"rating":5,"comm\u0065nt":"x","comment":"y"}`, `{"rating":5,"comment":"` + strings.Repeat("世", 668) + `"}`, strings.Repeat("x", 4097), string([]byte{0xff})} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		value, comment, err := decodeRatingRequest([]byte(s))
		if err != nil {
			return
		}
		if value < 1 || value > 5 || len(comment) > 2000 || len([]rune(comment)) > 1000 {
			t.Fatal("accepted out of bounds")
		}
		again, err := canonicalRatingComment(comment)
		if err != nil || again != comment {
			t.Fatal("not canonical")
		}
	})
}
