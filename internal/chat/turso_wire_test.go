package chat

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Independent response fixtures deliberately do not use the outbound bindCell.
func tursoWireCell(cell string) []byte {
	return []byte(`{"results":[{"type":"ok","response":{"type":"execute","result":{"cols":[{"name":"b"}],"rows":[[` + cell + `]],"affected_row_count":0}}},{"type":"ok","response":{"type":"close"}}]}`)
}

func tursoWireBlob(encoded string) []byte {
	b, _ := json.Marshal(encoded)
	return tursoWireCell(`{"type":"blob","base64":` + string(b) + `}`)
}

func TestContractTursoWireBlobs(t *testing.T) {
	for _, tc := range []struct {
		name, padded, raw string
		want              []byte
	}{
		{"empty", "", "", []byte{}},
		{"one", "AQ==", "AQ", []byte{1}},
		{"two", "AQI=", "AQI", []byte{1, 2}},
		{"three", "AQID", "AQID", []byte{1, 2, 3}},
		{"standard_alphabet", "+/8=", "+/8", []byte{251, 255}},
		{"digest32", "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE=", "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE", bytes.Repeat([]byte{1}, 32)},
	} {
		for _, form := range []struct{ name, encoded string }{{"padded", tc.padded}, {"raw", tc.raw}} {
			t.Run(tc.name+"/"+form.name, func(t *testing.T) {
				r, err := decodeTurso(tursoWireBlob(form.encoded))
				if err != nil || len(r.rows) != 1 || len(r.rows[0]) != 1 {
					t.Fatalf("known wire blob rejected: %v", err)
				}
				got, ok := r.rows[0][0].([]byte)
				if !ok || !bytes.Equal(got, tc.want) || r.affected != 0 {
					t.Fatalf("wrong typed blob: %#v", r)
				}
			})
		}
	}
}

func TestContractTursoWireBlobRejection(t *testing.T) {
	for _, encoded := range []string{
		"A", "AQIQQ", "AQ=", "AQ===", "AQI==", "AQID=", "=AQI", "AQ=I", "AQ==AA",
		"AR", "AR==", "AQJ", "AQJ=", // Nonzero trailing bits, raw and padded.
		"-_8", "-_8=", "AQ!", "AQ I", "世界",
	} {
		t.Run(encoded, func(t *testing.T) {
			if _, err := decodeTurso(tursoWireBlob(encoded)); !errors.Is(err, ErrUnavailable) {
				t.Fatalf("invalid blob accepted: %v", err)
			}
		})
	}
	for _, cell := range []string{
		`{"type":"blob"}`, `{"type":"blob","base64":null}`, `{"type":"blob","base64":1}`,
		`{"type":"blob","base64":true}`, `{"type":"blob","base64":[]}`, `{"type":"blob","base64":{}}`,
		`{"type":"blob","base64":"AQI","value":null}`, `{"type":"blob","base64":"AQI","value":"x"}`,
		`{"type":"text","base64":"AQI","value":"x"}`, `{"type":"null","base64":"AQI"}`,
	} {
		if _, err := decodeTurso(tursoWireCell(cell)); !errors.Is(err, ErrUnavailable) {
			t.Errorf("invalid typed cell accepted %s: %v", cell, err)
		}
	}
	valid := string(tursoWireBlob("AQI"))
	for _, wire := range []string{
		strings.Replace(valid, `"type":"close"`, `"type":"execute"`, 1),
		strings.Replace(valid, `"type":"ok"`, `"type":"error"`, 1),
		strings.Replace(valid, `"affected_row_count":0`, `"affected_row_count":-1`, 1),
		strings.Replace(valid, `"cols":[{"name":"b"}]`, `"cols":[]`, 1),
		valid + `{}`,
	} {
		if _, err := decodeTurso([]byte(wire)); !errors.Is(err, ErrUnavailable) {
			t.Errorf("invalid envelope accepted: %v", err)
		}
	}
}

// Known synthetic Conversation shape: SHA-256 of the ASCII capability "1" x64.
// The 32-byte/43-character raw-standard response matches the verified primary
// wire shape; this literal is not captured data or an outbound-bind roundtrip.
func tursoWireConversation(encoded string) []byte {
	return []byte(`{"results":[{"type":"ok","response":{"type":"execute","result":{"cols":[{"name":"id","decltype":null},{"name":"capability_hash","decltype":null},{"name":"context_json","decltype":null},{"name":"created_at","decltype":null}],"rows":[[{"type":"text","value":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},{"type":"blob","base64":"` + encoded + `"},{"type":"text","value":"{}"},{"type":"text","value":"2026-11-06T00:00:00Z"}]],"affected_row_count":0,"last_insert_rowid":null}}},{"type":"ok","response":{"type":"close"}}]}`)
}

func TestContractTursoWireConversationAuthorization(t *testing.T) {
	const rawDigest = "MTi7m8eN8nxHPs/RQQ971F66wfWc8/+c/k23eqt67dM"
	const emptyHistory = `{"results":[{"type":"ok","response":{"type":"execute","result":{"cols":[],"rows":[],"affected_row_count":0}}},{"type":"ok","response":{"type":"close"}}]}`
	id, capability := strings.Repeat("a", 32), strings.Repeat("1", 64)
	wantHash := sha256.Sum256([]byte(capability))
	for _, form := range []struct{ name, encoded string }{{"padded", rawDigest + "="}, {"raw", rawDigest}} {
		t.Run(form.name, func(t *testing.T) {
			wire := tursoWireConversation(form.encoded)
			var historyCalls atomic.Int32
			fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Requests []struct {
						Type string `json:"type"`
						Stmt struct {
							SQL string `json:"sql"`
						} `json:"stmt"`
					} `json:"requests"`
				}
				if r.Method != "POST" || r.URL.Path != "/v2/pipeline" || r.Header.Get("Authorization") != "Bearer synthetic-token" || json.NewDecoder(r.Body).Decode(&body) != nil || len(body.Requests) != 2 || body.Requests[0].Type != "execute" || body.Requests[1].Type != "close" {
					t.Error("unexpected local pipeline request")
					w.WriteHeader(500)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				switch body.Requests[0].Stmt.SQL {
				case `SELECT id,capability_hash,context_json,created_at FROM conversations WHERE id=?`:
					_, _ = w.Write(wire)
				case `SELECT ` + turnColumns + ` FROM turns WHERE conversation_id=? ORDER BY number ASC LIMIT 51`:
					historyCalls.Add(1)
					_, _ = w.Write([]byte(emptyHistory))
				default:
					t.Error("unexpected SQL at authorization seam")
					w.WriteHeader(500)
				}
			}))
			defer fixture.Close()
			s := newTurso(fixture.URL+"/v2/pipeline", "synthetic-token", fixture.Client())
			t.Run("Conversation", func(t *testing.T) {
				c, err := s.Conversation(context.Background(), id)
				if err != nil || c.ID != id || c.CapabilityHash != wantHash || c.Context != (HostContext{}) || !c.CreatedAt.Equal(time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)) {
					t.Fatalf("known Conversation rejected or changed: %v", err)
				}
			})
			h := testHandler(t, s, providerFunc(func(context.Context, ComposeInput) (Composition, error) {
				t.Error("history authorization must never reach provider")
				return Composition{}, ErrUnavailable
			}))
			for _, auth := range []struct{ name, bearer string }{{"missing", ""}, {"wrong", strings.Repeat("0", 64)}} {
				t.Run(auth.name, func(t *testing.T) {
					assertCode(t, perform(h, "GET", "/api/v1/conversations/"+id+"/messages", auth.bearer, ""), 401, "unauthorized")
					if historyCalls.Load() != 0 {
						t.Fatal("unauthorized request read history")
					}
				})
			}
			t.Run("valid", func(t *testing.T) {
				w := perform(h, "GET", "/api/v1/conversations/"+id+"/messages", capability, "")
				if w.Code != 200 || strings.TrimSpace(w.Body.String()) != `{"messages":[],"next_before":null}` || historyCalls.Load() != 1 {
					t.Fatalf("valid synthetic capability failed: %d %s", w.Code, w.Body)
				}
			})
		})
	}
}

func TestTursoBlobBindingRemainsPadded(t *testing.T) {
	cell, err := bindCell([]byte{1, 2})
	b, marshalErr := json.Marshal(cell)
	if err != nil || marshalErr != nil || string(b) != `{"base64":"AQI=","type":"blob"}` {
		t.Fatalf("outbound binding changed: %s %v %v", b, err, marshalErr)
	}
}
