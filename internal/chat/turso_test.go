package chat

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func tursoEnvelope(result sqlResult) []byte {
	cols := []any{}
	for _, col := range result.columns {
		cols = append(cols, map[string]any{"name": col, "decltype": nil})
	}
	rows := []any{}
	for _, row := range result.rows {
		values := []any{}
		for _, v := range row {
			cell, _ := bindCell(v)
			values = append(values, cell)
		}
		rows = append(rows, values)
	}
	b, _ := json.Marshal(map[string]any{"base_url": "https://never-follow.invalid", "baton": nil, "results": []any{map[string]any{"type": "ok", "response": map[string]any{"type": "execute", "result": map[string]any{"cols": cols, "rows": rows, "affected_row_count": result.affected, "last_insert_rowid": nil}}}, map[string]any{"type": "ok", "response": map[string]any{"type": "close"}}}})
	return b
}

func TestContractTursoTypedBindingsAndCommitAmbiguity(t *testing.T) {
	backing := testStore(t)
	ctx := context.Background()
	var calls atomic.Int32
	var loseAck atomic.Bool
	var failLookup atomic.Bool
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/v2/pipeline" || r.Method != "POST" || r.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Error("Turso endpoint/headers")
		}
		var body struct {
			Requests []struct {
				Type string `json:"type"`
				Stmt *struct {
					SQL      string      `json:"sql"`
					Args     []tursoCell `json:"args"`
					WantRows bool        `json:"want_rows"`
				} `json:"stmt,omitempty"`
			} `json:"requests"`
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil || len(body.Requests) != 2 || body.Requests[0].Type != "execute" || body.Requests[0].Stmt == nil || body.Requests[1].Type != "close" || body.Requests[1].Stmt != nil {
			t.Error("execute and close contract")
			w.WriteHeader(500)
			return
		}
		stmt := body.Requests[0].Stmt
		if !stmt.WantRows {
			t.Error("want_rows omitted")
		}
		args := []any{}
		for _, c := range stmt.Args {
			// Reuse the strict typed-result boundary to decode fixture bind cells.
			cellJSON, _ := json.Marshal(c)
			raw := []byte(`{"results":[{"type":"ok","response":{"type":"execute","result":{"cols":[{"name":"arg"}],"rows":[[` + string(cellJSON) + `]],"affected_row_count":0}}},{"type":"ok","response":{"type":"close"}}]}`)
			decoded, err := decodeTurso(raw)
			if err != nil {
				t.Error("untyped SQL bind", err)
				w.WriteHeader(500)
				return
			}
			args = append(args, decoded.rows[0][0])
		}
		if failLookup.Load() && strings.HasPrefix(stmt.SQL, "SELECT "+turnColumns) {
			w.WriteHeader(503)
			return
		}
		result, err := backing.query(r.Context(), stmt.SQL, args...)
		if err != nil {
			_, _ = w.Write([]byte(`{"results":[{"type":"error","error":{"message":"private SQL failure"}},{"type":"ok","response":{"type":"close"}}]}`))
			return
		}
		if strings.HasPrefix(stmt.SQL, "INSERT INTO turns") && loseAck.Swap(false) {
			w.WriteHeader(500)
			return
		}
		_, _ = w.Write(tursoEnvelope(result))
	}))
	defer fixture.Close()
	s := newTurso(fixture.URL+"/v2/pipeline", "fixture-token", fixture.Client())
	if calls.Load() != 0 {
		t.Fatal("constructor mutated remote database")
	}
	if err := s.SeedDemo(ctx); !errors.Is(err, ErrInvalid) || calls.Load() != 0 {
		t.Fatal("remote demo seed allowed")
	}
	// Explicit migration works against a local protocol fixture only.
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	c := testConversation(t, s, strings.Repeat("a", 32))
	turn := testTurn(c, "request_001", "start ' OR 1=1 --", 1)
	loseAck.Store(true)
	stored, err := s.CompleteTurn(ctx, turn, 0)
	if err != nil || stored.User.Text != turn.User.Text {
		t.Fatal("lost commit acknowledgment not resolved", err)
	}
	p, err := s.History(ctx, c.ID, 1, nil)
	if err != nil || len(p.Messages) != 1 || p.Messages[0].Seq != 2 {
		t.Fatal("remote history", err)
	}
	turn2 := testTurn(c, "request_002", "start", 2)
	loseAck.Store(true)
	failLookup.Store(true)
	if _, err = s.CompleteTurn(ctx, turn2, 1); !errors.Is(err, ErrUnavailable) {
		t.Fatal("unknown commit not unavailable", err)
	}
	failLookup.Store(false)
	stored2, err := s.CompleteTurn(ctx, turn2, 1)
	if err != nil || stored2.Number != 2 {
		t.Fatal("same-ID retry after ambiguity", err)
	}
	all, err := s.History(ctx, c.ID, 100, nil)
	if err != nil || len(all.Messages) != 4 {
		t.Fatal("ambiguous commit duplicated", err)
	}
	divergent := testTurn(c, "request_002", "changed", 2)
	if _, err = s.CompleteTurn(ctx, divergent, 1); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatal(err)
	}
	for _, endpoint := range []string{"http://remote.invalid", "https://u:p@remote.invalid", "https://remote.invalid/?token=x", "libsql://remote.invalid"} {
		if _, err := OpenTurso(endpoint, "fixture"); err == nil {
			t.Fatal("unsafe remote config")
		}
	}
}

func TestContractTursoFailureDecoding(t *testing.T) {
	valid := tursoEnvelope(sqlResult{columns: []string{"number", "blob", "text", "empty"}, rows: [][]any{{int64(42), []byte{0, 1}, "世界", nil}}, affected: 1})
	r, err := decodeTurso(valid)
	if err != nil || r.rows[0][0] != int64(42) || !reflect.DeepEqual(r.rows[0][1], []byte{0, 1}) || r.rows[0][2] != "世界" || r.rows[0][3] != nil {
		t.Fatal(r, err)
	}
	for _, raw := range []string{`{}`, `{"results":[]}`, strings.Replace(string(valid), `"type":"ok"`, `"type":"error"`, 1), strings.Replace(string(valid), `"type":"execute"`, `"type":"close"`, 1), strings.Replace(string(valid), `"value":"42"`, `"value":42`, 1), strings.Replace(string(valid), `"value":"42"`, `"value":"9223372036854775808"`, 1), strings.Replace(string(valid), `"value":"42"`, `"value":"01"`, 1), strings.Replace(string(valid), `"type":"blob"`, `"type":"float"`, 1), strings.Replace(string(valid), `"affected_row_count":1`, `"affected_row_count":-1`, 1), strings.Replace(string(valid), `"base64":"AAE="`, `"base64":"!!!!"`, 1), strings.Replace(string(valid), `"name":"number"`, `"name":null`, 1), string(valid) + `{}`} {
		if _, err := decodeTurso([]byte(raw)); err == nil {
			t.Fatalf("malformed typed result accepted %s", raw)
		}
	}
	for _, tc := range []struct {
		status int
		body   []byte
	}{{200, []byte(`{"results":[]}`)}, {429, []byte("private")}, {500, []byte("private")}, {200, []byte(strings.Repeat("x", storageBodyLimit+1))}} {
		fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.status); _, _ = w.Write(tc.body) }))
		if _, err := newTurso(fixture.URL, "fixture", fixture.Client()).query(context.Background(), "SELECT ?", int64(1)); err == nil {
			t.Error("bad HTTP result accepted")
		}
		fixture.Close()
	}
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { time.Sleep(50 * time.Millisecond); _, _ = w.Write(valid) }))
	defer fixture.Close()
	client := fixture.Client()
	client.Timeout = 5 * time.Millisecond
	if _, err := newTurso(fixture.URL, "fixture", client).query(context.Background(), "SELECT 1"); err == nil {
		t.Fatal("storage timeout accepted")
	}
	var redirects atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { redirects.Add(1) }))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 307) }))
	defer redirect.Close()
	if _, err := newTurso(redirect.URL, "fixture", redirect.Client()).query(context.Background(), "SELECT 1"); err == nil || redirects.Load() != 0 {
		t.Fatal("storage redirect followed")
	}
}

// Property: accepted pipeline results have two successful operations, bounded
// widths/counts and only exact typed scalar values. Typed roundtrip is stable.
func FuzzDecodeTurso(f *testing.F) {
	for _, b := range [][]byte{tursoEnvelope(sqlResult{columns: []string{"n", "s", "b", "null"}, rows: [][]any{{int64(1), "世界", []byte{0, 255}, nil}}, affected: 1}), []byte(`{"results":[{"type":"error"}]}`), {0xff}, []byte(strings.Repeat("x", storageBodyLimit+1))} {
		f.Add(b)
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		r, err := decodeTurso(b)
		if err != nil {
			return
		}
		if len(r.rows) > 101 || len(r.columns) > 16 || r.affected < 0 || r.affected > 100 {
			t.Fatal("result bound")
		}
		for _, row := range r.rows {
			if len(row) != len(r.columns) {
				t.Fatal("row width")
			}
			for _, v := range row {
				switch v.(type) {
				case nil, string, int64, []byte:
				default:
					t.Fatal("untyped value")
				}
			}
		}
		again, err := decodeTurso(tursoEnvelope(r))
		if err != nil || !reflect.DeepEqual(again.rows, r.rows) || again.affected != r.affected {
			t.Fatal("typed roundtrip", err)
		}
	})
}
