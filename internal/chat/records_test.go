package chat

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestValidationAndRetrieval(t *testing.T) {
	for _, s := range []string{"", " \t\n", "bad\x00value", "bad\x01value", strings.Repeat("a", 2001), strings.Repeat("界", 2001), string([]byte{0xff}), strings.Repeat(" ", 8001)} {
		if _, err := ValidateMessage(s); err == nil {
			t.Errorf("accepted invalid message %q", s[:min(len(s), 20)])
		}
	}
	if got, err := ValidateMessage(" \nhello\t世界 \n"); err != nil || got != "hello\t世界" {
		t.Fatalf("canonical message %q %v", got, err)
	}
	for _, c := range []HostContext{{"https://evil.test/", ""}, {"//evil.test", ""}, {"/help?secret=1", ""}, {"/a#b", ""}, {"/a\\b", ""}, {"/a%00", ""}, {"/a", "english"}, {"/a", "en-US-private-long-more"}} {
		if ValidateContext(c) == nil {
			t.Errorf("accepted context %+v", c)
		}
	}
	if ValidateContext(HostContext{"/help", "en-US"}) != nil {
		t.Fatal("safe context rejected")
	}
	if Fingerprint("hello", HostContext{}) == Fingerprint("hello", HostContext{Locale: "en"}) {
		t.Fatal("context omitted from fingerprint")
	}
	docs := []Knowledge{{"b", "Start", "", "start onboarding", true}, {"a", "Start", "", "start onboarding", true}, {"withdrawn", "Start", "", "start onboarding start start", false}, {"irrelevant", "Cooking", "", "baking", true}}
	got := Retrieve("How can you start onboarding?", docs)
	if len(got) != 2 || got[0].ID != "a" || got[1].ID != "b" {
		t.Fatalf("deterministic approved ranking %+v", got)
	}
	if len(Retrieve("the and please", docs)) != 0 {
		t.Fatal("stopwords counted")
	}
	big := strings.Repeat("start ", 1333)
	docs = []Knowledge{{"a", "Start", "", big, true}, {"b", "Start", "", big, true}, {"c", "Start", "", "start", true}, {"d", "Start", "", "start", true}, {"e", "Start", "", "start", true}}
	got = Retrieve("start", docs)
	size := 0
	for _, d := range got {
		size += len(d.Text)
	}
	if len(got) != 3 || size > 12000 || got[1].ID != "c" {
		t.Fatalf("whole-document budget %+v %d", got, size)
	}
	if validateKnowledge([]Knowledge{{"bad", "Title", "http://bad.test", "text", true}}) == nil {
		t.Fatal("unsafe source accepted")
	}
}

func TestHistoryAndCompositionBounds(t *testing.T) {
	messages := []Message{}
	for i := 0; i < 15; i++ {
		messages = append(messages, Message{Seq: int64(i + 1), Text: strings.Repeat("a", 1500)})
	}
	got := BoundHistory(messages)
	if len(got) != 8 || got[0].Seq != 8 || got[7].Seq != 15 {
		t.Fatalf("history byte budget %+v", got)
	}
	for i := range messages {
		messages[i].Text = "short"
	}
	got = BoundHistory(messages)
	if len(got) != 10 || got[0].Seq != 6 {
		t.Fatal("history message bound")
	}
	docs := []Knowledge{{"demo-start", "Title", "https://example.org", "text", true}}
	for _, c := range []Composition{{"a", []string{}, false}, {"a", []string{"unknown"}, false}, {"a", []string{"demo-start", "demo-start"}, false}, {"", []string{"demo-start"}, false}, {strings.Repeat("a", 4001), []string{"demo-start"}, false}} {
		if _, err := validateComposition(c, docs); err == nil {
			t.Fatalf("invalid composition accepted %+v", c)
		}
	}
	if _, err := validateComposition(Composition{"No support", []string{}, true}, docs); err != nil {
		t.Fatal(err)
	}
}

func TestProvisionalFAQSourceURLValidation(t *testing.T) {
	valid := Source{"demo-start", "Jitihada applications", "https://example.org/jitihada/help/applications"}
	if !validSource(valid) {
		t.Fatal("reserved HTTPS citation rejected")
	}
	for _, raw := range []string{"http://example.org/jitihada/help/applications", "https://user@example.org/jitihada/help/applications", "https://example.org/\nsecret"} {
		valid.URL = raw
		if validSource(valid) {
			t.Fatal("unsafe citation accepted", raw)
		}
	}
}

// Property: accepted messages are canonical, bounded, valid UTF-8 and stable
// under validation; arbitrary malformed/oversized input never panics.
func FuzzValidateMessage(f *testing.F) {
	for _, s := range []string{"hello", "\tHello 世界\n", "\x00", string([]byte{0xff}), strings.Repeat("a", 8001)} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		v, err := ValidateMessage(s)
		if err != nil {
			return
		}
		if !utf8.ValidString(v) || len(v) > 8000 || utf8.RuneCountInString(v) > 2000 || strings.TrimSpace(v) != v || !validText(v, 2000, 8000) {
			t.Fatal("accepted message violates record")
		}
		again, err := ValidateMessage(v)
		if err != nil || again != v {
			t.Fatal("normalization not idempotent")
		}
		b, _ := json.Marshal(v)
		var back string
		if json.Unmarshal(b, &back) != nil || back != v {
			t.Fatal("JSON roundtrip changed canonical text")
		}
	})
}
