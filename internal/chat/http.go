package chat

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

const clarification = "I need more information: the currently approved support material does not support an answer to that question. Please clarify what you need help with. Demo knowledge describes a fictional sample organization, not real organization policies."

type HTTPConfig struct {
	AssistantOrigin string
	ParentOrigins   []string
}
type flight struct {
	request     string
	fingerprint [32]byte
}
type rateWindow struct {
	start             time.Time
	creates, messages int
}
type server struct {
	store     Store
	provider  Provider
	config    HTTPConfig
	mu        sync.Mutex
	active    map[string]flight
	rates     map[string]rateWindow
	inference chan struct{}
}

// ValidOrigin accepts a canonical origin, never a path, userinfo or forwarded header.
func ValidOrigin(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" || strings.ContainsAny(raw, "\r\n\t ") {
		return false
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1")) {
		return false
	}
	return u.String() == raw
}

func NewHandler(store Store, provider Provider, cfg HTTPConfig, assets fs.FS) (http.Handler, error) {
	if store == nil || provider == nil || assets == nil || !ValidOrigin(cfg.AssistantOrigin) || len(cfg.ParentOrigins) == 0 {
		return nil, ErrInvalid
	}
	for _, origin := range cfg.ParentOrigins {
		if !ValidOrigin(origin) {
			return nil, ErrInvalid
		}
	}
	s := &server{store: store, provider: provider, config: cfg, active: map[string]flight{}, rates: map[string]rateWindow{}, inference: make(chan struct{}, 8)}
	api := chi.NewRouter()
	api.Post("/conversations", s.create)
	api.Get("/conversations/{id}/messages", s.history)
	api.Post("/conversations/{id}/messages", s.message)
	api.NotFound(func(w http.ResponseWriter, r *http.Request) { apiError(w, 404, "not_found") })
	api.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) { apiError(w, 405, "method_not_allowed") })
	r := chi.NewRouter()
	r.Handle("/api/v1", s.apiMiddleware(api))
	r.Mount("/api/v1", s.apiMiddleware(api))
	csp := "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors " + strings.Join(cfg.ParentOrigins, " ")
	files := http.StripPrefix("/widget/", http.FileServer(http.FS(assets)))
	r.Handle("/widget/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		policy := csp
		if r.URL.Path == "/widget/sample-host.html" {
			policy += "; frame-src 'self'"
		}
		w.Header().Set("Content-Security-Policy", policy)
		files.ServeHTTP(w, r)
	}))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/widget/sample-host.html", http.StatusFound)
	})
	// All API-prefixed unknown routes still use the safe JSON envelope.
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			jsonHeaders(w)
			apiError(w, 404, "not_found")
			return
		}
		http.NotFound(w, r)
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			jsonHeaders(w)
			apiError(w, 405, "method_not_allowed")
			return
		}
		w.WriteHeader(405)
	})
	return r, nil
}

func jsonHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}
func (s *server) apiMiddleware(next http.Handler) http.Handler {
	guard := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonHeaders(w)
		defer func() {
			if recover() != nil {
				apiError(w, 500, "internal_error")
			}
		}()
		origin, has := r.Header["Origin"]
		if has && (len(origin) != 1 || origin[0] != s.config.AssistantOrigin) {
			apiError(w, 403, "origin_not_allowed")
			return
		}
		next.ServeHTTP(w, r)
	})
	timed := http.TimeoutHandler(guard, 20*time.Second, `{"error":{"code":"service_busy","message":"The assistant is temporarily unavailable. Please retry."}}`)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { jsonHeaders(w); timed.ServeHTTP(w, r) })
}

func apiError(w http.ResponseWriter, status int, code string) {
	message := "The request could not be completed."
	switch code {
	case "provider_unavailable", "storage_unavailable", "service_busy":
		message = "The assistant is temporarily unavailable. Please retry."
	case "unauthorized":
		message = "Conversation authorization is required."
	case "conversation_busy":
		message = "A question is already being processed. Please retry."
		w.Header().Set("Retry-After", "1")
	case "rate_limited":
		message = "Too many requests. Please retry later."
		w.Header().Set("Retry-After", "60")
	case "conversation_full":
		message = "This conversation is full. Please start a new conversation."
	case "idempotency_conflict":
		message = "That request ID was already used for a different question."
	case "origin_not_allowed":
		message = "This request origin is not allowed."
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func decodeBody(w http.ResponseWriter, r *http.Request, max int64, dst any) bool {
	media, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" || (params["charset"] != "" && !strings.EqualFold(params["charset"], "utf-8")) {
		apiError(w, 415, "unsupported_media_type")
		return false
	}
	b, err := readBounded(r.Body, max)
	if err != nil {
		apiError(w, 413, "request_too_large")
		return false
	}
	if strictJSON(b, dst) != nil {
		apiError(w, 400, "invalid_request")
		return false
	}
	return true
}

func (s *server) rate(w http.ResponseWriter, r *http.Request, creation bool) bool {
	peer, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		peer = r.RemoteAddr
	}
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	window, ok := s.rates[peer]
	if !ok {
		if len(s.rates) >= 10000 {
			oldest := ""
			var oldestTime time.Time
			for key, entry := range s.rates {
				if oldest == "" || entry.start.Before(oldestTime) {
					oldest, oldestTime = key, entry.start
				}
			}
			delete(s.rates, oldest)
		}
		window.start = now
	}
	if now.Sub(window.start) >= time.Minute {
		window = rateWindow{start: now}
	}
	allowed := true
	if creation {
		allowed = window.creates < 10
		if allowed {
			window.creates++
		}
	} else {
		allowed = window.messages < 30
		if allowed {
			window.messages++
		}
	}
	s.rates[peer] = window
	if !allowed {
		apiError(w, 429, "rate_limited")
	}
	return allowed
}

func (s *server) create(w http.ResponseWriter, r *http.Request) {
	if !s.rate(w, r, true) {
		return
	}
	if r.URL.RawQuery != "" {
		apiError(w, 400, "invalid_request")
		return
	}
	var body struct {
		Context *struct {
			PagePath *string `json:"page_path"`
			Locale   *string `json:"locale"`
		} `json:"context"`
	}
	if !decodeBody(w, r, 2048, &body) {
		return
	}
	var hostContext HostContext
	if body.Context != nil {
		if body.Context.PagePath != nil {
			hostContext.PagePath = *body.Context.PagePath
			if hostContext.PagePath == "" {
				apiError(w, 400, "invalid_request")
				return
			}
		}
		if body.Context.Locale != nil {
			hostContext.Locale = *body.Context.Locale
			if hostContext.Locale == "" {
				apiError(w, 400, "invalid_request")
				return
			}
		}
	}
	if ValidateContext(hostContext) != nil {
		apiError(w, 400, "invalid_request")
		return
	}
	idBytes := make([]byte, 16)
	capBytes := make([]byte, 32)
	if _, err := rand.Read(idBytes); err != nil {
		apiError(w, 500, "internal_error")
		return
	}
	if _, err := rand.Read(capBytes); err != nil {
		apiError(w, 500, "internal_error")
		return
	}
	id, capability := hex.EncodeToString(idBytes), hex.EncodeToString(capBytes)
	now := time.Now().UTC()
	c := Conversation{id, sha256.Sum256([]byte(capability)), hostContext, now}
	if s.store.CreateConversation(r.Context(), c) != nil {
		apiError(w, 503, "storage_unavailable")
		return
	}
	writeJSON(w, 201, struct {
		ID         string    `json:"conversation_id"`
		Capability string    `json:"capability"`
		CreatedAt  time.Time `json:"created_at"`
	}{id, capability, now})
}

func (s *server) authorize(w http.ResponseWriter, r *http.Request) (Conversation, bool) {
	id := chi.URLParam(r, "id")
	if !idPattern.MatchString(id) {
		apiError(w, 400, "invalid_request")
		return Conversation{}, false
	}
	values := r.Header.Values("Authorization")
	token := ""
	if len(values) == 1 && strings.HasPrefix(values[0], "Bearer ") {
		token = strings.TrimPrefix(values[0], "Bearer ")
	}
	valid := capPattern.MatchString(token)
	// Digest comparison is fixed-size even for unknown conversations and malformed tokens.
	digest := sha256.Sum256([]byte(token))
	c, err := s.store.Conversation(r.Context(), id)
	match := subtle.ConstantTimeCompare(c.CapabilityHash[:], digest[:])
	if !valid || errors.Is(err, ErrNotFound) || err == nil && match != 1 {
		apiError(w, 401, "unauthorized")
		return Conversation{}, false
	}
	if err != nil {
		apiError(w, 503, "storage_unavailable")
		return Conversation{}, false
	}
	return c, true
}

func (s *server) history(w http.ResponseWriter, r *http.Request) {
	c, ok := s.authorize(w, r)
	if !ok {
		return
	}
	if r.ContentLength != 0 || len(r.TransferEncoding) > 0 {
		apiError(w, 400, "invalid_request")
		return
	}
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		apiError(w, 400, "invalid_request")
		return
	}
	limit := 20
	var before *int64
	for key, values := range q {
		if len(values) != 1 {
			apiError(w, 400, "invalid_request")
			return
		}
		switch key {
		case "limit":
			n, e := strconv.Atoi(values[0])
			if e != nil || n < 1 || n > 100 {
				apiError(w, 400, "invalid_request")
				return
			}
			limit = n
		case "before":
			n, e := strconv.ParseInt(values[0], 10, 64)
			if e != nil || n <= 0 {
				apiError(w, 400, "invalid_request")
				return
			}
			before = &n
		default:
			apiError(w, 400, "invalid_request")
			return
		}
	}
	p, err := s.store.History(r.Context(), c.ID, limit, before)
	if err != nil {
		apiError(w, 503, "storage_unavailable")
		return
	}
	writeJSON(w, 200, p)
}

type turnEnvelope struct {
	RequestID string    `json:"request_id"`
	Messages  []Message `json:"messages"`
}

func sendTurn(w http.ResponseWriter, t Turn) {
	writeJSON(w, 200, turnEnvelope{t.RequestID, []Message{t.User, t.Assistant}})
}
func (s *server) replay(w http.ResponseWriter, ctx context.Context, id, request string, fp [32]byte) bool {
	t, err := s.store.FindTurn(ctx, id, request)
	if errors.Is(err, ErrNotFound) {
		return false
	}
	if err != nil {
		apiError(w, 503, "storage_unavailable")
		return true
	}
	if t.Fingerprint != fp {
		apiError(w, 409, "idempotency_conflict")
		return true
	}
	sendTurn(w, t)
	return true
}

func (s *server) message(w http.ResponseWriter, r *http.Request) {
	if !s.rate(w, r, false) {
		return
	}
	if r.URL.RawQuery != "" {
		apiError(w, 400, "invalid_request")
		return
	}
	var body struct {
		RequestID string `json:"request_id"`
		Text      string `json:"text"`
	}
	if !decodeBody(w, r, 16384, &body) {
		return
	}
	text, err := ValidateMessage(body.Text)
	if err != nil || !requestPattern.MatchString(body.RequestID) {
		apiError(w, 400, "invalid_request")
		return
	}
	c, ok := s.authorize(w, r)
	if !ok {
		return
	}
	fp := Fingerprint(text, c.Context)
	if s.replay(w, r.Context(), c.ID, body.RequestID, fp) {
		return
	}
	s.mu.Lock()
	if f, exists := s.active[c.ID]; exists {
		s.mu.Unlock()
		if f.request == body.RequestID && f.fingerprint != fp {
			apiError(w, 409, "idempotency_conflict")
		} else {
			apiError(w, 409, "conversation_busy")
		}
		return
	}
	s.active[c.ID] = flight{body.RequestID, fp}
	s.mu.Unlock()
	defer func() { s.mu.Lock(); delete(s.active, c.ID); s.mu.Unlock() }()
	if s.replay(w, r.Context(), c.ID, body.RequestID, fp) {
		return
	}
	history, err := s.store.History(r.Context(), c.ID, 100, nil)
	if err != nil {
		apiError(w, 503, "storage_unavailable")
		return
	}
	previous := int64(0)
	if len(history.Messages) > 0 {
		previous = history.Messages[len(history.Messages)-1].Seq / 2
	}
	if previous >= 50 {
		apiError(w, 409, "conversation_full")
		return
	}
	docs, err := s.store.ApprovedKnowledge(r.Context())
	if err != nil {
		apiError(w, 503, "storage_unavailable")
		return
	}
	docs = Retrieve(text, docs)
	answer, status, sources := clarification, "needs_clarification", []Source{}
	userTime := time.Now().UTC()
	if len(docs) > 0 {
		select {
		case s.inference <- struct{}{}:
		default:
			apiError(w, 503, "service_busy")
			return
		}
		composition, composeErr := func() (Composition, error) {
			defer func() { <-s.inference }()
			return s.provider.Compose(r.Context(), ComposeInput{text, c.Context, BoundHistory(history.Messages), docs})
		}()
		if composeErr != nil {
			apiError(w, 503, "provider_unavailable")
			return
		}
		approved, validationErr := validateComposition(composition, docs)
		if validationErr != nil {
			apiError(w, 503, "provider_unavailable")
			return
		}
		if !composition.Unsupported {
			answer, status, sources = composition.Answer, "answered", approved
		}
	}
	n := previous + 1
	t := Turn{c.ID, body.RequestID, fp, n, Message{2*n - 1, body.RequestID, "user", text, userTime, "submitted", []Source{}}, Message{2 * n, body.RequestID, "assistant", answer, time.Now().UTC(), status, sources}}
	// A canceled client may still receive zero or one whole completed pair. This
	// independent short commit window also lets a retry resolve lost acknowledgments.
	commitCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 3*time.Second)
	defer cancel()
	stored, err := s.store.CompleteTurn(commitCtx, t, previous)
	if err != nil {
		switch {
		case errors.Is(err, ErrIdempotencyConflict):
			apiError(w, 409, "idempotency_conflict")
		case errors.Is(err, ErrFull):
			apiError(w, 409, "conversation_full")
		case errors.Is(err, ErrConflict):
			apiError(w, 409, "conversation_busy")
		default:
			apiError(w, 503, "storage_unavailable")
		}
		return
	}
	sendTurn(w, stored)
}
