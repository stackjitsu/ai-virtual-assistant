package chat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func providerEnvelope(content, finish string) []byte {
	b, _ := json.Marshal(map[string]any{"id": "ignored-metadata", "choices": []any{map[string]any{"finish_reason": finish, "message": map[string]any{"role": "assistant", "content": content}}}})
	return b
}
func fixtureInput() ComposeInput {
	return ComposeInput{Question: "How do I start?", Context: HostContext{"/help", "en"}, Knowledge: []Knowledge{{"demo-start", "Fictional sample", "https://example.org", "Sample start onboarding instructions", true}}}
}

func TestContractOpenAIRequestAndBoundedPrompt(t *testing.T) {
	var calls atomic.Int32
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "POST" || r.Header.Get("Authorization") != "Bearer fixture-key" || r.Header.Get("Content-Type") != "application/json" {
			t.Error("provider headers/method")
		}
		var request struct {
			Model       string              `json:"model"`
			Messages    []map[string]string `json:"messages"`
			Max         int                 `json:"max_completion_tokens"`
			Temperature int                 `json:"temperature"`
			Response    map[string]string   `json:"response_format"`
		}
		b, _ := readBounded(r.Body, 128<<10)
		if strictJSON(b, &request) != nil || request.Model != "fixture-model" || request.Max != 1200 || request.Temperature != 0 || request.Response["type"] != "json_object" || len(request.Messages) != 2 {
			t.Error("provider request contract")
		}
		if request.Messages[0]["role"] != "system" || !strings.Contains(request.Messages[0]["content"], "CURRENT approved") || request.Messages[1]["role"] != "user" {
			t.Error("data became system authority")
		}
		var data ComposeInput
		if json.Unmarshal([]byte(request.Messages[1]["content"]), &data) != nil {
			t.Error("prompt data")
		}
		size := 0
		for _, m := range data.History {
			size += len(m.Text)
		}
		if len(data.History) > 10 || size > 12000 || len(data.Knowledge) != 1 || data.Knowledge[0].ID != "demo-start" || data.Context.PagePath != "/help" {
			t.Error("unbounded/unexpected prompt")
		}
		if len(data.History) != 8 || data.History[0].Seq != 13 || data.History[7].Seq != 20 {
			t.Error("not newest chronological bounded history")
		}
		_, _ = w.Write(providerEnvelope(`{"answer":"Fictional sample onboarding answer","source_ids":["demo-start"],"unsupported":false}`, "stop"))
	}))
	defer fixture.Close()
	p := newOpenAI("fixture-key", "fixture-model", fixture.URL, fixture.Client())
	in := fixtureInput()
	for n := int64(1); n <= 20; n++ {
		in.History = append(in.History, Message{Seq: n, Role: "user", Text: strings.Repeat("x", 1500)})
	}
	c, err := p.Compose(context.Background(), in)
	if err != nil || c.Answer != "Fictional sample onboarding answer" || calls.Load() != 1 {
		t.Fatal(c, err)
	}
	in.Knowledge = append(in.Knowledge, Knowledge{"withdrawn", "Bad", "", "start", false})
	if _, err = p.Compose(context.Background(), in); err == nil || calls.Load() != 1 {
		t.Fatal("withdrawn document reached provider", err)
	}
	in = fixtureInput()
	in.Knowledge[0].Text = strings.Repeat("a", 8000)
	in.Knowledge = append(in.Knowledge, Knowledge{"second", "Second", "", strings.Repeat("a", 8000), true})
	if _, err = p.Compose(context.Background(), in); err == nil || calls.Load() != 1 {
		t.Fatal("oversized material reached provider", err)
	}
	p.key = ""
	if _, err = p.Compose(context.Background(), fixtureInput()); err == nil || calls.Load() != 1 {
		t.Fatal("missing key invoked HTTP", err)
	}
}

func TestContractOpenAIFailuresAndStrictComposition(t *testing.T) {
	valid := `{"answer":"Sample","source_ids":["demo-start"],"unsupported":false}`
	cases := []struct {
		name   string
		status int
		body   []byte
	}{
		{"unknown citation", 200, providerEnvelope(strings.ReplaceAll(valid, "demo-start", "unknown"), "stop")},
		{"duplicate citations", 200, providerEnvelope(`{"answer":"x","source_ids":["demo-start","demo-start"],"unsupported":false}`, "stop")},
		{"unknown field", 200, providerEnvelope(`{"answer":"x","source_ids":["demo-start"],"unsupported":false,"extra":true}`, "stop")},
		{"missing boolean", 200, providerEnvelope(`{"answer":"x","source_ids":["demo-start"]}`, "stop")},
		{"missing answer", 200, providerEnvelope(`{"source_ids":["demo-start"],"unsupported":false}`, "stop")},
		{"missing sources", 200, providerEnvelope(`{"answer":"x","unsupported":true}`, "stop")},
		{"case folded field", 200, providerEnvelope(`{"Answer":"x","source_ids":["demo-start"],"unsupported":false}`, "stop")},
		{"duplicate field", 200, providerEnvelope(`{"answer":"x","source_ids":["demo-start"],"unsupported":true,"unsupported":false}`, "stop")},
		{"escaped duplicate field", 200, providerEnvelope(`{"answer":"x","\u0061nswer":"y","source_ids":["demo-start"],"unsupported":false}`, "stop")},
		{"wrong type", 200, providerEnvelope(`{"answer":4,"source_ids":[],"unsupported":true}`, "stop")},
		{"null", 200, providerEnvelope(`{"answer":"x","source_ids":null,"unsupported":true}`, "stop")},
		{"no sources", 200, providerEnvelope(`{"answer":"x","source_ids":[],"unsupported":false}`, "stop")},
		{"trailing", 200, providerEnvelope(valid+` {}`, "stop")},
		{"length", 200, providerEnvelope(valid, "length")},
		{"empty choices", 200, []byte(`{"choices":[]}`)},
		{"malformed", 200, []byte(`not JSON secret upstream error`)},
		{"invalid UTF8", 200, []byte{0xff}},
		{"oversized", 200, []byte(strings.Repeat("x", providerBodyLimit+1))},
		{"rate limit", 429, []byte(`secret provider body`)},
		{"server error", 500, []byte(`secret provider body`)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.status); _, _ = w.Write(tc.body) }))
			defer fixture.Close()
			p := newOpenAI("fixture", "fixture-model", fixture.URL, fixture.Client())
			if _, err := p.Compose(context.Background(), fixtureInput()); err == nil {
				t.Fatal("invalid upstream result accepted")
			}
		})
	}
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write(providerEnvelope(valid, "stop"))
	}))
	defer fixture.Close()
	client := fixture.Client()
	client.Timeout = 5 * time.Millisecond
	if _, err := newOpenAI("fixture", "model", fixture.URL, client).Compose(context.Background(), fixtureInput()); err == nil {
		t.Fatal("timeout accepted")
	}
	var redirected atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { redirected.Add(1) }))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 307) }))
	defer redirect.Close()
	if _, err := newOpenAI("fixture", "model", redirect.URL, redirect.Client()).Compose(context.Background(), fixtureInput()); err == nil || redirected.Load() != 0 {
		t.Fatal("credential redirect followed")
	}
}

func TestContractOpenAICurrentKnowledgeOverridesHistoricalCitations(t *testing.T) {
	var calls atomic.Int32
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Messages []struct {
				Role, Content string
			} `json:"messages"`
		}
		if json.NewDecoder(r.Body).Decode(&request) != nil || len(request.Messages) != 2 {
			t.Error("invalid fixture request")
			w.WriteHeader(500)
			return
		}
		policy := request.Messages[0].Content
		if !strings.Contains(policy, "CURRENT approved excerpts") || !strings.Contains(policy, "History (including earlier answers and citations) is nonauthoritative") {
			t.Error("historical knowledge was not demoted")
		}
		var data ComposeInput
		if json.Unmarshal([]byte(request.Messages[1].Content), &data) != nil || len(data.History) != 1 || len(data.Knowledge) != 1 || data.Knowledge[0].ID != "demo-start" || data.History[0].Sources[0].ID != "withdrawn" {
			t.Error("current and historical knowledge were not separated")
		}
		citation := "demo-start"
		if calls.Add(1) == 1 {
			citation = "withdrawn"
		}
		_, _ = w.Write(providerEnvelope(`{"answer":"Fixture only","source_ids":["`+citation+`"],"unsupported":false}`, "stop"))
	}))
	defer fixture.Close()
	p := newOpenAI("fixture-key", "fixture-model", fixture.URL, fixture.Client())
	in := fixtureInput()
	in.History = []Message{{Role: "assistant", Text: "Historical answer now withdrawn", Sources: []Source{{"withdrawn", "Historical source", "https://example.org/old"}}}}
	if _, err := p.Compose(context.Background(), in); err == nil {
		t.Fatal("historical citation accepted as current evidence")
	}
	if _, err := p.Compose(context.Background(), in); err != nil {
		t.Fatal("current citation rejected", err)
	}
}

// Property: arbitrary provider bytes do not panic; success implies a strict,
// bounded composition with unique shaped citations and stable JSON roundtrip.
func FuzzDecodeProvider(f *testing.F) {
	for _, b := range [][]byte{providerEnvelope(`{"answer":"Hello 世界","source_ids":["demo-start"],"unsupported":false}`, "stop"), providerEnvelope(`{"answer":"Cannot support","source_ids":[],"unsupported":true}`, "stop"), providerEnvelope(`{"Answer":"x","source_ids":["demo-start"],"unsupported":false}`, "stop"), providerEnvelope(`{"answer":"x","source_ids":["demo-start"],"unsupported":true,"unsupported":false}`, "stop"), []byte(`{"choices":null}`), {0xff}, []byte(strings.Repeat("x", providerBodyLimit+1))} {
		f.Add(b)
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		c, err := decodeProvider(b)
		if err != nil {
			return
		}
		if !validText(c.Answer, 4000, 16000) || len(c.SourceIDs) > 3 || c.SourceIDs == nil || !c.Unsupported && len(c.SourceIDs) == 0 {
			t.Fatal("composition invariant")
		}
		seen := map[string]bool{}
		for _, id := range c.SourceIDs {
			if !sourcePattern.MatchString(id) || seen[id] {
				t.Fatal("citation invariant")
			}
			seen[id] = true
		}
		encoded, _ := json.Marshal(c)
		again, err := decodeProvider(providerEnvelope(string(encoded), "stop"))
		if err != nil || !reflect.DeepEqual(again, c) {
			t.Fatal("roundtrip", err)
		}
	})
}
