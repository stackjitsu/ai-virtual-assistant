package chat

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Escalation is one immutably appended ticket record. The visitor-provided
// email is stored only in its bounded, validated, lowercase-trimmed form.
type Escalation struct {
	ConversationID  string
	TicketID        string
	Email           string
	QuestionNote    string
	RecordedAt      time.Time
	DeliveryOutcome string
}

type EscalationRequest struct {
	Email        string
	QuestionNote string
}

// Escalation constants are additive slice bounds; existing message bounds are
// unchanged. The honest outcome is defined by construction: the schema CHECK
// admits only delivery_not_wired, so a recorded ticket can never claim a
// sent email while no outgoing transport exists anywhere in this codebase.
const (
	DeliveryNotWired = "delivery_not_wired"
	// DeliveryNotWiredMessage is the only truthful visitor-facing delivery
	// wording this slice supports.
	DeliveryNotWiredMessage = "Your request has been recorded. Email delivery is not configured, so no email has been sent."
	maxEscalationBodyBytes  = 4096
	// Escalation email bound follows the practical SMTP address limit; local
	// and domain parts carry their own tighter bounds below.
	maxEscalationEmailBytes = 254
	maxEmailLocalBytes      = 64
	maxEmailDomainBytes     = 253
	maxEscalationNoteCP     = 1000
	maxEscalationNoteBytes  = 2000
	// maxEscalationsPerConversation is enforced atomically by the store's
	// UNIQUE(conversation_id,ticket_number) constraint, not by a count race.
	maxEscalationsPerConversation = 5
	// escalationAttemptsPerMinute is the per direct-peer-IP fixed window,
	// counting rejected attempts exactly like message attempts.
	escalationAttemptsPerMinute = 10
	// localOutboxBound keeps the recording local adapter process-bounded.
	localOutboxBound = 100
)

var (
	emailLocalPattern  = regexp.MustCompile(`^[a-z0-9]+(?:[._%+-][a-z0-9]+)*$`)
	emailDomainPattern = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$`)
)

// ValidateEscalationEmail accepts the raw visitor email, trims and lowercases
// it exactly once, and returns the canonical stored form. Accepted values are
// pure ASCII, so validation is idempotent on canonical input.
func ValidateEscalationEmail(raw string) (string, error) {
	if len(raw) > maxEscalationEmailBytes || !utf8.ValidString(raw) {
		return "", ErrInvalid
	}
	local, domain, found := strings.Cut(strings.ToLower(strings.TrimSpace(raw)), "@")
	if !found || len(local) > maxEmailLocalBytes || len(domain) > maxEmailDomainBytes ||
		!emailLocalPattern.MatchString(local) || !emailDomainPattern.MatchString(domain) {
		return "", ErrInvalid
	}
	return local + "@" + domain, nil
}

// ValidateEscalationNote trims the optional note; empty means absent.
func ValidateEscalationNote(raw string) (string, error) {
	note := strings.TrimSpace(raw)
	if note == "" {
		return "", nil
	}
	if !validText(note, maxEscalationNoteCP, maxEscalationNoteBytes) {
		return "", ErrInvalid
	}
	return note, nil
}

// decodeEscalationRequest applies the strict wire contract: consent must be
// the literal boolean true and fails closed; email is required; the note is
// optional. The decoded records already carry canonical field values.
func decodeEscalationRequest(emailRaw, noteRaw *string, consent *bool) (EscalationRequest, error) {
	if consent == nil || !*consent || emailRaw == nil {
		return EscalationRequest{}, ErrInvalid
	}
	email, err := ValidateEscalationEmail(*emailRaw)
	if err != nil {
		return EscalationRequest{}, err
	}
	noteRawValue := ""
	if noteRaw != nil {
		noteRawValue = *noteRaw
	}
	note, err := ValidateEscalationNote(noteRawValue)
	if err != nil {
		return EscalationRequest{}, err
	}
	return EscalationRequest{email, note}, nil
}

// validateEscalation is called by the store on every path; callers cannot
// bypass the record invariants. Canonical values must re-validate unchanged.
func validateEscalation(t Escalation) error {
	if !idPattern.MatchString(t.ConversationID) || !idPattern.MatchString(t.TicketID) ||
		t.RecordedAt.IsZero() || t.DeliveryOutcome != DeliveryNotWired {
		return ErrInvalid
	}
	email, err := ValidateEscalationEmail(t.Email)
	if err != nil || email != t.Email {
		return ErrInvalid
	}
	note, err := ValidateEscalationNote(t.QuestionNote)
	if err != nil || note != t.QuestionNote {
		return ErrInvalid
	}
	return nil
}

// EmailOutbox is the outgoing-delivery seam for stored escalation tickets.
// Real transport remains out of scope; only the recording adapter exists.
type EmailOutbox interface {
	// Deliver observes a ticket after it is stored. The consulted outcome is
	// already recorded on the ticket, so a transport error can never turn a
	// saved request into a claimed sent email.
	Deliver(ctx context.Context, t Escalation) error
}

// LocalEmailOutbox records pending tickets in process memory only; it
// performs zero network I/O and is the adapter used while delivery is not
// configured.
type LocalEmailOutbox struct {
	mu      sync.Mutex
	pending []Escalation
}

// Pending returns a bounded copy of recorded tickets for local inspection.
func (o *LocalEmailOutbox) Pending() []Escalation {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]Escalation{}, o.pending...)
}

func (o *LocalEmailOutbox) Deliver(_ context.Context, t Escalation) error {
	if validateEscalation(t) != nil {
		return ErrInvalid
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if len(o.pending) >= localOutboxBound {
		o.pending = o.pending[1:]
	}
	o.pending = append(o.pending, t)
	return nil
}
