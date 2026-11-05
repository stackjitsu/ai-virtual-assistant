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
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"
	"unicode/utf8"
)

func escalationPath(id string) string { return "/api/v1/conversations/" + id + "/escalation" }

func escalationBody(email, note string) string {
	body := struct {
		Email        string  `json:"email"`
		QuestionNote *string `json:"question_note,omitempty"`
		Consent      bool    `json:"consent"`
	}{Email: email, Consent: true}
	if note != "" {
		body.QuestionNote = &note
	}
	b, _ := json.Marshal(body)
	return string(b)
}

// escalationTicket checks the exact 201 visitor-safe envelope and headers.
func escalationTicket(t *testing.T, w *httptest.ResponseRecorder) escalationEnvelope {
	t.Helper()
	if w.Code != 201 {
		t.Fatalf("escalation %d %s", w.Code, w.Body)
	}
	if w.Header().Get("Content-Type") != "application/json" || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("API security headers", w.Header())
	}
	var result escalationEnvelope
	if strictJSON(w.Body.Bytes(), &result) != nil || !idPattern.MatchString(result.TicketID) || result.RecordedAt.IsZero() ||
		result.DeliveryOutcome != DeliveryNotWired || result.Message != DeliveryNotWiredMessage {
		t.Fatalf("ticket envelope %d %s", w.Code, w.Body)
	}
	return result
}

type outboxSpy struct {
	mu       sync.Mutex
	recorded []Escalation
	fail     atomic.Bool
}

func (o *outboxSpy) Deliver(_ context.Context, t Escalation) error {
	o.mu.Lock()
	o.recorded = append(o.recorded, t)
	o.mu.Unlock()
	if o.fail.Load() {
		return ErrUnavailable
	}
	return nil
}

func (o *outboxSpy) delivered() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.recorded)
}

func (o *outboxSpy) last() Escalation {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.recorded[len(o.recorded)-1]
}

func escalationHandler(t *testing.T, s Store, outbox EmailOutbox) http.Handler {
	t.Helper()
	h, err := NewHandlerWithOutbox(s, providerFunc(func(context.Context, ComposeInput) (Composition, error) {
		t.Error("escalation invoked paid inference")
		return Composition{}, ErrUnavailable
	}), HTTPConfig{AssistantOrigin: "http://127.0.0.1:8080", ParentOrigins: []string{"http://localhost:8080", "http://127.0.0.1:8080"}},
		fstest.MapFS{"widget.html": &fstest.MapFile{Data: []byte("fixture assets")}, "sample-host.html": &fstest.MapFile{Data: []byte("fixture host")}}, outbox)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestContractHTTPEscalationValidationConsentAndAuth(t *testing.T) {
	s := testStore(t)
	h := escalationHandler(t, s, nil)
	id, capability := createHTTP(t, h, `{}`)
	path := escalationPath(id)
	for name, body := range map[string]string{
		"empty body":             `{}`,
		"missing consent":        `{"email":"visitor@example.test"}`,
		"consent false":          `{"email":"visitor@example.test","consent":false}`,
		"consent null":           `{"email":"visitor@example.test","consent":null}`,
		"consent string":         `{"email":"visitor@example.test","consent":"true"}`,
		"consent number":         `{"email":"visitor@example.test","consent":1}`,
		"missing email":          `{"consent":true}`,
		"email null":             `{"email":null,"consent":true}`,
		"email empty":            `{"email":"","consent":true}`,
		"email no at":            `{"email":"visitor.example.test","consent":true}`,
		"email two at":           `{"email":"v@x@example.test","consent":true}`,
		"email no tld dot":       `{"email":"v@example","consent":true}`,
		"email local dot edge":   `{"email":".v@example.test","consent":true}`,
		"email local sep run":    `{"email":"v..x@example.test","consent":true}`,
		"email leading hyphen":   `{"email":"v@-example.test","consent":true}`,
		"email numeric tld":      `{"email":"v@example.t3","consent":true}`,
		"email domain gap":       `{"email":"v@example.test","consent":true,"x":1}`,
		"email space":            `{"email":"v x@example.test","consent":true}`,
		"email multibyte":        `{"email":"vístor@example.test","consent":true}`,
		"email raw control":      "{\x01}",
		"email local over 64":    `{"email":"` + strings.Repeat("a", 65) + `@example.test","consent":true}`,
		"email over 254 bytes":   `{"email":"` + strings.Repeat("a", 250) + `@example.test","consent":true}`,
		"note over bytes":        `{"email":"v@example.test","question_note":"` + strings.Repeat("世", 1001) + `","consent":true}`,
		"note over code points":  `{"email":"v@example.test","question_note":"` + strings.Repeat("a", 1001) + `","consent":true}`,
		"note control":           `{"email":"v@example.test","question_note":"note` + "\x00" + `","consent":true}`,
		"unknown field":          `{"email":"v@example.test","consent":true,"extra":1}`,
		"duplicate email":        `{"email":"v@example.test","email":"w@example.test","consent":true}`,
		"duplicate escaped":      `{"email":"v@example.test","consent":true,"question_note":"a","question_note":"a"}`,
		"trailing json":          `{"email":"v@example.test","consent":true} {}`,
		"array root":             `[]`,
		"upper-case key consent": `{"email":"v@example.test","Consent":true}`,
		"upper-case key email":   `{"Email":"v@example.test","consent":true}`,
		"invalid utf-8":          string([]byte{0xff}),
		"wrong note type":        `{"email":"v@example.test","question_note":12,"consent":true}`,
		"wrong email type":       `{"email":true,"consent":true}`,
	} {
		t.Run(name, func(t *testing.T) {
			assertCode(t, perform(h, "POST", path, capability, body), 400, "invalid_request")
		})
	}
	// A rejected request stores nothing: consent fails closed and invalid
	// payloads never persist partial tickets.
	if n, err := s.EscalationCount(context.Background(), id); err != nil || n != 0 {
		t.Fatalf("rejected escalations stored %d %v", n, err)
	}
	assertCode(t, perform(h, "POST", path, capability, strings.Repeat("x", 4097)), 413, "request_too_large")
	assertCode(t, perform(h, "POST", path+"?x=1", capability, escalationBody("v@example.test", "note")), 400, "invalid_request")
	assertCode(t, perform(h, "GET", path, capability, ""), 405, "method_not_allowed")
	assertCode(t, perform(h, "DELETE", path, capability, ""), 405, "method_not_allowed")
	r := httptest.NewRequest("POST", path, strings.NewReader(escalationBody("v@example.test", "note")))
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	assertCode(t, w, 415, "unsupported_media_type")
	unauthorizedBody := ""
	for _, auth := range []string{"", strings.Repeat("f", 64), "wrong", "bad"} {
		w := perform(h, "POST", path, auth, escalationBody("v@example.test", "note"))
		assertCode(t, w, 401, "unauthorized")
		if unauthorizedBody == "" {
			unauthorizedBody = w.Body.String()
		} else if unauthorizedBody != w.Body.String() {
			t.Fatal("auth existence leak")
		}
	}
	unknown := perform(h, "POST", escalationPath(strings.Repeat("0", 32)), capability, escalationBody("v@example.test", "note"))
	assertCode(t, unknown, 401, "unauthorized")
	if unknown.Body.String() != unauthorizedBody {
		t.Fatal("unknown conversation oracle leak")
	}
	// Escalations never consume turns or mutate the message history.
	p, err := s.History(context.Background(), id, 100, nil)
	if err != nil || len(p.Messages) != 0 {
		t.Fatal("escalation mutated history", p, err)
	}
}

func TestContractHTTPEscalationTicketReplayCapAndIsolation(t *testing.T) {
	s := testStore(t)
	spy := &outboxSpy{}
	h := escalationHandler(t, s, spy)
	id, capability := createHTTP(t, h, `{}`)
	path := escalationPath(id)
	first := perform(h, "POST", path, capability, `{"email":"VISITOR@Example.COM","question_note":"  please  ","consent":true}`)
	stored := escalationTicket(t, first)
	replayed := perform(h, "POST", path, capability, `{"email":"visitor@example.com","question_note":"please","consent":true}`)
	if replayed.Code != 201 || replayed.Body.String() != first.Body.String() {
		t.Fatal("duplicate did not replay the stored ticket byte-for-byte", replayed.Body)
	}
	if spy.delivered() != 1 {
		t.Fatalf("outbox re-recorded on replay %d", spy.delivered())
	}
	ticket := spy.last()
	if ticket.Email != "visitor@example.com" || ticket.QuestionNote != "please" {
		t.Fatalf("canonical email/note not stored %q %q", ticket.Email, ticket.QuestionNote)
	}
	if again, err := ValidateEscalationEmail(ticket.Email); err != nil || again != ticket.Email {
		t.Fatal("stored email not canonical", ticket.Email)
	}
	// A different canonical pair appends a new immutable ticket record.
	second := escalationTicket(t, perform(h, "POST", path, capability, `{"email":"visitor@example.com","question_note":"different","consent":true}`))
	if second.TicketID == stored.TicketID {
		t.Fatal("different payload reused ticket id")
	}
	if spy.delivered() != 2 {
		t.Fatalf("outbox missed the new ticket %d", spy.delivered())
	}
	if w := perform(h, "POST", path, capability, `{"email":"visitor@example.com","question_note":"please","consent":true}`); w.Code != 201 || w.Body.String() != first.Body.String() || spy.delivered() != 2 {
		t.Fatal("replay identity broken")
	}
	// Honest outcome without transport: the reply names recording, never a
	// sent email; delivery_not_wired is the only outcome in this slice.
	if !strings.Contains(stored.Message, "not configured") || !strings.Contains(stored.Message, "no email has been sent") {
		t.Fatal("dishonest delivery wording", stored.Message)
	}
	for n := 3; n <= 5; n++ {
		escalationTicket(t, perform(h, "POST", path, capability, fmt.Sprintf(`{"email":"v%d@example.test","consent":true}`, n)))
	}
	assertCode(t, perform(h, "POST", path, capability, `{"email":"overflow@example.test","consent":true}`), 409, "conflict")
	if n, err := s.EscalationCount(context.Background(), id); err != nil || n != 5 {
		t.Fatalf("cap violated %d %v", n, err)
	}
	if w := perform(h, "POST", path, capability, `{"email":"visitor@example.com","question_note":"please","consent":true}`); w.Code != 201 || w.Body.String() != first.Body.String() {
		t.Fatal("full conversation dropped replay", w.Body)
	}
	if w := perform(h, "GET", "/api/v1/conversations/"+id+"/messages", capability, ""); w.Code != 200 || w.Body.String() != "{\"messages\":[],\"next_before\":null}\n" {
		t.Fatal("escalation changed history", w.Body)
	}
	// Chat turns stay available after escalations and never touch tickets.
	turn := perform(h, "POST", "/api/v1/conversations/"+id+"/messages", capability, postBody("request_001", "start onboarding"))
	if turn.Code != 200 {
		t.Fatalf("escalation blocked the chat turn %d %s", turn.Code, turn.Body)
	}
	if n, err := s.EscalationCount(context.Background(), id); err != nil || n != 5 {
		t.Fatal("chat turn touched tickets", n, err)
	}
	p, err := s.History(context.Background(), id, 100, nil)
	if err != nil || len(p.Messages) != 2 || p.Messages[0].Seq != 1 || p.Messages[1].Seq != 2 {
		t.Fatal("escalation mutated turns", p, err)
	}
}

func TestContractHTTPEscalationRateLimits(t *testing.T) {
	s := testStore(t)
	h := escalationHandler(t, s, nil)
	id, _ := createHTTP(t, h, `{}`)
	send := func(peer string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", escalationPath(id), strings.NewReader(escalationBody("v@example.test", "note")))
		r.RemoteAddr = peer + ":1234"
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	for n := 0; n < 10; n++ {
		if w := send("10.9.0.1"); w.Code != 401 {
			t.Fatalf("unauth attempt %d got %d", n, w.Code)
		}
	}
	// Rejected and unauthorized attempts count toward the same fixed window.
	w := send("10.9.0.1")
	assertCode(t, w, 429, "rate_limited")
	if w.Header().Get("Retry-After") != "60" {
		t.Fatal("rate retry hint")
	}
	if n, err := s.EscalationCount(context.Background(), id); err != nil || n != 0 {
		t.Fatal("rate limited attempts stored", n, err)
	}
	id2, _ := createHTTP(t, h, `{}`)
	for n := 0; n < 10; n++ {
		send("10.9.0.2")
	}
	assertCode(t, send("10.9.0.2"), 429, "rate_limited")
	if n, err := s.EscalationCount(context.Background(), id2); err != nil || n != 0 {
		t.Fatal("rate limited attempts stored", n, err)
	}
}

func TestContractHTTPEscalationHonestOutbox(t *testing.T) {
	s := testStore(t)
	spy := &outboxSpy{}
	h := escalationHandler(t, s, spy)
	id, capability := createHTTP(t, h, `{}`)
	path := escalationPath(id)
	first := escalationTicket(t, perform(h, "POST", path, capability, escalationBody("v@example.test", "note")))
	if spy.delivered() != 1 {
		t.Fatal("outbox not consulted")
	}
	if got := spy.last(); got.ConversationID != id || got.TicketID != first.TicketID || got.DeliveryOutcome != DeliveryNotWired {
		t.Fatalf("delivered record %+v", got)
	}
	spy.fail.Store(true)
	// A delivery-observer failure can never change the recorded honest outcome.
	second := escalationTicket(t, perform(h, "POST", path, capability, escalationBody("w@example.test", "")))
	if second.TicketID == first.TicketID || second.Message != DeliveryNotWiredMessage {
		t.Fatal("observer failure leaked into the visitor response", second)
	}
	if n, err := s.EscalationCount(context.Background(), id); err != nil || n != 2 {
		t.Fatalf("tickets after observer failure %d %v", n, err)
	}
	// The local adapter records outcomes with zero network I/O and a bound.
	box := &LocalEmailOutbox{}
	for n := 0; n < localOutboxBound+50; n++ {
		ticket := Escalation{strings.Repeat("a", 32), fmt.Sprintf("%032x", n), "v@example.test", "", time.Now().UTC(), DeliveryNotWired}
		if err := box.Deliver(context.Background(), ticket); err != nil {
			t.Fatal(err)
		}
	}
	pending := box.Pending()
	if len(pending) != localOutboxBound || pending[0].TicketID != fmt.Sprintf("%032x", 50) || pending[len(pending)-1].TicketID != fmt.Sprintf("%032x", localOutboxBound+49) {
		t.Fatalf("local outbox bound %d %v", len(pending), pending[0])
	}
	dishonest := Escalation{ConversationID: strings.Repeat("a", 32), TicketID: strings.Repeat("b", 32), Email: "BAD", RecordedAt: time.Now(), DeliveryOutcome: "email sent"}
	if err := box.Deliver(context.Background(), dishonest); !errors.Is(err, ErrInvalid) {
		t.Fatal("local outbox accepted an invalid ticket", err)
	}
}

func TestSQLiteEscalationStoreInvariantsRestartAndConcurrency(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "escalation.db")
	s, closeStore, err := OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err = s.Migrate(ctx); err != nil {
		t.Fatal("migration not idempotent", err)
	}
	c := testConversation(t, s, strings.Repeat("a", 32))
	valid := Escalation{c.ID, strings.Repeat("c", 32), "Visitor@Example.COM", "  note  ", time.Now().UTC(), DeliveryNotWired}
	valid.Email, valid.QuestionNote = "visitor@example.com", "note"
	stored, err := s.CreateEscalation(ctx, valid)
	if err != nil || !reflect.DeepEqual(stored, valid) {
		t.Fatalf("store roundtrip %+v %v", stored, err)
	}
	// Honesty is enforced at the record boundary: a claimed sent outcome is
	// impossible, as are oversized or uncanonical records.
	for name, bad := range map[string]Escalation{
		"claimed sent outcome": {c.ID, strings.Repeat("d", 32), "d@example.test", "", time.Now().UTC(), "email_sent"},
		"uncanonical email":    {c.ID, strings.Repeat("d", 32), "D@example.test", "", time.Now().UTC(), DeliveryNotWired},
		"overlong note":        {c.ID, strings.Repeat("d", 32), "d@example.test", strings.Repeat("a", 2001), time.Now().UTC(), DeliveryNotWired},
		"bad ticket id":        {c.ID, "short", "d@example.test", "", time.Now().UTC(), DeliveryNotWired},
		"zero time":            {c.ID, strings.Repeat("d", 32), "d@example.test", "", time.Time{}, DeliveryNotWired},
	} {
		if _, err := s.CreateEscalation(ctx, bad); !errors.Is(err, ErrInvalid) {
			t.Fatalf("%s accepted: %v", name, err)
		}
	}
	if _, err := s.CreateEscalation(ctx, valid); err != nil || !reflect.DeepEqual(stored, valid) {
		t.Fatal("duplicate create returned a different ticket", err)
	}
	if _, err := s.FindEscalation(ctx, c.ID, "other@example.test", ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown dedup lookup %v", err)
	}
	if n, err := s.EscalationCount(ctx, c.ID); err != nil || n != 1 {
		t.Fatalf("count %d %v", n, err)
	}
	// Identical concurrent duplicates converge on exactly one stored ticket.
	same := make(chan Escalation, 2)
	var wg sync.WaitGroup
	wg.Go(func() {
		t2, _ := s.CreateEscalation(ctx, Escalation{c.ID, strings.Repeat("e", 32), "dupe@example.test", "", time.Now().UTC(), DeliveryNotWired})
		same <- t2
	})
	wg.Go(func() {
		t2, _ := s.CreateEscalation(ctx, Escalation{c.ID, strings.Repeat("f", 32), "dupe@example.test", "", time.Now().UTC().Add(time.Second), DeliveryNotWired})
		same <- t2
	})
	wg.Wait()
	close(same)
	firstTicket := <-same
	secondTicket := <-same
	if firstTicket.TicketID != secondTicket.TicketID || !reflect.DeepEqual(firstTicket, secondTicket) {
		t.Fatalf("identical concurrent duplicates diverged %+v %+v", firstTicket, secondTicket)
	}
	if firstTicket.Email != "dupe@example.test" || firstTicket.QuestionNote != "" {
		t.Fatalf("duplicate converged on a foreign record %+v", firstTicket)
	}
	if n, err := s.EscalationCount(ctx, c.ID); err != nil || n != 2 {
		t.Fatalf("dupe convergence count %d %v", n, err)
	}
	// Racing inserts stay inside the atomic per-conversation cap.
	results := make(chan error, 10)
	for n := 3; n <= 10; n++ {
		wg.Go(func() {
			ticket := Escalation{c.ID, fmt.Sprintf("%032x", n), fmt.Sprintf("racer%d@example.test", n), "", time.Now().UTC(), DeliveryNotWired}
			_, err := s.CreateEscalation(ctx, ticket)
			results <- err
		})
	}
	wg.Wait()
	close(results)
	conflicts := 0
	for err := range results {
		if err == nil {
			continue
		}
		if errors.Is(err, ErrConflict) {
			conflicts++
			continue
		}
		t.Fatal(err)
	}
	if conflicts != 5 {
		t.Fatalf("cap concurrency results %d", conflicts)
	}
	if n, err := s.EscalationCount(ctx, c.ID); err != nil || n != 5 {
		t.Fatalf("atomic cap violated %d %v", n, err)
	}
	// At the cap an existing stored repeat still replays from storage while
	// any new payload conflicts.
	replica := Escalation{c.ID, strings.Repeat("7", 32), "visitor@example.com", "note", time.Now().UTC().Add(time.Minute), DeliveryNotWired}
	if replayed, err := s.CreateEscalation(ctx, replica); err != nil || !reflect.DeepEqual(replayed, valid) {
		t.Fatalf("cap replay %+v %v", replayed, err)
	}
	if n, err := s.EscalationCount(ctx, c.ID); err != nil || n != 5 {
		t.Fatalf("cap replay stored a ticket %d %v", n, err)
	}
	// Escalations append immutably without touching the turn sequence.
	turn := testTurn(c, "request_001", "start", 1)
	if _, err := s.CompleteTurn(ctx, turn, 0); err != nil {
		t.Fatal(err)
	}
	p, err := s.History(ctx, c.ID, 100, nil)
	if err != nil || len(p.Messages) != 2 || p.Messages[0].Seq != 1 {
		t.Fatal("escalation mutated turns", p, err)
	}
	if err = closeStore(); err != nil {
		t.Fatal(err)
	}
	// Simulated restart: a reopened store returns ticket records byte-equal.
	reopened, closeReopen, err := OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = closeReopen() }()
	if err = reopened.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	again, err := reopened.FindEscalation(ctx, c.ID, "visitor@example.com", "note")
	if err != nil || !reflect.DeepEqual(again, valid) {
		t.Fatalf("restart persistence %+v %v", again, err)
	}
	if n, err := reopened.EscalationCount(ctx, c.ID); err != nil || n != 5 {
		t.Fatalf("restart count %d %v", n, err)
	}
	var ticketNumber int64
	if err = reopened.db.QueryRow(`SELECT ticket_number FROM escalation_tickets WHERE ticket_id=?`, valid.TicketID).Scan(&ticketNumber); err != nil || ticketNumber != 1 {
		t.Fatal("ticket number column", ticketNumber, err)
	}
}

func TestContractHTTPEscalationTursoFixture(t *testing.T) {
	backing := testStore(t)
	ctx := context.Background()
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/pipeline" || r.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Error("Turso endpoint/headers")
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
				} `json:"stmt,omitempty"`
			} `json:"requests"`
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil || len(body.Requests) != 2 || body.Requests[0].Type != "execute" || body.Requests[1].Type != "close" {
			t.Error("execute and close contract")
			w.WriteHeader(500)
			return
		}
		stmt := body.Requests[0].Stmt
		if !stmt.WantRows {
			t.Error("want_rows omitted")
		}
		args := []any{}
		for _, cell := range stmt.Args {
			cellJSON, _ := json.Marshal(cell)
			raw := []byte(`{"results":[{"type":"ok","response":{"type":"execute","result":{"cols":[{"name":"arg"}],"rows":[[` + string(cellJSON) + `]],"affected_row_count":0}}},{"type":"ok","response":{"type":"close"}}]}`)
			decoded, err := decodeTurso(raw)
			if err != nil {
				t.Error("untyped SQL bind", err)
				w.WriteHeader(500)
				return
			}
			args = append(args, decoded.rows[0][0])
		}
		result, err := backing.query(r.Context(), stmt.SQL, args...)
		if err != nil {
			_, _ = w.Write([]byte(`{"results":[{"type":"error"},{"type":"ok","response":{"type":"close"}}]}`))
			return
		}
		_, _ = w.Write(tursoEnvelope(result))
	}))
	defer fixture.Close()
	s := newTurso(fixture.URL+"/v2/pipeline", "fixture-token", fixture.Client())
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	c := testConversation(t, s, strings.Repeat("a", 32))
	h := escalationHandler(t, s, nil)
	stored := escalationTicket(t, perform(h, "POST", escalationPath(c.ID), strings.Repeat("c", 64), escalationBody("visitor@example.com", "note")))
	replayed := perform(h, "POST", escalationPath(c.ID), strings.Repeat("c", 64), escalationBody("visitor@example.com", "note"))
	if replayed.Code != 201 || !strings.Contains(replayed.Body.String(), stored.TicketID) || !strings.Contains(replayed.Body.String(), DeliveryNotWired) {
		t.Fatal("remote replay", replayed.Body)
	}
	escalationTicket(t, perform(h, "POST", escalationPath(c.ID), strings.Repeat("c", 64), escalationBody("other@example.com", "")))
	if n, err := s.EscalationCount(ctx, c.ID); err != nil || n != 2 {
		t.Fatalf("remote tickets %d %v", n, err)
	}
	if again, err := s.FindEscalation(ctx, c.ID, "visitor@example.com", "note"); err != nil || again.TicketID != stored.TicketID {
		t.Fatal("remote dedup lookup", again, err)
	}
}

// Property: escalation request decoding accepts only canonical bounded
// records with literal true consent; malformed input never panics.
func FuzzDecodeEscalation(f *testing.F) {
	for _, s := range []string{
		`{"email":"a@b.co","question_note":"hello","consent":true}`,
		`{"email":"A@B.Co","consent":true}`,
		`{"email":"a@b.co","consent":false}`,
		`{"email":"a@b.co","consent":"true"}`,
		`{"email":"a@b.co","consent":true,"email":"c@d.io"}`,
		`{"email":null,"consent":true}`,
		`{"email":"a@b.co","consent":true,"unknown":1}`,
		`{"email":"a@b.co","consent":true,"question_note":"` + strings.Repeat("a", 4000) + `"}`,
		`{}`, `[]`, string([]byte{0xff}),
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		var body struct {
			Email        *string `json:"email"`
			QuestionNote *string `json:"question_note"`
			Consent      *bool   `json:"consent"`
		}
		if strictJSON([]byte(s), &body) != nil {
			return
		}
		req, err := decodeEscalationRequest(body.Email, body.QuestionNote, body.Consent)
		if err != nil {
			return
		}
		if req.Email != strings.ToLower(strings.TrimSpace(req.Email)) || !utf8.ValidString(req.Email) || len(req.Email) > maxEscalationEmailBytes {
			t.Fatal("accepted email violates the record")
		}
		if again, err := ValidateEscalationEmail(req.Email); err != nil || again != req.Email {
			t.Fatal("email validation not idempotent")
		}
		if !utf8.ValidString(req.QuestionNote) || len(req.QuestionNote) > maxEscalationNoteBytes || strings.Contains(req.QuestionNote, "\x00") {
			t.Fatal("accepted note violates the record")
		}
		if again, err := ValidateEscalationNote(req.QuestionNote); err != nil || again != req.QuestionNote {
			t.Fatal("note normalization not idempotent")
		}
	})
}
