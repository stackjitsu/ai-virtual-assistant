package chat

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

// Bounded import limits. The file size, record count, record bounds and
// approved count mirror the retrieval rules, so an imported corpus behaves
// exactly like seeded knowledge instead of widening retrieval.
const (
	maxImportBytes   = 256 << 10
	maxImportRecords = 100
	maxApprovedDocs  = 100
)

// Import actions reported by a plan. They describe the effect on the stored
// row only; record text is never part of a plan or a result.
const (
	importInsert    = "insert"
	importUpdate    = "update"
	importUnchanged = "unchanged"
	importReject    = "reject"
)

// ImportRecord is one reviewed JSONL record. It mirrors the stored knowledge
// row and is validated with the same rules retrieval applies.
type ImportRecord struct {
	ID       string
	Title    string
	URL      string
	Text     string
	Approved bool
}

// ImportPlanEntry reports what an import would do with one record.
type ImportPlanEntry struct {
	ID     string `json:"id"`
	Action string `json:"action"`
}

// ImportPlan is the read-only preview of an import.
type ImportPlan struct {
	Inserted  int               `json:"inserted"`
	Updated   int               `json:"updated"`
	Unchanged int               `json:"unchanged"`
	Rejected  int               `json:"rejected"`
	Records   []ImportPlanEntry `json:"records"`
}

// ImportResult counts what an applied import changed.
type ImportResult struct {
	Inserted  int `json:"inserted"`
	Updated   int `json:"updated"`
	Unchanged int `json:"unchanged"`
	Rejected  int `json:"rejected"`
}

// importLine mirrors the accepted JSONL object. Pointer fields distinguish a
// missing required key from an empty value, so "text":"" is rejected as blank
// while a missing "text" is rejected as absent.
type importLine struct {
	ID       *string `json:"id"`
	Title    *string `json:"title"`
	URL      *string `json:"url"`
	Text     *string `json:"text"`
	Approved *bool   `json:"approved"`
}

// ParseImportJSONL parses a bounded, strict JSONL import file. Every record is
// validated in memory, so a caller can never apply a partially valid file.
// Unknown or duplicate field names, missing required fields, wrong types, a
// non-boolean approval, duplicate record IDs, invalid UTF-8, control
// characters, an empty file, an oversized file and too many records are all
// refused before any database write.
func ParseImportJSONL(data []byte) ([]ImportRecord, error) {
	if len(data) > maxImportBytes || !utf8.Valid(data) {
		return nil, ErrInvalid
	}
	records := []ImportRecord{}
	seen := map[string]bool{}
	for number, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var parsed importLine
		if strictJSON([]byte(line), &parsed) != nil {
			return nil, fmt.Errorf("import line %d: %w", number+1, ErrInvalid)
		}
		if parsed.ID == nil || parsed.Title == nil || parsed.Text == nil || parsed.Approved == nil {
			return nil, fmt.Errorf("import line %d: %w", number+1, ErrInvalid)
		}
		record := ImportRecord{ID: *parsed.ID, Title: *parsed.Title, Text: *parsed.Text, Approved: *parsed.Approved}
		if parsed.URL != nil {
			record.URL = *parsed.URL
		}
		if seen[record.ID] {
			return nil, fmt.Errorf("import line %d: %w", number+1, ErrInvalid)
		}
		seen[record.ID] = true
		records = append(records, record)
		if len(records) > maxImportRecords {
			return nil, ErrInvalid
		}
	}
	if len(records) == 0 {
		return nil, ErrInvalid
	}
	docs := make([]Knowledge, 0, len(records))
	for _, record := range records {
		docs = append(docs, Knowledge{ID: record.ID, Title: record.Title, URL: record.URL, Text: record.Text, Approved: record.Approved})
	}
	if validateKnowledge(docs) != nil {
		return nil, ErrInvalid
	}
	return records, nil
}

// ReadImportFile reads a bounded JSONL import file from disk and parses it.
func ReadImportFile(path string) ([]ImportRecord, error) {
	if path == "" {
		return nil, ErrInvalid
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, ErrInvalid
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxImportBytes+1))
	if err != nil {
		return nil, ErrInvalid
	}
	return ParseImportJSONL(data)
}

// validImportRecord applies the stored knowledge bounds to one record: the
// source ID and title rules, the HTTPS-only source URL and the text bound
// retrieval uses.
func validImportRecord(record ImportRecord) bool {
	return validSource(Source{ID: record.ID, Title: record.Title, URL: record.URL}) && validText(record.Text, 8000, 8000)
}

// knowledgeRows reads the current rows for the given IDs. Values are bound,
// never interpolated.
func (s *SQLStore) knowledgeRows(ctx context.Context, ids []string) (map[string]Knowledge, error) {
	out := map[string]Knowledge{}
	if len(ids) == 0 {
		return out, nil
	}
	// Only literal placeholders are concatenated; every value stays bound.
	placeholders := strings.Repeat("?,", len(ids)-1) + "?"
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	r, err := s.query(ctx, `SELECT id,title,url,text,approved FROM knowledge WHERE id IN (`+placeholders+`)`, args...)
	if err != nil {
		return nil, err
	}
	for _, row := range r.rows {
		if len(row) != 5 {
			return nil, ErrUnavailable
		}
		id, a := textCell(row[0])
		title, b := textCell(row[1])
		u, c := textCell(row[2])
		text, d := textCell(row[3])
		approved, e := row[4].(int64)
		if !a || !b || !c || !d || !e || (approved != 0 && approved != 1) {
			return nil, ErrUnavailable
		}
		out[id] = Knowledge{ID: id, Title: title, URL: u, Text: text, Approved: approved == 1}
	}
	return out, nil
}

// approvedKnowledgeCount counts the currently approved rows. The count is a
// retrieval bound, not a storage limit.
func (s *SQLStore) approvedKnowledgeCount(ctx context.Context) (int, error) {
	r, err := s.query(ctx, `SELECT COUNT(*) FROM knowledge WHERE approved=1`)
	if err != nil {
		return 0, err
	}
	if len(r.rows) != 1 || len(r.rows[0]) != 1 {
		return 0, ErrUnavailable
	}
	n, ok := r.rows[0][0].(int64)
	if !ok || n < 0 || n > 1<<20 {
		return 0, ErrUnavailable
	}
	return int(n), nil
}

// importActions decides what every record does and counts the newly approved
// rows the import would create. It is pure: the existing rows and the approved
// count are read by the caller. The running approved count is the same
// computation the retrieval bound uses, so the plan and the applied import can
// never disagree about the bound.
func importActions(records []ImportRecord, existing map[string]Knowledge, approvedCount int, approve bool) ([]ImportPlanEntry, ImportResult, int) {
	entries := make([]ImportPlanEntry, 0, len(records))
	result := ImportResult{}
	approved := approvedCount
	newApproved := 0
	for _, record := range records {
		action := importInsert
		switch {
		case !validImportRecord(record):
			// An invalid record is rejected, never repaired.
			action = importReject
		case record.Approved && !approve:
			// Approval is an explicit operator decision.
			action = importReject
		default:
			current, ok := existing[record.ID]
			if ok {
				action = importUpdate
				// The stored approval flag is never overwritten, so only the
				// content decides between a refresh and no change.
				if current.Title == record.Title && current.URL == record.URL && current.Text == record.Text {
					action = importUnchanged
				}
			} else if record.Approved && approve {
				// A new approved row consumes one of the retrieval slots.
				newApproved++
				if approved >= maxApprovedDocs {
					action = importReject
				} else {
					approved++
				}
			}
		}
		entries = append(entries, ImportPlanEntry{ID: record.ID, Action: action})
		switch action {
		case importInsert:
			result.Inserted++
		case importUpdate:
			result.Updated++
		case importUnchanged:
			result.Unchanged++
		default:
			result.Rejected++
		}
	}
	return entries, result, newApproved
}

// PlanImport reports what an import would do without writing anything. A
// record that cannot be applied is reported as rejected; the plan never
// contains record text.
func (s *SQLStore) PlanImport(ctx context.Context, records []ImportRecord, approve bool) (ImportPlan, error) {
	if len(records) == 0 || len(records) > maxImportRecords {
		return ImportPlan{}, ErrInvalid
	}
	ids := make([]string, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.ID)
	}
	existing, err := s.knowledgeRows(ctx, ids)
	if err != nil {
		return ImportPlan{}, err
	}
	approved, err := s.approvedKnowledgeCount(ctx)
	if err != nil {
		return ImportPlan{}, err
	}
	entries, result, _ := importActions(records, existing, approved, approve)
	return ImportPlan{
		Inserted:  result.Inserted,
		Updated:   result.Updated,
		Unchanged: result.Unchanged,
		Rejected:  result.Rejected,
		Records:   entries,
	}, nil
}

// importInsertSQL builds the single multi-row upsert that applies a whole
// validated batch. Only literal placeholders are used, so every value stays a
// bound parameter, and one statement is one atomic unit of work: the batch
// commits as a whole or leaves the corpus untouched. The conflict clause
// refreshes title, url and text only, so the stored approval flag of an
// existing row is never rewritten.
func importInsertSQL(rows int) string {
	values := strings.TrimSuffix(strings.Repeat("(?,?,?,?,?),", rows), ",")
	return `INSERT INTO knowledge(id,title,url,text,approved) VALUES ` + values + ` ON CONFLICT(id) DO UPDATE SET title=excluded.title,url=excluded.url,text=excluded.text`
}

// ImportKnowledge applies validated records. Every record is checked before
// the first statement runs: an approval that was not granted, an invalid
// record or an import that would push the approved corpus above the retrieval
// bound refuses the whole import without writing anything. A new row is stored
// approved only when the record asks for approval and the operator granted
// it; every other new row is stored withdrawn.
//
// The whole batch is written by one parameterized statement, so it is applied
// completely or not at all. An import can refresh the title, url and text of
// an existing row, but it can never withdraw an existing approved row:
// withdrawal needs a separate reviewed operation.
func (s *SQLStore) ImportKnowledge(ctx context.Context, records []ImportRecord, approve bool) (ImportResult, error) {
	if len(records) == 0 || len(records) > maxImportRecords {
		return ImportResult{}, ErrInvalid
	}
	ids := make([]string, 0, len(records))
	seen := map[string]bool{}
	for _, record := range records {
		if !validImportRecord(record) || seen[record.ID] {
			return ImportResult{}, ErrInvalid
		}
		seen[record.ID] = true
		ids = append(ids, record.ID)
	}
	// The approval decision is checked before any SQL runs at all.
	for _, record := range records {
		if record.Approved && !approve {
			return ImportResult{}, ErrInvalid
		}
	}
	existing, err := s.knowledgeRows(ctx, ids)
	if err != nil {
		return ImportResult{}, err
	}
	approved, err := s.approvedKnowledgeCount(ctx)
	if err != nil {
		return ImportResult{}, err
	}
	_, result, newApproved := importActions(records, existing, approved, approve)
	// The single authoritative retrieval bound: the approved corpus may not
	// exceed the limit. The planner refused the records that would exceed it,
	// so the whole import is refused instead of silently dropping records.
	if approved+newApproved > maxApprovedDocs {
		return ImportResult{}, ErrInvalid
	}
	args := make([]any, 0, len(records)*5)
	for _, record := range records {
		// The approved value is bound as an integer so both adapters accept
		// it; the conflict clause never rewrites an existing approval flag.
		stored := int64(0)
		if record.Approved && approve {
			stored = 1
		}
		args = append(args, record.ID, record.Title, record.URL, record.Text, stored)
	}
	if _, err = s.query(ctx, importInsertSQL(len(records)), args...); err != nil {
		// The statement is atomic, so a failure means nothing was applied.
		// A lost acknowledgement is the one exception: the batch may have
		// committed, so the stored rows decide what is reported.
		if s.importApplied(ctx, records, existing, approve) {
			return result, nil
		}
		return ImportResult{}, err
	}
	return result, nil
}

// importApplied reports whether the stored rows already hold the state the
// import would have written. It resolves an uncertain write outcome instead
// of guessing.
func (s *SQLStore) importApplied(ctx context.Context, records []ImportRecord, existing map[string]Knowledge, approve bool) bool {
	ids := make([]string, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.ID)
	}
	stored, err := s.knowledgeRows(ctx, ids)
	if err != nil {
		return false
	}
	for _, record := range records {
		current, ok := stored[record.ID]
		if !ok || current.Title != record.Title || current.URL != record.URL || current.Text != record.Text {
			return false
		}
		wantApproved := false
		if previous, existed := existing[record.ID]; existed {
			wantApproved = previous.Approved
		} else {
			wantApproved = record.Approved && approve
		}
		if current.Approved != wantApproved {
			return false
		}
	}
	return true
}
