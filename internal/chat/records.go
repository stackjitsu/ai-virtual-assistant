// Package chat owns the support-chat records, pure transformations and I/O adapters.
package chat

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	ErrNotFound            = errors.New("not found")
	ErrConflict            = errors.New("stale turn")
	ErrIdempotencyConflict = errors.New("idempotency conflict")
	ErrFull                = errors.New("conversation full")
	ErrUnavailable         = errors.New("unavailable")
	ErrInvalid             = errors.New("invalid record")
)

type HostContext struct {
	PagePath string `json:"page_path,omitempty"`
	Locale   string `json:"locale,omitempty"`
}
type Source struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
}
type Knowledge struct {
	ID, Title, URL, Text string
	Approved             bool
}
type Conversation struct {
	ID             string
	CapabilityHash [32]byte
	Context        HostContext
	CreatedAt      time.Time
}
type Message struct {
	Seq       int64     `json:"seq"`
	RequestID string    `json:"request_id"`
	Role      string    `json:"role"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"`
	Sources   []Source  `json:"sources"`
}
type Turn struct {
	ConversationID, RequestID string
	Fingerprint               [32]byte
	Number                    int64
	User, Assistant           Message
}
type HistoryPage struct {
	Messages   []Message `json:"messages"`
	NextBefore *int64    `json:"next_before"`
}
type Store interface {
	CreateConversation(context.Context, Conversation) error
	Conversation(context.Context, string) (Conversation, error)
	FindTurn(context.Context, string, string) (Turn, error)
	History(context.Context, string, int, *int64) (HistoryPage, error)
	CompleteTurn(context.Context, Turn, int64) (Turn, error)
	ApprovedKnowledge(context.Context) ([]Knowledge, error)
	// Escalation ticket records append immutably to the conversation record
	// set; they share the capability authorization seam and bounded lookups.
	CreateEscalation(context.Context, Escalation) (Escalation, error)
	FindEscalation(context.Context, string, string, string) (Escalation, error)
	EscalationCount(context.Context, string) (int, error)
}
type ComposeInput struct {
	Question  string      `json:"question"`
	Context   HostContext `json:"context"`
	History   []Message   `json:"history"`
	Knowledge []Knowledge `json:"approved_excerpts"`
}
type Composition struct {
	Answer      string   `json:"answer"`
	SourceIDs   []string `json:"source_ids"`
	Unsupported bool     `json:"unsupported"`
}
type Provider interface {
	Compose(context.Context, ComposeInput) (Composition, error)
}

var (
	idPattern      = regexp.MustCompile(`^[0-9a-f]{32}$`)
	capPattern     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	requestPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)
	sourcePattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
	localePattern  = regexp.MustCompile(`^[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8}){0,2}$`)
)

func validText(s string, runes, bytes int) bool {
	if !utf8.ValidString(s) || len(s) > bytes || utf8.RuneCountInString(s) > runes || strings.TrimSpace(s) == "" {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return false
		}
	}
	return true
}

// ValidateMessage returns the canonical text used for both inference and fingerprints.
func ValidateMessage(s string) (string, error) {
	// Check byte bound before allocating a normalized copy.
	if len(s) > 8000 || !utf8.ValidString(s) {
		return "", ErrInvalid
	}
	s = strings.TrimSpace(s)
	if !validText(s, 2000, 8000) {
		return "", ErrInvalid
	}
	return s, nil
}

func ValidateContext(c HostContext) error {
	if c.Locale != "" && (len(c.Locale) > 24 || !localePattern.MatchString(c.Locale)) {
		return ErrInvalid
	}
	p := c.PagePath
	if p != "" {
		if !utf8.ValidString(p) || len(p) > 256 || !strings.HasPrefix(p, "/") || strings.ContainsAny(p, "?#\\") || strings.Contains(p, "//") {
			return ErrInvalid
		}
		for _, r := range p {
			if unicode.IsControl(r) {
				return ErrInvalid
			}
		}
		// Encoded query delimiters are harmless path data; encoded controls/backslashes
		// are rejected as well so the host and server have the same safe context.
		decoded, err := url.PathUnescape(p)
		if err != nil || strings.ContainsAny(decoded, "\\\x00") {
			return ErrInvalid
		}
		for _, r := range decoded {
			if unicode.IsControl(r) {
				return ErrInvalid
			}
		}
	}
	return nil
}

func Fingerprint(text string, c HostContext) [32]byte {
	b, _ := json.Marshal(struct {
		Text    string
		Context HostContext
	}{text, c})
	return sha256.Sum256(b)
}

func validSource(s Source) bool {
	if !sourcePattern.MatchString(s.ID) || !validText(s.Title, 160, 640) {
		return false
	}
	if s.URL == "" {
		return true
	}
	if len(s.URL) > 2048 || !utf8.ValidString(s.URL) {
		return false
	}
	for _, r := range s.URL {
		if unicode.IsControl(r) {
			return false
		}
	}
	u, err := url.Parse(s.URL)
	return err == nil && u.Scheme == "https" && u.Host != "" && u.User == nil
}

func validateKnowledge(docs []Knowledge) error {
	if len(docs) > 100 {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for _, d := range docs {
		if !validSource(Source{d.ID, d.Title, d.URL}) || !validText(d.Text, 8000, 8000) || seen[d.ID] {
			return ErrInvalid
		}
		seen[d.ID] = true
	}
	return nil
}

var stopwords = map[string]bool{"the": true, "and": true, "for": true, "with": true, "that": true, "this": true, "how": true, "what": true, "can": true, "you": true, "your": true, "are": true, "from": true, "have": true, "does": true, "please": true}

func tokens(s string) map[string]bool {
	out := map[string]bool{}
	for _, t := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		if utf8.RuneCountInString(t) >= 3 && !stopwords[t] {
			out[t] = true
		}
	}
	return out
}

// Retrieve ranks only currently approved complete documents; context never alters approval.
func Retrieve(question string, docs []Knowledge) []Knowledge {
	q := tokens(question)
	type scored struct {
		doc   Knowledge
		score int
	}
	var ranked []scored
	for _, d := range docs {
		if !d.Approved {
			continue
		}
		score := 0
		for t := range tokens(d.Title + " " + d.Text) {
			if q[t] {
				score++
			}
		}
		if score > 0 {
			ranked = append(ranked, scored{d, score})
		}
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score == ranked[j].score {
			return ranked[i].doc.ID < ranked[j].doc.ID
		}
		return ranked[i].score > ranked[j].score
	})
	out := []Knowledge{}
	size := 0
	for _, s := range ranked {
		if len(out) == 3 {
			break
		}
		if size+len(s.doc.Text) > 12000 {
			continue
		}
		out = append(out, s.doc)
		size += len(s.doc.Text)
	}
	return out
}

func BoundHistory(messages []Message) []Message {
	out := []Message{}
	size := 0
	for i := len(messages) - 1; i >= 0 && len(out) < 10; i-- {
		if size+len(messages[i].Text) > 12000 {
			break
		}
		out = append(out, messages[i])
		size += len(messages[i].Text)
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// strictJSON uses exact JSON field names and rejects duplicates, null, invalid
// UTF-8 and trailing JSON. encoding/json alone accepts case-folded names and
// silently overwrites duplicate keys, even with DisallowUnknownFields.
func strictJSON(b []byte, dst any) error {
	if !validJSON(b) {
		return ErrInvalid
	}
	d := json.NewDecoder(strings.NewReader(string(b)))
	if checkJSONShape(d, reflect.TypeOf(dst)) != nil {
		return ErrInvalid
	}
	if _, err := d.Token(); err != io.EOF {
		return ErrInvalid
	}
	if json.Unmarshal(b, dst) != nil {
		return ErrInvalid
	}
	return nil
}

// The owned wire records use named struct fields, string-keyed maps and slices;
// types/requiredness remain enforced by decoding and the record validators.
func checkJSONShape(d *json.Decoder, typ reflect.Type) error {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	token, err := d.Token()
	if err != nil || token == nil {
		return ErrInvalid
	}
	switch token {
	case json.Delim('{'):
		seen := map[string]bool{}
		for d.More() {
			keyToken, err := d.Token()
			key, ok := keyToken.(string)
			if err != nil || !ok || seen[key] {
				return ErrInvalid
			}
			seen[key] = true
			var field reflect.Type
			switch typ.Kind() {
			case reflect.Struct:
				for i := 0; i < typ.NumField(); i++ {
					f := typ.Field(i)
					name := strings.Split(f.Tag.Get("json"), ",")[0]
					if name == "" {
						name = f.Name
					}
					if f.IsExported() && name != "-" && name == key {
						field = f.Type
						break
					}
				}
			case reflect.Map:
				if typ.Key().Kind() == reflect.String {
					field = typ.Elem()
				}
			}
			if field == nil || checkJSONShape(d, field) != nil {
				return ErrInvalid
			}
		}
		if end, err := d.Token(); err != nil || end != json.Delim('}') {
			return ErrInvalid
		}
	case json.Delim('['):
		if typ.Kind() != reflect.Slice && typ.Kind() != reflect.Array {
			return ErrInvalid
		}
		for d.More() {
			if checkJSONShape(d, typ.Elem()) != nil {
				return ErrInvalid
			}
		}
		if end, err := d.Token(); err != nil || end != json.Delim(']') {
			return ErrInvalid
		}
	}
	return nil
}

func validateComposition(c Composition, docs []Knowledge) ([]Source, error) {
	if !validText(c.Answer, 4000, 16000) || c.SourceIDs == nil || len(c.SourceIDs) > 3 {
		return nil, ErrInvalid
	}
	known := map[string]Source{}
	for _, d := range docs {
		if d.Approved {
			known[d.ID] = Source{d.ID, d.Title, d.URL}
		}
	}
	out := []Source{}
	seen := map[string]bool{}
	for _, id := range c.SourceIDs {
		s, ok := known[id]
		if !ok || seen[id] || !validSource(s) {
			return nil, ErrInvalid
		}
		seen[id] = true
		out = append(out, s)
	}
	if !c.Unsupported && len(out) == 0 {
		return nil, ErrInvalid
	}
	return out, nil
}

func validateTurn(t Turn, previous int64) error {
	if !idPattern.MatchString(t.ConversationID) || !requestPattern.MatchString(t.RequestID) || previous < 0 || previous >= 50 || t.Number != previous+1 {
		return ErrInvalid
	}
	u, a := t.User, t.Assistant
	text, err := ValidateMessage(u.Text)
	if err != nil || text != u.Text || u.Role != "user" || a.Role != "assistant" || u.RequestID != t.RequestID || a.RequestID != t.RequestID || u.Seq != 2*t.Number-1 || a.Seq != 2*t.Number || u.Status != "submitted" || len(u.Sources) != 0 || !validText(a.Text, 4000, 16000) || u.CreatedAt.IsZero() || a.CreatedAt.IsZero() {
		return ErrInvalid
	}
	if len(a.Sources) > 3 || (a.Status != "answered" && a.Status != "needs_clarification") || (a.Status == "answered" && len(a.Sources) == 0) || (a.Status == "needs_clarification" && len(a.Sources) != 0) {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for _, s := range a.Sources {
		if !validSource(s) || seen[s.ID] {
			return ErrInvalid
		}
		seen[s.ID] = true
	}
	return nil
}

func historyPage(turns []Turn, limit int, before *int64) (HistoryPage, error) {
	if limit < 1 || limit > 100 || (before != nil && *before <= 0) {
		return HistoryPage{}, ErrInvalid
	}
	all := []Message{}
	for _, t := range turns {
		for _, m := range []Message{t.User, t.Assistant} {
			if before == nil || m.Seq < *before {
				all = append(all, m)
			}
		}
	}
	p := HistoryPage{Messages: all}
	if len(all) > limit {
		p.Messages = all[len(all)-limit:]
		n := p.Messages[0].Seq
		p.NextBefore = &n
	}
	return p, nil
}
