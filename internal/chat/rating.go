package chat

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

const ratingAttemptsPerMinute = 10

// Rating is the single canonical, revisable record belonging to a conversation.
type Rating struct {
	ConversationID string    `json:"-"`
	Value          int       `json:"rating"`
	Comment        string    `json:"comment"`
	RatedAt        time.Time `json:"rated_at"`
}

var ErrRatingRequiresTurn = errors.New("rating requires a completed turn")

func canonicalRatingComment(raw string) (string, error) {
	comment := strings.TrimSpace(raw)
	if comment != "" && !validText(comment, 1000, 2000) {
		return "", ErrInvalid
	}
	return comment, nil
}

func decodeRatingRequest(b []byte) (int, string, error) {
	var body struct {
		Rating  json.RawMessage `json:"rating,omitempty"`
		Comment json.RawMessage `json:"comment,omitempty"`
	}
	if len(b) > 4096 || strictJSON(b, &body) != nil || len(body.Rating) != 1 || body.Rating[0] < '1' || body.Rating[0] > '5' {
		return 0, "", ErrInvalid
	}
	comment := ""
	if len(body.Comment) != 0 {
		if string(body.Comment) == "null" || json.Unmarshal(body.Comment, &comment) != nil || len(body.Comment) == 0 || body.Comment[0] != '"' {
			return 0, "", ErrInvalid
		}
	}
	canonical, err := canonicalRatingComment(comment)
	if err != nil {
		return 0, "", err
	}
	return int(body.Rating[0] - '0'), canonical, nil
}

func validateRating(r Rating) error {
	if !idPattern.MatchString(r.ConversationID) || r.Value < 1 || r.Value > 5 || r.RatedAt.IsZero() || !utf8.ValidString(r.Comment) {
		return ErrInvalid
	}
	canonical, err := canonicalRatingComment(r.Comment)
	if err != nil || canonical != r.Comment {
		return ErrInvalid
	}
	return nil
}

func (s *server) rateRating(w http.ResponseWriter, r *http.Request) bool {
	peer, now := peerAddress(r), time.Now()
	s.mu.Lock()
	window := s.window(peer, now)
	allowed := window.ratings < ratingAttemptsPerMinute
	if allowed {
		window.ratings++
	}
	s.rates[peer] = window
	s.mu.Unlock()
	if !allowed {
		apiError(w, 429, "rate_limited")
	}
	return allowed
}

func (s *server) rateConversation(w http.ResponseWriter, r *http.Request) {
	if !s.rateRating(w, r) {
		return
	}
	if r.URL.RawQuery != "" {
		apiError(w, 400, "invalid_request")
		return
	}
	media := struct {
		Rating  json.RawMessage `json:"rating,omitempty"`
		Comment json.RawMessage `json:"comment,omitempty"`
	}{}
	// Reuse the existing 4 KiB media/body/strict-JSON gate; the raw
	// rating token is then checked for the exact integer literal 1–5.
	if !decodeBody(w, r, 4096, &media) {
		return
	}
	b, _ := json.Marshal(media)
	value, comment, err := decodeRatingRequest(b)
	if err != nil {
		apiError(w, 400, "invalid_request")
		return
	}
	c, ok := s.authorize(w, r)
	if !ok {
		return
	}
	commitCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 3*time.Second)
	defer cancel()
	stored, err := s.store.SaveRating(commitCtx, Rating{c.ID, value, comment, time.Now().UTC()})
	if err != nil {
		if errors.Is(err, ErrRatingRequiresTurn) {
			apiError(w, 409, "rating_requires_turn")
		} else {
			apiError(w, 503, "storage_unavailable")
		}
		return
	}
	writeJSON(w, 200, stored)
}
