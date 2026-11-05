package chat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

// Any attempted disabled-route storage operation fails the test's expected 403.
type forbiddenSupportStore struct{ Store }

func (s forbiddenSupportStore) Conversation(context.Context, string) (Conversation, error) {
	panic("unexpected auth lookup")
}
func (s forbiddenSupportStore) FindEscalation(context.Context, string, string, string) (Escalation, error) {
	panic("unexpected replay lookup")
}
func (s forbiddenSupportStore) CreateEscalation(context.Context, Escalation) (Escalation, error) {
	panic("unexpected write")
}

type unreadBody struct{}

func (unreadBody) Read([]byte) (int, error) { panic("disabled route read body") }
func (unreadBody) Close() error             { return nil }

func deploymentHandler(t *testing.T, store Store, demo bool, box EmailOutbox) http.Handler {
	t.Helper()
	h, err := NewHandlerWithOutbox(store, providerFunc(func(context.Context, ComposeInput) (Composition, error) {
		t.Error("unexpected provider call")
		return Composition{}, ErrUnavailable
	}), HTTPConfig{PublicDemo: demo, AssistantOrigin: "http://127.0.0.1:8080", ParentOrigins: []string{"http://127.0.0.1:8080"}}, fstest.MapFS{"widget.html": &fstest.MapFile{Data: []byte("fixture")}}, box)
	if err != nil {
		t.Fatal(err)
	}
	return h
}
func TestContractDeploymentPolicy(t *testing.T) {
	for _, demo := range []bool{false, true} {
		h := deploymentHandler(t, forbiddenSupportStore{}, demo, nil)
		w := perform(h, "GET", "/api/v1/deployment", "", "")
		want := `{"public_demo":false,"email_support_enabled":true}` + "\n"
		if demo {
			want = `{"public_demo":true,"email_support_enabled":false}` + "\n"
		}
		if w.Code != 200 || w.Body.String() != want || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("X-Content-Type-Options") != "nosniff" || w.Header().Get("Content-Type") != "application/json" {
			t.Fatalf("discovery: %d %s %v", w.Code, w.Body, w.Header())
		}
		assertCode(t, perform(h, "POST", "/api/v1/deployment", "", "{}"), 405, "method_not_allowed")
		assertCode(t, perform(h, "GET", "/api/v1/deployment?x=1", "", ""), 400, "invalid_request")
		r := httptest.NewRequest("GET", "/api/v1/deployment", nil)
		r.Header.Set("Origin", "https://wrong.example")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, r)
		assertCode(t, w, 403, "origin_not_allowed")
		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatal("CORS bypass")
		}
	}
}
func TestContractPublicDemoDisablesSupportBeforeIOAndReplay(t *testing.T) {
	store := testStore(t)
	box := &outboxSpy{}
	legacy := escalationHandler(t, store, box)
	id, capability := createHTTP(t, legacy, "{}")
	body := escalationBody("user@example.org", "Existing ticket")
	old := perform(legacy, "POST", escalationPath(id), capability, body)
	escalationTicket(t, old)
	demo := deploymentHandler(t, forbiddenSupportStore{store}, true, box)
	for _, tc := range []struct{ id, cap, body string }{
		{id, capability, body}, {id, "", body}, {id, "wrong", body}, {id, capability, "{"}, {id, capability, strings.Repeat("x", 20000)}, {"not-an-id", "", "null"},
	} {
		assertCode(t, perform(demo, "POST", escalationPath(tc.id), tc.cap, tc.body), 403, "email_support_disabled")
	}
	r := httptest.NewRequest("POST", escalationPath(id), nil)
	r.Body = unreadBody{}
	r.RemoteAddr = "203.0.113.2:1234"
	for i := 0; i < 11; i++ {
		w := httptest.NewRecorder()
		demo.ServeHTTP(w, r)
		if i < 10 {
			assertCode(t, w, 403, "email_support_disabled")
		} else {
			assertCode(t, w, 429, "rate_limited")
		}
	}
	r = httptest.NewRequest("POST", escalationPath(id), nil)
	r.Body = unreadBody{}
	r.Header.Set("Origin", "null")
	w := httptest.NewRecorder()
	demo.ServeHTTP(w, r)
	assertCode(t, w, 403, "origin_not_allowed")
	replay := perform(legacy, "POST", escalationPath(id), capability, body)
	if replay.Body.String() != old.Body.String() || box.delivered() != 1 {
		t.Fatal("disabled requests changed old ticket/outbox")
	}
	// Actual demo-mode chat and rating are still usable without any model call.
	demo = deploymentHandler(t, store, true, box)
	turn := perform(demo, "POST", "/api/v1/conversations/"+id+"/messages", capability, postBody("demo_req_12345678", "unmatchedxyz"))
	if turn.Code != 200 {
		t.Fatalf("chat %d %s", turn.Code, turn.Body)
	}
	rating := perform(demo, "POST", "/api/v1/conversations/"+id+"/rating", capability, `{"rating":5}`)
	if rating.Code != 200 {
		t.Fatalf("rating %d %s", rating.Code, rating.Body)
	}
}
