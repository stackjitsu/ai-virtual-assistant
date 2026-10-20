package chat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
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

// The configured endpoint override must receive the exact provider POST:
// same method, bearer of the key, content type, bounded prompt and strict
// composition decode, at the configured complete URL including its path.
func TestContractOpenAIConfiguredEndpoint(t *testing.T) {
	const configuredPath = "/v1beta/openai/chat/completions"
	var calls atomic.Int32
	var mode atomic.Int32 // 1 valid envelope; 2 invalid composition
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != configuredPath {
			http.NotFound(w, r)
			return
		}
		if r.Method != "POST" || r.Header.Get("Authorization") != "Bearer endpoint-key" || r.Header.Get("Content-Type") != "application/json" {
			t.Error("configured endpoint method/headers")
		}
		b, err := readBounded(r.Body, providerBodyLimit)
		if err != nil || len(b) == 0 || len(b) >= providerBodyLimit {
			t.Error("unbounded/missing provider request body")
		}
		var request struct {
			Model       string              `json:"model"`
			Messages    []map[string]string `json:"messages"`
			Max         int                 `json:"max_completion_tokens"`
			Temperature int                 `json:"temperature"`
			Response    map[string]string   `json:"response_format"`
		}
		if strictJSON(b, &request) != nil || request.Model != "endpoint-model" || request.Max != 1200 || request.Temperature != 0 || request.Response["type"] != "json_object" || len(request.Messages) != 2 {
			t.Error("override changed the provider request discipline")
		}
		var data ComposeInput
		if json.Unmarshal([]byte(request.Messages[1]["content"]), &data) != nil || len(data.Knowledge) != 1 || data.Knowledge[0].ID != "demo-start" || len(data.History) > 10 {
			t.Error("prompt data")
		}
		if mode.Load() == 2 {
			_, _ = w.Write(providerEnvelope(`{"answer":"x","source_ids":["demo-start"],"unsupported":false,"extra":true}`, "stop"))
			return
		}
		_, _ = w.Write(providerEnvelope(`{"answer":"Configured endpoint answer","source_ids":["demo-start"],"unsupported":false}`, "stop"))
	}))
	defer fixture.Close()
	p := NewOpenAIEndpoint("endpoint-key", "endpoint-model", fixture.URL+configuredPath)
	c, err := p.Compose(context.Background(), fixtureInput())
	if err != nil || c.Answer != "Configured endpoint answer" || calls.Load() != 1 {
		t.Fatal(c, err)
	}
	mode.Store(2)
	if _, err = p.Compose(context.Background(), fixtureInput()); err == nil || calls.Load() != 2 {
		t.Fatal("override bypassed the strict composition decode", err)
	}
	p.key = ""
	if _, err = p.Compose(context.Background(), fixtureInput()); err == nil || calls.Load() != 2 {
		t.Fatal("missing key invoked HTTP", err)
	}
}

// NewOpenAI must keep its fixed endpoint and shape, and constructing the
// provider with that same endpoint must yield identical shared fields.
func TestOpenAIDefaultEndpointEquivalence(t *testing.T) {
	fixed := NewOpenAI("same-key", "same-model")
	seam := NewOpenAIEndpoint("same-key", "same-model", providerEndpoint)
	if fixed.endpoint != providerEndpoint || fixed.endpoint != seam.endpoint || fixed.key != seam.key || fixed.model != seam.model {
		t.Fatal("configured constructor diverged from the fixed endpoint constructor")
	}
	for _, p := range []*OpenAI{fixed, seam} {
		if p.client.Timeout != 15*time.Second {
			t.Fatal("endpoint constructor lost the 15s provider timeout")
		}
		if err := p.client.CheckRedirect(nil, nil); err != http.ErrUseLastResponse {
			t.Fatal("endpoint constructor dropped the no-redirect rule", err)
		}
	}
}

func TestValidateOpenAIEndpoint(t *testing.T) {
	cases := []struct {
		name     string
		raw      string
		endpoint string
		ok       bool
	}{
		{"missing", "", "", true},
		{"blank falls back", " \t\n", "", true},
		{"gemini compatible endpoint", "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions", "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions", true},
		{"default host https", "https://api.openai.com/v1/chat/completions", "https://api.openai.com/v1/chat/completions", true},
		{"https loopback", "https://127.0.0.1:8443/v1/chat/completions", "https://127.0.0.1:8443/v1/chat/completions", true},
		{"http loopback ip", "http://127.0.0.1:8080/v1/chat/completions", "http://127.0.0.1:8080/v1/chat/completions", true},
		{"http loopback localhost", "http://localhost:9000/chat/completions", "http://localhost:9000/chat/completions", true},
		{"case folded loopback host", "http://LOCALHOST:9000/chat/completions", "http://LOCALHOST:9000/chat/completions", true},
		{"uppercase scheme normalized by parse", "HTTPS://example.org/v1/chat/completions", "HTTPS://example.org/v1/chat/completions", true},
		{"trimmed value", " https://example.org/v1/chat/completions\t", "https://example.org/v1/chat/completions", true},
		{"relative url", "example.org/v1/chat/completions", "", false},
		{"scheme relative", "//example.org/v1/chat/completions", "", false},
		{"http elsewhere rejected", "http://example.org/v1/chat/completions", "", false},
		{"http private network rejected", "http://192.168.1.10:8080/v1/chat/completions", "", false},
		{"http ipv6 rejected", "http://[::1]:8080/v1/chat/completions", "", false},
		{"userinfo rejected", "https://user:pass@example.org/v1/chat/completions", "", false},
		{"bare user rejected", "https://user@example.org/v1/chat/completions", "", false},
		{"fragment rejected", "https://example.org/v1/chat/completions#results", "", false},
		{"query rejected", "https://example.org/v1/chat/completions?api=1", "", false},
		{"missing host", "https://", "", false},
		{"port only host", "https://:8080/v1/chat/completions", "", false},
		{"wrong scheme", "ftp://example.org/v1/chat/completions", "", false},
		{"opaque value", "junk:opaque", "", false},
		{"inner whitespace", "https://exa mple.org/v1/chat/completions", "", false},
		{"trailing garbage", "https://example.org/v1/chat/completions trailing", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			endpoint, err := ValidateOpenAIEndpoint(tc.raw)
			if tc.ok != (err == nil) || endpoint != tc.endpoint {
				t.Fatalf("raw=%q endpoint=%q err=%v", tc.raw, endpoint, err)
			}
			if err == nil && endpoint != "" {
				again, aerr := ValidateOpenAIEndpoint(endpoint)
				if aerr != nil || again != endpoint {
					t.Fatal("accepted endpoint is not stably valid")
				}
			}
		})
	}
}

// Property: endpoint validation never panics; a blank value falls back to the
// default endpoint, and an accepted override is trimmed, absolute, https
// unless loopback, free of userinfo/fragment/query, and stably valid.
func FuzzValidateOpenAIEndpoint(f *testing.F) {
	for _, raw := range []string{
		"https://generativelanguage.googleapis.com/v1beta/openai/chat/completions",
		"http://127.0.0.1:8080/v1/chat/completions",
		" https://example.org/v1/chat/completions ",
		"   ", "//example.org", "https://u:p@h/x?a=b#c", "https://", "example.org", "ftp://h", "junk:opaque",
	} {
		f.Add(raw)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		endpoint, err := ValidateOpenAIEndpoint(raw)
		if err != nil {
			return
		}
		if endpoint == "" {
			if strings.TrimSpace(raw) != "" {
				t.Fatal("nonblank raw fell back to the default endpoint")
			}
			return
		}
		if endpoint != strings.TrimSpace(raw) {
			t.Fatal("accepted endpoint kept surrounding whitespace")
		}
		if strings.ContainsFunc(endpoint, func(r rune) bool { return r <= ' ' }) {
			t.Fatal("accepted endpoint kept an ASCII space/control character", endpoint)
		}
		u, uerr := url.Parse(endpoint)
		if uerr != nil || !u.IsAbs() || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" ||
			(u.Scheme != "https" && !(u.Scheme == "http" && loopbackEndpointHost(u.Hostname()))) {
			t.Fatal("accepted override violated the endpoint invariants", endpoint)
		}
		again, aerr := ValidateOpenAIEndpoint(endpoint)
		if aerr != nil || again != endpoint {
			t.Fatal("accepted endpoint is not stably valid")
		}
	})
}
