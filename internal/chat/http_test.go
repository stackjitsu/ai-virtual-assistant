package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"
)

type providerFunc func(context.Context, ComposeInput) (Composition, error)

func (f providerFunc) Compose(c context.Context, in ComposeInput) (Composition, error) {
	return f(c, in)
}

var peerCounter atomic.Int64

func perform(h http.Handler, method, path, capability, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.RemoteAddr = fmt.Sprintf("192.0.%d.%d:1234", peerCounter.Add(1)/250%250, peerCounter.Load()%250)
	if method == "POST" {
		r.Header.Set("Content-Type", "application/json")
	}
	if capability != "" {
		r.Header.Set("Authorization", "Bearer "+capability)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func testHandler(t *testing.T, s Store, p Provider) http.Handler {
	t.Helper()
	h, err := NewHandler(s, p, HTTPConfig{"http://127.0.0.1:8080", []string{"http://localhost:8080", "http://127.0.0.1:8080"}}, fstest.MapFS{"widget.html": &fstest.MapFile{Data: []byte("fixture assets")}, "sample-host.html": &fstest.MapFile{Data: []byte("fixture host")}})
	if err != nil {
		t.Fatal(err)
	}
	return h
}
func createHTTP(t *testing.T, h http.Handler, body string) (string, string) {
	t.Helper()
	w := perform(h, "POST", "/api/v1/conversations", "", body)
	if w.Code != 201 {
		t.Fatalf("create %d %s", w.Code, w.Body)
	}
	var result struct {
		ID         string    `json:"conversation_id"`
		Capability string    `json:"capability"`
		CreatedAt  time.Time `json:"created_at"`
	}
	if strictJSON(w.Body.Bytes(), &result) != nil || !idPattern.MatchString(result.ID) || !capPattern.MatchString(result.Capability) || result.CreatedAt.IsZero() {
		t.Fatal("create shape", w.Body)
	}
	return result.ID, result.Capability
}
func assertCode(t *testing.T, w *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	var result struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if w.Code != status || strictJSON(w.Body.Bytes(), &result) != nil || result.Error.Code != code || result.Error.Message == "" {
		t.Fatalf("want %d %s got %d %s", status, code, w.Code, w.Body)
	}
	if w.Header().Get("Content-Type") != "application/json" || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("API security headers", w.Header())
	}
	for _, bad := range []string{"private SQL", "fixture-secret", "stack trace"} {
		if strings.Contains(w.Body.String(), bad) {
			t.Fatal("unsafe error")
		}
	}
}
func postBody(req, text string) string {
	b, _ := json.Marshal(map[string]string{"request_id": req, "text": text})
	return string(b)
}

type observedStore struct {
	Store
	finds, knowledge, histories atomic.Int32
}

func (s *observedStore) FindTurn(ctx context.Context, id, req string) (Turn, error) {
	s.finds.Add(1)
	return s.Store.FindTurn(ctx, id, req)
}
func (s *observedStore) ApprovedKnowledge(ctx context.Context) ([]Knowledge, error) {
	s.knowledge.Add(1)
	return s.Store.ApprovedKnowledge(ctx)
}
func (s *observedStore) History(ctx context.Context, id string, n int, b *int64) (HistoryPage, error) {
	s.histories.Add(1)
	return s.Store.History(ctx, id, n, b)
}

func TestContractHTTPAuthorizationReplayIsolationAndOrdering(t *testing.T) {
	store := testStore(t)
	if err := store.SeedDemo(context.Background()); err != nil {
		t.Fatal(err)
	}
	observed := &observedStore{Store: store}
	var calls atomic.Int32
	p := providerFunc(func(ctx context.Context, in ComposeInput) (Composition, error) {
		calls.Add(1)
		if len(in.Knowledge) == 0 || in.Context.PagePath != "/help" {
			t.Error("missing retrieved knowledge/context")
		}
		return Composition{"AI-composed fictional sample answer", []string{"demo-start"}, false}, nil
	})
	h := testHandler(t, observed, p)
	id, capability := createHTTP(t, h, `{"context":{"page_path":"/help","locale":"en"}}`)
	id2, cap2 := createHTTP(t, h, `{}`)
	path := "/api/v1/conversations/" + id + "/messages"
	unauthorizedBody := ""
	for _, auth := range []string{"", strings.Repeat("f", 64), cap2, "bad"} {
		w := perform(h, "POST", path, auth, postBody("request_001", "start onboarding"))
		assertCode(t, w, 401, "unauthorized")
		if unauthorizedBody == "" {
			unauthorizedBody = w.Body.String()
		} else if unauthorizedBody != w.Body.String() {
			t.Fatal("auth existence leak")
		}
		assertCode(t, perform(h, "GET", path, auth, ""), 401, "unauthorized")
	}
	w := perform(h, "GET", "/api/v1/conversations/"+strings.Repeat("0", 32)+"/messages", capability, "")
	assertCode(t, w, 401, "unauthorized")
	if w.Body.String() != unauthorizedBody || calls.Load() != 0 || observed.finds.Load() != 0 || observed.knowledge.Load() != 0 || observed.histories.Load() != 0 {
		t.Fatal("unauthorized request reached private/paid work")
	}
	first := perform(h, "POST", path, capability, postBody("request_001", "  start onboarding \n"))
	if first.Code != 200 {
		t.Fatalf("send %d %s", first.Code, first.Body)
	}
	var turn turnEnvelope
	if json.Unmarshal(first.Body.Bytes(), &turn) != nil || turn.RequestID != "request_001" || len(turn.Messages) != 2 || turn.Messages[0].Seq != 1 || turn.Messages[1].Seq != 2 || turn.Messages[0].Text != "start onboarding" || turn.Messages[1].Status != "answered" || len(turn.Messages[1].Sources) != 1 {
		t.Fatal("completed envelope", first.Body)
	}
	replay := perform(h, "POST", path, capability, postBody("request_001", "start onboarding"))
	if replay.Code != 200 || first.Body.String() != replay.Body.String() || calls.Load() != 1 {
		t.Fatal("replay not exact or repeated provider")
	}
	assertCode(t, perform(h, "POST", path, capability, postBody("request_001", "different")), 409, "idempotency_conflict")
	assertCode(t, perform(h, "POST", path, cap2, postBody("request_001", "start onboarding")), 401, "unauthorized")
	if calls.Load() != 1 {
		t.Fatal("unauthorized replay incurred inference")
	}
	for n := 2; n <= 3; n++ {
		w = perform(h, "POST", path, capability, postBody(fmt.Sprintf("request_%03d", n), "start onboarding"))
		if w.Code != 200 {
			t.Fatal(w.Body)
		}
	}
	w = perform(h, "GET", path+"?limit=3&before=6", capability, "")
	var page HistoryPage
	if json.Unmarshal(w.Body.Bytes(), &page) != nil || w.Code != 200 || len(page.Messages) != 3 || page.Messages[0].Seq != 3 || page.Messages[2].Seq != 5 || page.NextBefore == nil || *page.NextBefore != 3 {
		t.Fatal("sequence cursor ordering", w.Body)
	}
	w = perform(h, "GET", "/api/v1/conversations/"+id2+"/messages", cap2, "")
	if w.Code != 200 || w.Body.String() != "{\"messages\":[],\"next_before\":null}\n" {
		t.Fatal("conversation isolation", w.Body)
	}
	if strings.Contains(w.Body.String(), "capability") {
		t.Fatal("capability returned again")
	}
}

func TestContractHTTPValidationErrorsAndOrigins(t *testing.T) {
	s := testStore(t)
	h := testHandler(t, s, providerFunc(func(context.Context, ComposeInput) (Composition, error) {
		t.Error("unexpected provider call")
		return Composition{}, ErrUnavailable
	}))
	id, capability := createHTTP(t, h, `{}`)
	path := "/api/v1/conversations/" + id + "/messages"
	for _, body := range []string{`{"context":{"other":"x"}}`, `{"context":{"page_path":"/a?token=x"}}`, `{"context":{"locale":"en_us"}}`, `{"context":null}`, `{"context":{"locale":null}}`, `[]`, `{} {}`, `{"context":{"page_path":12}}`, string([]byte{0xff})} {
		assertCode(t, perform(h, "POST", "/api/v1/conversations", "", body), 400, "invalid_request")
	}
	for _, body := range []string{`{"request_id":"short","text":"hi"}`, `{"request_id":"request_001","text":""}`, `{"request_id":"request_001","text":"x","extra":1}`, `{"request_id":"request_001","text":null}`, postBody("request_001", strings.Repeat("a", 2001)), postBody("request_001", "bad\x00value")} {
		assertCode(t, perform(h, "POST", path, capability, body), 400, "invalid_request")
	}
	for _, q := range []string{"?limit=0", "?limit=101", "?limit=2&limit=3", "?before=0", "?before=-2", "?other=1", "?limit=x", "?before=9223372036854775808", "?bad=%GG"} {
		assertCode(t, perform(h, "GET", path+q, capability, ""), 400, "invalid_request")
	}
	assertCode(t, perform(h, "GET", "/api/v1/conversations/bad/messages", capability, ""), 400, "invalid_request")
	assertCode(t, perform(h, "POST", "/api/v1/conversations", "", strings.Repeat("x", 2049)), 413, "request_too_large")
	assertCode(t, perform(h, "POST", path, capability, strings.Repeat("x", 16385)), 413, "request_too_large")
	for _, origin := range []string{"null", "https://evil.test", "http://127.0.0.1:8080.evil.test", "http://127.0.0.1:8081", "http://127.0.0.1:8080/"} {
		r := httptest.NewRequest("POST", "/api/v1/conversations", strings.NewReader(`{}`))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		assertCode(t, w, 403, "origin_not_allowed")
		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatal("CORS enabled")
		}
	}
	r := httptest.NewRequest("POST", "/api/v1/conversations", strings.NewReader(`{}`))
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	assertCode(t, w, 415, "unsupported_media_type")
	for _, route := range []string{"/api/v1/not-a-route", "/api/v1", "/api/unknown"} {
		assertCode(t, perform(h, "GET", route, "", ""), 404, "not_found")
	}
	assertCode(t, perform(h, "DELETE", path, capability, ""), 405, "method_not_allowed")
	assertCode(t, perform(h, "GET", path, capability, "body"), 400, "invalid_request")
	widget := perform(h, "GET", "/widget/widget.html", "", "")
	if widget.Code != 200 || widget.Header().Get("X-Frame-Options") != "" || !strings.Contains(widget.Header().Get("Content-Security-Policy"), "frame-ancestors http://localhost:8080 http://127.0.0.1:8080") || !strings.Contains(widget.Header().Get("Content-Security-Policy"), "connect-src 'self'") {
		t.Fatal("embedding CSP", widget.Header())
	}
	root := perform(h, "GET", "/", "", "")
	if root.Code != 302 || root.Header().Get("Location") != "/widget/sample-host.html" {
		t.Fatal("sample redirect")
	}
}

func TestContractHTTPClarificationWithdrawalAndNoHalfTurn(t *testing.T) {
	s := testStore(t)
	var calls atomic.Int32
	var mode atomic.Int32
	p := providerFunc(func(context.Context, ComposeInput) (Composition, error) {
		calls.Add(1)
		switch mode.Load() {
		case 0:
			return Composition{}, ErrUnavailable
		case 1:
			return Composition{"private invalid answer", []string{"unknown"}, false}, nil
		default:
			return Composition{"DO NOT EMIT THIS UNSUPPORTED MODEL PROSE", []string{}, true}, nil
		}
	})
	h := testHandler(t, s, p)
	id, capability := createHTTP(t, h, `{}`)
	path := "/api/v1/conversations/" + id + "/messages"
	w := perform(h, "POST", path, capability, postBody("request_001", "start onboarding"))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "needs_clarification") || calls.Load() != 0 {
		t.Fatal("no-doc clarification", w.Body)
	}
	if err := s.SeedDemo(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, n := range []int32{0, 1} {
		mode.Store(n)
		assertCode(t, perform(h, "POST", path, capability, postBody("request_002", "start onboarding")), 503, "provider_unavailable")
	}
	history, err := s.History(context.Background(), id, 100, nil)
	if err != nil || len(history.Messages) != 2 {
		t.Fatal("failed half-turn persisted", history, err)
	}
	mode.Store(2)
	w = perform(h, "POST", path, capability, postBody("request_002", "start onboarding"))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "needs_clarification") || strings.Contains(w.Body.String(), "DO NOT EMIT") {
		t.Fatal("unsupported output was not replaced", w.Body)
	}
	if _, err = s.db.Exec(`UPDATE knowledge SET approved=0`); err != nil {
		t.Fatal(err)
	}
	before := calls.Load()
	w = perform(h, "POST", path, capability, postBody("request_003", "start onboarding"))
	if w.Code != 200 || calls.Load() != before || !strings.Contains(w.Body.String(), "needs_clarification") {
		t.Fatal("withdrawn knowledge reached inference", w.Body)
	}
	// A real empty-key adapter must not invent a successful answer when knowledge exists.
	if err = s.SeedDemo(context.Background()); err != nil {
		t.Fatal(err)
	}
	h = testHandler(t, s, NewOpenAI("", "fixture-model"))
	assertCode(t, perform(h, "POST", path, capability, postBody("request_004", "start onboarding")), 503, "provider_unavailable")
	history, err = s.History(context.Background(), id, 100, nil)
	if err != nil || len(history.Messages) != 6 {
		t.Fatal("missing-key half-turn", err)
	}
}

func TestContractHTTPExactJSONNamesPresenceAndDuplicates(t *testing.T) {
	s := testStore(t)
	h := testHandler(t, s, providerFunc(func(context.Context, ComposeInput) (Composition, error) {
		t.Fatal("invalid request reached provider")
		return Composition{}, ErrUnavailable
	}))
	for _, body := range []string{
		`{"Context":{"page_path":"/help"}}`,
		`{"context":{"Page_Path":"/help"}}`,
		`{"context":{},"context":{}}`,
		`{"context":{"locale":"en","locale":"fr"}}`,
		`{"context":{"locale":"en","\u006cocale":"fr"}}`,
		`{"context":{"page_path":""}}`,
		`{"context":{"locale":""}}`,
	} {
		t.Run(body, func(t *testing.T) {
			assertCode(t, perform(h, "POST", "/api/v1/conversations", "", body), 400, "invalid_request")
		})
	}
	// All context fields remain optional; only a supplied value must be valid.
	for _, body := range []string{`{}`, `{"context":{}}`, `{"context":{"locale":"en"}}`, `{"context":{"page_path":"/help"}}`} {
		createHTTP(t, h, body)
	}
	id, capability := createHTTP(t, h, `{}`)
	path := "/api/v1/conversations/" + id + "/messages"
	for _, body := range []string{
		`{"Request_ID":"request_001","text":"start"}`,
		`{"request_id":"request_001","Text":"start"}`,
		`{"request_id":"request_001","text":"start","text":"different"}`,
		`{"request_id":"request_001","text":"start","\u0074ext":"start"}`,
		`{"request_id":"request_001"}`,
		`{"text":"start"}`,
	} {
		t.Run(body, func(t *testing.T) {
			assertCode(t, perform(h, "POST", path, capability, body), 400, "invalid_request")
		})
	}
	for _, media := range []string{"application/json; charset=iso-8859-1", "application/json; charset=utf-16"} {
		r := httptest.NewRequest("POST", "/api/v1/conversations", strings.NewReader(`{}`))
		r.Header.Set("Content-Type", media)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		assertCode(t, w, 415, "unsupported_media_type")
	}
	r := httptest.NewRequest("POST", "/api/v1/conversations", strings.NewReader(`{}`))
	r.Header.Set("Content-Type", "application/json; charset=UTF-8")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 201 {
		t.Fatal("UTF-8 JSON rejected", w.Body)
	}
	page, err := s.History(context.Background(), id, 100, nil)
	if err != nil || len(page.Messages) != 0 {
		t.Fatal("invalid JSON persisted a turn", err)
	}
}

func TestContractHTTPBusyFingerprintsAndReleasedSlots(t *testing.T) {
	s := testStore(t)
	if err := s.SeedDemo(context.Background()); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	var calls atomic.Int32
	p := providerFunc(func(ctx context.Context, in ComposeInput) (Composition, error) {
		calls.Add(1)
		select {
		case started <- struct{}{}:
		default:
		}
		select {
		case <-release:
		case <-ctx.Done():
			return Composition{}, ctx.Err()
		}
		return Composition{"Sample", []string{"demo-start"}, false}, nil
	})
	h := testHandler(t, s, p)
	id, capability := createHTTP(t, h, `{}`)
	path := "/api/v1/conversations/" + id + "/messages"
	done := make(chan *httptest.ResponseRecorder, 1)
	go func() { done <- perform(h, "POST", path, capability, postBody("request_001", "start onboarding")) }()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		close(release)
		t.Fatal("provider did not start")
	}
	assertCode(t, perform(h, "POST", path, capability, postBody("request_001", "different")), 409, "idempotency_conflict")
	w := perform(h, "POST", path, capability, postBody("request_001", "start onboarding"))
	assertCode(t, w, 409, "conversation_busy")
	if w.Header().Get("Retry-After") != "1" {
		t.Fatal("busy retry hint")
	}
	assertCode(t, perform(h, "POST", path, capability, postBody("request_002", "start onboarding")), 409, "conversation_busy")
	close(release)
	first := <-done
	if first.Code != 200 {
		t.Fatal(first.Body)
	}
	w = perform(h, "POST", path, capability, postBody("request_001", "start onboarding"))
	if w.Code != 200 || w.Body.String() != first.Body.String() || calls.Load() != 1 {
		t.Fatal("busy retry replay")
	}
	w = perform(h, "POST", path, capability, postBody("request_002", "start onboarding"))
	if w.Code != 200 || calls.Load() != 2 {
		t.Fatal("conversation lock entry leaked", w.Body)
	}
}

func TestContractHTTPGlobalInferenceBound(t *testing.T) {
	s := testStore(t)
	if err := s.SeedDemo(context.Background()); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{}, 8)
	release := make(chan struct{})
	var calls atomic.Int32
	h := testHandler(t, s, providerFunc(func(ctx context.Context, in ComposeInput) (Composition, error) {
		calls.Add(1)
		started <- struct{}{}
		select {
		case <-release:
		case <-ctx.Done():
			return Composition{}, ctx.Err()
		}
		return Composition{"Sample", []string{"demo-start"}, false}, nil
	}))
	results := make(chan *httptest.ResponseRecorder, 8)
	for n := 1; n <= 8; n++ {
		c := testConversation(t, s, fmt.Sprintf("%032x", n))
		go func() {
			results <- perform(h, "POST", "/api/v1/conversations/"+c.ID+"/messages", strings.Repeat("c", 64), postBody("request_001", "start onboarding"))
		}()
	}
	for range 8 {
		select {
		case <-started:
		case <-time.After(3 * time.Second):
			close(release)
			t.Fatal("eight concurrent conversations did not progress")
		}
	}
	c := testConversation(t, s, fmt.Sprintf("%032x", 9))
	assertCode(t, perform(h, "POST", "/api/v1/conversations/"+c.ID+"/messages", strings.Repeat("c", 64), postBody("request_001", "start onboarding")), 503, "service_busy")
	if calls.Load() != 8 {
		t.Fatal("global provider bound violated")
	}
	close(release)
	for range 8 {
		if w := <-results; w.Code != 200 {
			t.Fatal(w.Body)
		}
	}
}

func TestContractHTTPFullReplayAndRateLimits(t *testing.T) {
	s := testStore(t)
	c := testConversation(t, s, strings.Repeat("a", 32))
	ctx := context.Background()
	for n := int64(1); n <= 50; n++ {
		if _, err := s.CompleteTurn(ctx, testTurn(c, fmt.Sprintf("request_%03d", n), "start", n), n-1); err != nil {
			t.Fatal(err)
		}
	}
	h := testHandler(t, s, providerFunc(func(context.Context, ComposeInput) (Composition, error) {
		t.Error("full conversation invoked provider")
		return Composition{}, ErrUnavailable
	}))
	path := "/api/v1/conversations/" + c.ID + "/messages"
	capability := strings.Repeat("c", 64)
	assertCode(t, perform(h, "POST", path, capability, postBody("request_051", "start")), 409, "conversation_full")
	if w := perform(h, "POST", path, capability, postBody("request_001", "start")); w.Code != 200 {
		t.Fatal("full conversation replay rejected", w.Body)
	}
	for n := 0; n < 11; n++ {
		r := httptest.NewRequest("POST", "/api/v1/conversations", strings.NewReader(`{}`))
		r.RemoteAddr = "192.0.2.1:1234"
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Forwarded-For", fmt.Sprintf("10.0.0.%d", n))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if n < 10 && w.Code != 201 {
			t.Fatal("early create rate limit")
		}
		if n == 10 {
			assertCode(t, w, 429, "rate_limited")
			if w.Header().Get("Retry-After") != "60" {
				t.Fatal("rate retry hint")
			}
		}
	}
	for n := 0; n < 31; n++ {
		r := httptest.NewRequest("POST", path, strings.NewReader(postBody("request_001", "start")))
		r.RemoteAddr = "192.0.2.2:1234"
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", "Bearer wrong")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if n < 30 {
			assertCode(t, w, 401, "unauthorized")
		} else {
			assertCode(t, w, 429, "rate_limited")
		}
	}
	// Bounded registry and expiry without allocating unbounded peer mutexes.
	ss := &server{rates: map[string]rateWindow{}}
	for n := 0; n < 10005; n++ {
		r := httptest.NewRequest("POST", "/", nil)
		r.RemoteAddr = fmt.Sprintf("10.%d.%d.%d:1", n/65536, n/256%256, n%256)
		if !ss.rate(httptest.NewRecorder(), r, true) {
			t.Fatal("new peer limited")
		}
	}
	if len(ss.rates) != 10000 {
		t.Fatal("unbounded rate registry")
	}
	ss.rates["old"] = rateWindow{start: time.Now().Add(-2 * time.Minute), creates: 10}
	r := httptest.NewRequest("POST", "/", nil)
	r.RemoteAddr = "old:1"
	if !ss.rate(httptest.NewRecorder(), r, true) {
		t.Fatal("expired window retained")
	}
}
