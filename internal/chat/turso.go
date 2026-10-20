package chat

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const storageBodyLimit = 8 << 20

type tursoHTTP struct {
	endpoint, token string
	client          *http.Client
}

// OpenTurso validates the production URL; constructing the adapter performs no I/O.
func OpenTurso(endpoint, token string) (*SQLStore, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || token == "" {
		return nil, ErrInvalid
	}
	return newTurso(strings.TrimRight(endpoint, "/")+"/v2/pipeline", token, &http.Client{Timeout: 3 * time.Second}), nil
}

// newTurso injects a loopback transport in package tests, never from runtime env.
func newTurso(endpoint, token string, client *http.Client) *SQLStore {
	c := *client
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &SQLStore{remote: &tursoHTTP{endpoint, token, &c}}
}

type tursoCell struct {
	Type   string          `json:"type"`
	Value  json.RawMessage `json:"value,omitempty"`
	Base64 *string         `json:"base64,omitempty"`
}

func bindCell(v any) (any, error) {
	switch x := v.(type) {
	case nil:
		return map[string]any{"type": "null"}, nil
	case string:
		return map[string]any{"type": "text", "value": x}, nil
	case int64:
		return map[string]any{"type": "integer", "value": strconv.FormatInt(x, 10)}, nil
	case int:
		return map[string]any{"type": "integer", "value": strconv.Itoa(x)}, nil
	case []byte:
		return map[string]any{"type": "blob", "base64": base64.StdEncoding.EncodeToString(x)}, nil
	default:
		return nil, ErrInvalid
	}
}

func (t *tursoHTTP) query(ctx context.Context, sql string, args ...any) (sqlResult, error) {
	bound := []any{}
	for _, arg := range args {
		v, err := bindCell(arg)
		if err != nil {
			return sqlResult{}, err
		}
		bound = append(bound, v)
	}
	body, _ := json.Marshal(map[string]any{"requests": []any{map[string]any{"type": "execute", "stmt": map[string]any{"sql": sql, "args": bound, "want_rows": true}}, map[string]any{"type": "close"}}})
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, t.endpoint, bytes.NewReader(body))
	if err != nil {
		return sqlResult{}, ErrUnavailable
	}
	r.Header.Set("Authorization", "Bearer "+t.token)
	r.Header.Set("Content-Type", "application/json")
	resp, err := t.client.Do(r)
	if err != nil {
		return sqlResult{}, ErrUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return sqlResult{}, ErrUnavailable
	}
	b, err := readBounded(resp.Body, storageBodyLimit)
	if err != nil {
		return sqlResult{}, ErrUnavailable
	}
	return decodeTurso(b)
}

func readBounded(r io.Reader, max int64) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, max+1))
	if err != nil || int64(len(b)) > max {
		return nil, ErrUnavailable
	}
	return b, nil
}

func decodeTurso(b []byte) (sqlResult, error) {
	if len(b) > storageBodyLimit {
		return sqlResult{}, ErrUnavailable
	}
	// Envelope metadata is deliberately not trusted or followed (base_url/baton).
	var envelope struct {
		Results []struct {
			Type     string `json:"type"`
			Response struct {
				Type   string          `json:"type"`
				Result json.RawMessage `json:"result"`
			} `json:"response"`
		} `json:"results"`
	}
	if !validJSON(b) || json.Unmarshal(b, &envelope) != nil || len(envelope.Results) != 2 {
		return sqlResult{}, ErrUnavailable
	}
	ex, close := envelope.Results[0], envelope.Results[1]
	if ex.Type != "ok" || ex.Response.Type != "execute" || close.Type != "ok" || close.Response.Type != "close" {
		return sqlResult{}, ErrUnavailable
	}
	var result struct {
		Cols []struct {
			Name *string `json:"name"`
		} `json:"cols"`
		Rows     [][]tursoCell `json:"rows"`
		Affected *int64        `json:"affected_row_count"`
	}
	if json.Unmarshal(ex.Response.Result, &result) != nil || result.Cols == nil || result.Rows == nil || result.Affected == nil || *result.Affected < 0 || *result.Affected > 100 || len(result.Rows) > 101 || len(result.Cols) > 16 {
		return sqlResult{}, ErrUnavailable
	}
	out := sqlResult{rows: [][]any{}, affected: *result.Affected}
	for _, col := range result.Cols {
		if col.Name == nil || len(*col.Name) > 128 {
			return sqlResult{}, ErrUnavailable
		}
		out.columns = append(out.columns, *col.Name)
	}
	for _, row := range result.Rows {
		if len(row) != len(result.Cols) {
			return sqlResult{}, ErrUnavailable
		}
		values := []any{}
		for _, cell := range row {
			var v any
			switch cell.Type {
			case "text", "integer":
				var s string
				if len(cell.Value) == 0 || string(cell.Value) == "null" || json.Unmarshal(cell.Value, &s) != nil || cell.Base64 != nil {
					return sqlResult{}, ErrUnavailable
				}
				if cell.Type == "integer" {
					n, err := strconv.ParseInt(s, 10, 64)
					if err != nil || strconv.FormatInt(n, 10) != s {
						return sqlResult{}, ErrUnavailable
					}
					v = n
				} else {
					v = s
				}
			case "blob":
				if cell.Base64 == nil || len(cell.Value) != 0 {
					return sqlResult{}, ErrUnavailable
				}
				raw, err := base64.StdEncoding.Strict().DecodeString(*cell.Base64)
				if err != nil {
					return sqlResult{}, ErrUnavailable
				}
				v = raw
			case "null":
				if len(cell.Value) != 0 || cell.Base64 != nil {
					return sqlResult{}, ErrUnavailable
				}
				v = nil
			default:
				return sqlResult{}, ErrUnavailable
			}
			values = append(values, v)
		}
		out.rows = append(out.rows, values)
	}
	return out, nil
}

func validJSON(b []byte) bool {
	// Unlike encoding/json alone, reject invalid UTF-8 instead of replacement runes.
	return json.Valid(b) && utf8.Valid(b)
}
