package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func importRecords() []ImportRecord {
	return []ImportRecord{
		{ID: "faq-one", Title: "First reviewed FAQ", URL: "https://example.org/jitihada/help/one", Text: "First reviewed answer body.", Approved: true},
		{ID: "faq-two", Title: "Second reviewed FAQ", Text: "Second reviewed answer body."},
	}
}

func TestImportKnowledgeAppliesAndSurvivesReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "private", "import.db")
	s, closeStore, err := OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	result, err := s.ImportKnowledge(ctx, importRecords(), true)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inserted != 2 || result.Updated != 0 || result.Unchanged != 0 || result.Rejected != 0 {
		t.Fatalf("unexpected counts %+v", result)
	}
	if err = closeStore(); err != nil {
		t.Fatal(err)
	}
	s, closeStore, err = OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = closeStore() }()
	docs, err := s.ApprovedKnowledge(ctx)
	if err != nil || len(docs) != 1 || docs[0].ID != "faq-one" || docs[0].Text != "First reviewed answer body." {
		t.Fatalf("approved corpus after reopen %+v %v", docs, err)
	}
	// A record without approval is stored withdrawn and never retrieved.
	var approved int
	if err = s.db.QueryRow(`SELECT approved FROM knowledge WHERE id='faq-two'`).Scan(&approved); err != nil {
		t.Fatal(err)
	}
	if approved != 0 {
		t.Fatal("record without approval was stored approved")
	}
}

func TestImportKnowledgeDistinguishesInsertUpdateAndUnchanged(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	record := ImportRecord{ID: "faq-one", Title: "First reviewed FAQ", Text: "First reviewed answer body.", Approved: true}
	result, err := s.ImportKnowledge(ctx, []ImportRecord{record}, true)
	if err != nil || result.Inserted != 1 {
		t.Fatalf("insert %+v %v", result, err)
	}
	result, err = s.ImportKnowledge(ctx, []ImportRecord{record}, true)
	if err != nil || result.Unchanged != 1 || result.Inserted != 0 || result.Updated != 0 {
		t.Fatalf("unchanged %+v %v", result, err)
	}
	refreshed := record
	refreshed.Text = "Refreshed reviewed answer body."
	result, err = s.ImportKnowledge(ctx, []ImportRecord{refreshed}, true)
	if err != nil || result.Updated != 1 || result.Inserted != 0 {
		t.Fatalf("update %+v %v", result, err)
	}
	var text string
	if err = s.db.QueryRow(`SELECT text FROM knowledge WHERE id='faq-one'`).Scan(&text); err != nil || text != refreshed.Text {
		t.Fatalf("refreshed content %q %v", text, err)
	}
	result, err = s.ImportKnowledge(ctx, []ImportRecord{{ID: "faq-two", Title: "Second reviewed FAQ", Text: "Second body."}}, false)
	if err != nil || result.Inserted != 1 {
		t.Fatalf("second insert %+v %v", result, err)
	}
}

func TestImportKnowledgePreservesExistingWithdrawal(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	withdrawn := ImportRecord{ID: "faq-one", Title: "First reviewed FAQ", Text: "Withdrawn answer body."}
	if _, err := s.ImportKnowledge(ctx, []ImportRecord{withdrawn}, false); err != nil {
		t.Fatal(err)
	}
	approved := withdrawn
	approved.Text = "Refreshed reviewed answer body."
	approved.Approved = true
	result, err := s.ImportKnowledge(ctx, []ImportRecord{approved}, true)
	if err != nil || result.Updated != 1 {
		t.Fatalf("refresh %+v %v", result, err)
	}
	var approvedFlag int
	var text string
	if err = s.db.QueryRow(`SELECT approved,text FROM knowledge WHERE id='faq-one'`).Scan(&approvedFlag, &text); err != nil {
		t.Fatal(err)
	}
	if approvedFlag != 0 {
		t.Fatal("withdrawn record was re-approved by an import")
	}
	if text != approved.Text {
		t.Fatal("withdrawn record was not refreshed")
	}
	docs, err := s.ApprovedKnowledge(ctx)
	if err != nil || len(docs) != 0 {
		t.Fatalf("withdrawn record became retrievable %+v %v", docs, err)
	}
}

func TestImportKnowledgeRejectsUnapprovedRecordWithoutWrites(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	records := []ImportRecord{{ID: "faq-one", Title: "First reviewed FAQ", Text: "First reviewed answer body.", Approved: true}}
	if _, err := s.ImportKnowledge(ctx, records, false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("approved record accepted without approval: %v", err)
	}
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM knowledge`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("rejected import wrote rows")
	}
}

func TestImportKnowledgeRefusesApprovedCap(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	records := []ImportRecord{}
	for i := 0; i < 100; i++ {
		records = append(records, ImportRecord{ID: fmt.Sprintf("faq-%03d", i), Title: "Reviewed FAQ", Text: "Reviewed answer body.", Approved: true})
	}
	if _, err := s.ImportKnowledge(ctx, records, true); err != nil {
		t.Fatal(err)
	}
	extra := ImportRecord{ID: "faq-extra", Title: "Reviewed FAQ", Text: "Reviewed answer body.", Approved: true}
	if _, err := s.ImportKnowledge(ctx, []ImportRecord{extra}, true); !errors.Is(err, ErrInvalid) {
		t.Fatalf("approved cap not enforced: %v", err)
	}
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM knowledge WHERE id='faq-extra'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("refused import wrote a row")
	}
	// A withdrawn record does not consume an approved slot.
	if _, err := s.ImportKnowledge(ctx, []ImportRecord{{ID: "faq-withdrawn", Title: "Reviewed FAQ", Text: "Reviewed answer body."}}, true); err != nil {
		t.Fatal(err)
	}
}

func TestPlanImportDoesNotWrite(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	plan, err := s.PlanImport(ctx, importRecords(), false)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Inserted != 1 || plan.Rejected != 1 || plan.Updated != 0 || plan.Unchanged != 0 {
		t.Fatalf("unexpected plan %+v", plan)
	}
	if len(plan.Records) != 2 || plan.Records[0].ID != "faq-one" || plan.Records[0].Action != importReject || plan.Records[1].Action != importInsert {
		t.Fatalf("unexpected plan records %+v", plan.Records)
	}
	encoded, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "answer body") {
		t.Fatal("plan leaked record text")
	}
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM knowledge`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("preview wrote rows")
	}
	approvedPlan, err := s.PlanImport(ctx, importRecords(), true)
	if err != nil || approvedPlan.Inserted != 2 || approvedPlan.Rejected != 0 {
		t.Fatalf("approved plan %+v %v", approvedPlan, err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM knowledge`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("approved preview wrote rows")
	}
	// The applied import reports the same actions as the preview.
	result, err := s.ImportKnowledge(ctx, importRecords(), true)
	if err != nil || result.Inserted != approvedPlan.Inserted {
		t.Fatalf("applied import %+v %v", result, err)
	}
}

func TestParseImportJSONLAcceptsReviewedFile(t *testing.T) {
	data := "\n" + `{"id":"faq-one","title":"First reviewed FAQ","url":"https://example.org/jitihada/help/one","text":"First reviewed answer body.","approved":true}` + "\n\n" +
		`{"id":"faq-two","title":"Second reviewed FAQ","text":"Second reviewed answer body.","approved":false}` + "\n"
	records, err := ParseImportJSONL([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("unexpected record count %d", len(records))
	}
	if records[0].ID != "faq-one" || records[0].URL != "https://example.org/jitihada/help/one" || !records[0].Approved {
		t.Fatalf("unexpected first record %+v", records[0])
	}
	if records[1].ID != "faq-two" || records[1].URL != "" || records[1].Approved {
		t.Fatalf("unexpected second record %+v", records[1])
	}
}

func TestParseImportJSONLRejectsMalformedRecords(t *testing.T) {
	valid := `{"id":"faq-one","title":"First reviewed FAQ","text":"First reviewed answer body.","approved":true}`
	for name, data := range map[string]string{
		"unknown field":       `{"id":"faq-one","title":"T","text":"x","approved":true,"extra":1}`,
		"duplicate key":       `{"id":"faq-one","id":"faq-two","title":"T","text":"x","approved":true}`,
		"case folded field":   `{"ID":"faq-one","title":"T","text":"x","approved":true}`,
		"missing id":          `{"title":"T","text":"x","approved":true}`,
		"missing title":       `{"id":"faq-one","text":"x","approved":true}`,
		"missing text":        `{"id":"faq-one","title":"T","approved":true}`,
		"missing approved":    `{"id":"faq-one","title":"T","text":"x"}`,
		"wrong type id":       `{"id":5,"title":"T","text":"x","approved":true}`,
		"wrong type title":    `{"id":"faq-one","title":["T"],"text":"x","approved":true}`,
		"wrong type text":     `{"id":"faq-one","title":"T","text":5,"approved":true}`,
		"string approved":     `{"id":"faq-one","title":"T","text":"x","approved":"true"}`,
		"numeric approved":    `{"id":"faq-one","title":"T","text":"x","approved":1}`,
		"null approved":       `{"id":"faq-one","title":"T","text":"x","approved":null}`,
		"null id":             `{"id":null,"title":"T","text":"x","approved":true}`,
		"trailing json":       valid + `{"id":"faq-two","title":"T","text":"x","approved":true}`,
		"array line":          `["faq-one"]`,
		"bare number line":    `42`,
		"bare string line":    `"faq-one"`,
		"empty file":          "",
		"blank file":          "\n\n  \n",
		"invalid utf-8":       "{\"id\":\"faq-one\",\"title\":\"T\",\"text\":\"\xff\xfe\",\"approved\":true}",
		"control character":   `{"id":"faq-one","title":"T","text":"bad\u0001value","approved":true}`,
		"insecure url":        `{"id":"faq-one","title":"T","url":"http://example.org/faq","text":"x","approved":true}`,
		"url with credential": `{"id":"faq-one","title":"T","url":"https://user@example.org/faq","text":"x","approved":true}`,
		"invalid id":          `{"id":"FAQ One","title":"T","text":"x","approved":true}`,
		"blank text":          `{"id":"faq-one","title":"T","text":"   ","approved":true}`,
		"blank title":         `{"id":"faq-one","title":" ","text":"x","approved":true}`,
		"oversized text":      `{"id":"faq-one","title":"T","text":"` + strings.Repeat("a", 8001) + `","approved":true}`,
		"oversized title":     `{"id":"faq-one","title":"` + strings.Repeat("a", 641) + `","text":"x","approved":true}`,
		"duplicate record id": valid + "\n" + valid,
	} {
		if _, err := ParseImportJSONL([]byte(data)); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s accepted: %v", name, err)
		}
	}
	if _, err := ParseImportJSONL([]byte(strings.Repeat("x", maxImportBytes+1))); !errors.Is(err, ErrInvalid) {
		t.Error("oversized file accepted")
	}
	many := strings.Builder{}
	for i := 0; i < 101; i++ {
		fmt.Fprintf(&many, `{"id":"faq-%03d","title":"T","text":"x","approved":false}`+"\n", i)
	}
	if _, err := ParseImportJSONL([]byte(many.String())); !errors.Is(err, ErrInvalid) {
		t.Error("too many records accepted")
	}
}

func TestReadImportFileIsBounded(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "records.jsonl")
	if err := os.WriteFile(path, []byte(`{"id":"faq-one","title":"T","text":"x","approved":false}`+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	records, err := ReadImportFile(path)
	if err != nil || len(records) != 1 {
		t.Fatalf("unexpected import %+v %v", records, err)
	}
	if _, err = ReadImportFile(filepath.Join(dir, "missing.jsonl")); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing file accepted: %v", err)
	}
	if _, err = ReadImportFile(""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("empty path accepted: %v", err)
	}
	oversized := filepath.Join(dir, "oversized.jsonl")
	if err := os.WriteFile(oversized, []byte(strings.Repeat("x", maxImportBytes+1)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = ReadImportFile(oversized); !errors.Is(err, ErrInvalid) {
		t.Fatalf("oversized file accepted: %v", err)
	}
}

// The remote adapter applies the same single multi-row statement as SQLite;
// the fixture is a local protocol double, not a live Turso certification.
func TestContractImportKnowledgeOverTursoFixture(t *testing.T) {
	backing := testStore(t)
	ctx := context.Background()
	var calls atomic.Int32
	var statements []string
	var argumentCounts []int
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var body struct {
			Requests []struct {
				Type string `json:"type"`
				Stmt *struct {
					SQL  string      `json:"sql"`
					Args []tursoCell `json:"args"`
				} `json:"stmt,omitempty"`
			} `json:"requests"`
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil || len(body.Requests) != 2 || body.Requests[0].Stmt == nil {
			t.Error("unexpected pipeline request")
			w.WriteHeader(500)
			return
		}
		stmt := body.Requests[0].Stmt
		statements = append(statements, stmt.SQL)
		argumentCounts = append(argumentCounts, len(stmt.Args))
		args := []any{}
		for _, cell := range stmt.Args {
			cellJSON, _ := json.Marshal(cell)
			raw := []byte(`{"results":[{"type":"ok","response":{"type":"execute","result":{"cols":[{"name":"arg"}],"rows":[[` + string(cellJSON) + `]],"affected_row_count":0}}},{"type":"ok","response":{"type":"close"}}]}`)
			decoded, err := decodeTurso(raw)
			if err != nil {
				t.Error("untyped SQL bind", err)
				w.WriteHeader(500)
				return
			}
			args = append(args, decoded.rows[0][0])
		}
		result, err := backing.query(r.Context(), stmt.SQL, args...)
		if err != nil {
			_, _ = w.Write([]byte(`{"results":[{"type":"error","error":{"message":"private SQL failure"}},{"type":"ok","response":{"type":"close"}}]}`))
			return
		}
		_, _ = w.Write(tursoEnvelope(result))
	}))
	defer fixture.Close()
	s := newTurso(fixture.URL+"/v2/pipeline", "fixture-token", fixture.Client())
	if calls.Load() != 0 {
		t.Fatal("constructor contacted the remote database")
	}
	records := importRecords()
	result, err := s.ImportKnowledge(ctx, records, true)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inserted != 2 {
		t.Fatalf("unexpected remote counts %+v", result)
	}
	inserts := 0
	for index, statement := range statements {
		if strings.HasPrefix(statement, "INSERT INTO knowledge") {
			inserts++
			// The whole batch is one statement with five bound values per row.
			if statement != importInsertSQL(len(records)) {
				t.Fatalf("unexpected remote statement %q", statement)
			}
			if argumentCounts[index] != len(records)*5 {
				t.Fatalf("unexpected remote argument count %d", argumentCounts[index])
			}
			if strings.Contains(statement, "faq-one") || strings.Contains(statement, "First reviewed") {
				t.Fatal("remote statement interpolated a record value")
			}
		}
	}
	if inserts != 1 {
		t.Fatalf("unexpected remote statement count %d", inserts)
	}
	docs, err := backing.ApprovedKnowledge(ctx)
	if err != nil || len(docs) != 1 || docs[0].ID != "faq-one" {
		t.Fatalf("remote corpus %+v %v", docs, err)
	}
}

// One statement applies the whole batch: a failure in the middle of the batch
// leaves the corpus exactly as it was.
func TestImportKnowledgeAppliesWholeBatchAtomically(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	records := []ImportRecord{}
	for i := 0; i < 60; i++ {
		records = append(records, ImportRecord{ID: fmt.Sprintf("faq-%03d", i), Title: "Reviewed FAQ", Text: "Reviewed answer body.", Approved: true})
	}
	if _, err := s.db.Exec(`CREATE TRIGGER reject_one BEFORE INSERT ON knowledge WHEN NEW.id='faq-050' BEGIN SELECT RAISE(ABORT,'fixture failure'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ImportKnowledge(ctx, records, true); err == nil {
		t.Fatal("failing batch reported success")
	}
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM knowledge`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("partially applied batch left %d rows", count)
	}
	if _, err := s.db.Exec(`DROP TRIGGER reject_one`); err != nil {
		t.Fatal(err)
	}
	result, err := s.ImportKnowledge(ctx, records, true)
	if err != nil || result.Inserted != 60 {
		t.Fatalf("complete batch %+v %v", result, err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM knowledge`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 60 {
		t.Fatalf("incomplete batch stored %d rows", count)
	}
}

// The largest supported batch stays inside the adapter parameter limits.
func TestImportKnowledgeAppliesMaximumBatchInOneStatement(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	records := []ImportRecord{}
	for i := 0; i < 100; i++ {
		records = append(records, ImportRecord{ID: fmt.Sprintf("faq-%03d", i), Title: "Reviewed FAQ", Text: "Reviewed answer body.", Approved: true})
	}
	statement := importInsertSQL(len(records))
	if got := strings.Count(statement, "?"); got != 500 {
		t.Fatalf("unexpected placeholder count %d", got)
	}
	if got := strings.Count(statement, "(?,?,?,?,?)"); got != 100 {
		t.Fatalf("unexpected row count %d", got)
	}
	result, err := s.ImportKnowledge(ctx, records, true)
	if err != nil || result.Inserted != 100 {
		t.Fatalf("maximum batch %+v %v", result, err)
	}
	docs, err := s.ApprovedKnowledge(ctx)
	if err != nil || len(docs) != 100 {
		t.Fatalf("maximum batch corpus %d %v", len(docs), err)
	}
}

// Property: an accepted import file contains only valid, bounded, unique
// records; arbitrary bytes never panic and never produce an invalid record.
func FuzzParseImportJSONL(f *testing.F) {
	for _, data := range []string{
		`{"id":"faq-one","title":"T","text":"x","approved":true}`,
		`{"id":"faq-one","title":"T","url":"https://example.org/faq","text":"x","approved":false}`,
		`{"id":"faq-one","title":"T","text":"x"}`,
		`{"id":"faq-one","id":"faq-two","title":"T","text":"x","approved":true}`,
		`[]`,
		``,
		"\xff\xfe",
		strings.Repeat("a", 9000),
	} {
		f.Add([]byte(data))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		records, err := ParseImportJSONL(data)
		if err != nil {
			return
		}
		if len(records) == 0 || len(records) > maxImportRecords {
			t.Fatal("accepted an empty or oversized import")
		}
		seen := map[string]bool{}
		for _, record := range records {
			if !validImportRecord(record) {
				t.Fatalf("accepted an invalid record %+v", record)
			}
			if seen[record.ID] {
				t.Fatal("accepted duplicate record IDs")
			}
			seen[record.ID] = true
		}
		docs := make([]Knowledge, 0, len(records))
		for _, record := range records {
			docs = append(docs, Knowledge{ID: record.ID, Title: record.Title, URL: record.URL, Text: record.Text, Approved: record.Approved})
		}
		if validateKnowledge(docs) != nil {
			t.Fatal("accepted records violate the knowledge rules")
		}
	})
}
