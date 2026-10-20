package chat

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func testStore(t *testing.T) *SQLStore {
	t.Helper()
	s, closeStore, err := OpenSQLite(filepath.Join(t.TempDir(), "store.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = closeStore() })
	if err = s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return s
}
func testConversation(t *testing.T, s Store, id string) Conversation {
	t.Helper()
	c := Conversation{id, sha256.Sum256([]byte(strings.Repeat("c", 64))), HostContext{"/help", "en"}, time.Now().UTC()}
	if err := s.CreateConversation(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	return c
}
func testTurn(c Conversation, req, text string, n int64) Turn {
	now := time.Now().UTC()
	return Turn{c.ID, req, Fingerprint(text, c.Context), n, Message{2*n - 1, req, "user", text, now, "submitted", []Source{}}, Message{2 * n, req, "assistant", "Sample answer", now, "answered", []Source{{"demo-start", "Sample source", "https://example.org/help"}}}}
}

func TestSQLiteReopenAtomicRollbackAndReplay(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "private", "store.db")
	s, closeStore, err := OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeStore != nil {
			_ = closeStore()
		}
	}()
	if err = s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err = s.Migrate(ctx); err != nil {
		t.Fatal("migration not idempotent", err)
	}
	c := testConversation(t, s, strings.Repeat("a", 32))
	turn := testTurn(c, "request_001", "start", 1)
	// A rejected atomic insert cannot consume a sequence or persist a half turn.
	if _, err = s.db.Exec(`CREATE TRIGGER reject_turn BEFORE INSERT ON turns BEGIN SELECT RAISE(ABORT,'fixture failure'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err = s.CompleteTurn(ctx, turn, 0); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("rejected write %v", err)
	}
	p, err := s.History(ctx, c.ID, 100, nil)
	if err != nil || len(p.Messages) != 0 {
		t.Fatal("partial pair after rejected insert", p, err)
	}
	if _, err = s.db.Exec(`DROP TRIGGER reject_turn`); err != nil {
		t.Fatal(err)
	}
	stored, err := s.CompleteTurn(ctx, turn, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err = closeStore(); err != nil {
		t.Fatal(err)
	}
	closeStore = nil
	s, closeStore, err = OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := s.Conversation(ctx, c.ID)
	if err != nil || !reflect.DeepEqual(reopened, c) {
		t.Fatalf("conversation reopen %+v %v", reopened, err)
	}
	replay, err := s.FindTurn(ctx, c.ID, turn.RequestID)
	if err != nil || !reflect.DeepEqual(replay, stored) {
		t.Fatalf("replay differs %+v %v", replay, err)
	}
	replacement := turn
	replacement.Assistant.Text = "must not replace stored answer"
	replayed, err := s.CompleteTurn(ctx, replacement, 0)
	if err != nil || !reflect.DeepEqual(replayed, stored) {
		t.Fatal("did not return stored replay", err)
	}
	divergent := testTurn(c, turn.RequestID, "different", 1)
	if _, err = s.CompleteTurn(ctx, divergent, 0); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("fingerprint conflict %v", err)
	}
	var digest []byte
	var rawContext string
	if err = s.db.QueryRow(`SELECT capability_hash,context_json FROM conversations`).Scan(&digest, &rawContext); err != nil || len(digest) != 32 || strings.Contains(rawContext, strings.Repeat("c", 64)) {
		t.Fatal("capability storage", err)
	}
	b, err := os.ReadFile(path)
	if err != nil || strings.Contains(string(b), strings.Repeat("c", 64)) {
		t.Fatal("raw capability persisted", err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("DB permission", err)
	}
	if len(replay.Assistant.Sources) != 1 || replay.Assistant.Sources[0].Title != "Sample source" {
		t.Fatal("source snapshot lost")
	}
}

func TestSQLiteCASConcurrencyBoundsAndPagination(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := testConversation(t, s, strings.Repeat("a", 32))
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, req := range []string{"request_001", "request_002"} {
		wg.Go(func() { _, err := s.CompleteTurn(ctx, testTurn(c, req, "start", 1), 0); results <- err })
	}
	wg.Wait()
	close(results)
	success, stale := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ErrConflict) {
			stale++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || stale != 1 {
		t.Fatalf("CAS results %d %d", success, stale)
	}
	var firstRequest string
	if err := s.db.QueryRow(`SELECT request_id FROM turns`).Scan(&firstRequest); err != nil {
		t.Fatal(err)
	}
	// Concurrent equal retries both resolve to the exact stored whole pair.
	for range 2 {
		wg.Go(func() {
			_, err := s.CompleteTurn(ctx, testTurn(c, firstRequest, "start", 1), 0)
			if err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	for n := int64(2); n <= 50; n++ {
		if _, err := s.CompleteTurn(ctx, testTurn(c, "request_"+strings.Repeat("x", int(n)), "start", n), n-1); err != nil {
			t.Fatal(n, err)
		}
	}
	p, err := s.History(ctx, c.ID, 3, nil)
	if err != nil || len(p.Messages) != 3 || p.Messages[0].Seq != 98 || p.Messages[2].Seq != 100 || p.NextBefore == nil || *p.NextBefore != 98 {
		t.Fatalf("odd latest page %+v %v", p, err)
	}
	before := int64(4)
	p, err = s.History(ctx, c.ID, 2, &before)
	if err != nil || p.Messages[0].Seq != 2 || p.Messages[1].Seq != 3 || p.NextBefore == nil || *p.NextBefore != 2 {
		t.Fatalf("odd cursor %+v %v", p, err)
	}
	before = 2
	p, err = s.History(ctx, c.ID, 2, &before)
	if err != nil || len(p.Messages) != 1 || p.NextBefore != nil {
		t.Fatal(p, err)
	}
	if _, err = s.CompleteTurn(ctx, testTurn(c, "request_full", "start", 50), 49); !errors.Is(err, ErrFull) {
		t.Fatalf("full %v", err)
	}
	invalid := testTurn(c, "request_bad", "start", 1)
	invalid.Assistant.Seq = 900
	if _, err = s.CompleteTurn(ctx, invalid, 0); !errors.Is(err, ErrInvalid) {
		t.Fatal("accepted invalid pair", err)
	}
	if _, err = s.History(ctx, c.ID, 101, nil); !errors.Is(err, ErrInvalid) {
		t.Fatal("history bound", err)
	}
}

func TestKnowledgeSeedIsExplicitAndWithdrawal(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	docs, err := s.ApprovedKnowledge(ctx)
	if err != nil || len(docs) != 0 {
		t.Fatal("unconditional seed", err)
	}
	if err = s.SeedDemo(ctx); err != nil {
		t.Fatal(err)
	}
	if err = s.SeedDemo(ctx); err != nil {
		t.Fatal(err)
	}
	docs, err = s.ApprovedKnowledge(ctx)
	if err != nil || len(docs) != 2 {
		t.Fatal("seed", err)
	}
	for _, d := range docs {
		if !strings.Contains(strings.ToLower(d.Text), "fictional") {
			t.Fatal("unlabelled sample")
		}
	}
	if _, err = s.db.Exec(`UPDATE knowledge SET approved=0 WHERE id=?`, "demo-start"); err != nil {
		t.Fatal(err)
	}
	docs, err = s.ApprovedKnowledge(ctx)
	if err != nil || len(docs) != 1 || docs[0].ID != "demo-support" {
		t.Fatal("withdrawal ignored", err)
	}
}
