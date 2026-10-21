package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"ai-virtual-assistant/internal/chat"
)

// Invoke the actual CLI in a subprocess, with an explicit credential-free env.
func TestAssistantCLIProcess(t *testing.T) {
	if os.Getenv("ASSISTANT_CLI_HELPER") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{"assistant"}, os.Args[i+1:]...)
			main()
			return
		}
	}
	t.Fatal("missing helper command")
}

func cliCommand(ctx context.Context, env []string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=^TestAssistantCLIProcess$", "--"}, args...)...)
	cmd.Env = append([]string{"ASSISTANT_CLI_HELPER=1", "OPENAI_API_KEY=", "TURSO_AUTH_TOKEN=", "TURSO_DATABASE_URL="}, env...)
	return cmd
}

func TestContractCLIRemoteMigrationAndSeedGuards(t *testing.T) {
	var connections atomic.Int32
	fixture := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("guarded command contacted remote database")
	}))
	fixture.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}
	fixture.StartTLS()
	defer fixture.Close()
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"migrate"}, "remote migration requires migrate --allow-remote"},
		{[]string{"serve", "--allow-remote"}, "invalid command or flags"},
		{[]string{"serve", "--seed-demo"}, "demo seed is local-only"},
		{[]string{"seed-demo"}, "demo seed is local-only"},
		{[]string{"unknown"}, "invalid command or flags"},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cmd := cliCommand(ctx, []string{"STORE_MODE=turso", "TURSO_DATABASE_URL=" + fixture.URL, "TURSO_AUTH_TOKEN=fixture-token"}, tc.args...)
			out, err := cmd.CombinedOutput()
			if err == nil || !strings.Contains(string(out), tc.want) || connections.Load() != 0 {
				t.Fatalf("guard: err=%v connections=%d output=%s", err, connections.Load(), out)
			}
		})
	}
}

func TestContractCLILocalSeedNoProviderAndRestart(t *testing.T) {
	dir := t.TempDir()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()
	origin := "http://" + addr
	env := []string{"STORE_MODE=sqlite", "SQLITE_PATH=" + filepath.Join(dir, "private", "assistant.db"), "LISTEN_ADDR=" + addr, "ASSISTANT_ORIGIN=" + origin, "WIDGET_PARENT_ORIGINS=" + origin}
	client := &http.Client{Timeout: time.Second}
	start := func(args ...string) func() {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		cmd := cliCommand(ctx, env, args...)
		logFile, err := os.CreateTemp(dir, "cli-*.log")
		if err != nil {
			cancel()
			t.Fatal(err)
		}
		cmd.Stdout, cmd.Stderr = logFile, logFile
		if err = cmd.Start(); err != nil {
			cancel()
			_ = logFile.Close()
			t.Fatal(err)
		}
		stopped := false
		stop := func() {
			if stopped {
				return
			}
			stopped = true
			_ = cmd.Process.Signal(syscall.SIGTERM)
			err := cmd.Wait()
			cancel()
			_ = logFile.Close()
			if err != nil {
				out, _ := os.ReadFile(logFile.Name())
				t.Errorf("CLI stop: %v %s", err, out)
			}
		}
		t.Cleanup(stop)
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			resp, err := client.Get(origin + "/widget/sample-host.html")
			if err == nil {
				_ = resp.Body.Close()
				if resp.StatusCode == 200 {
					return stop
				}
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatal("local CLI did not serve embedded sample host")
		return stop
	}
	request := func(method, path, capability, body string, status int) []byte {
		t.Helper()
		r, err := http.NewRequest(method, origin+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		if method == "POST" {
			r.Header.Set("Content-Type", "application/json")
		}
		if capability != "" {
			r.Header.Set("Authorization", "Bearer "+capability)
		}
		resp, err := client.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		if err != nil || resp.StatusCode != status {
			t.Fatalf("%s %s: status=%d body=%s err=%v", method, path, resp.StatusCode, b, err)
		}
		return b
	}
	stop := start("serve", "--seed-demo")
	var created struct {
		ID         string `json:"conversation_id"`
		Capability string `json:"capability"`
	}
	if json.Unmarshal(request("POST", "/api/v1/conversations", "", `{}`, 201), &created) != nil || created.ID == "" || created.Capability == "" {
		t.Fatal("invalid creation response")
	}
	path := "/api/v1/conversations/" + created.ID + "/messages"
	unavailable := request("POST", path, created.Capability, `{"request_id":"request_001","text":"start onboarding"}`, 503)
	if !bytes.Contains(unavailable, []byte(`"code":"provider_unavailable"`)) {
		t.Fatal("missing provider did not fail honestly", string(unavailable))
	}
	if got := request("GET", path, created.Capability, "", 200); string(got) != "{\"messages\":[],\"next_before\":null}\n" {
		t.Fatal("no-key failure left a half turn", string(got))
	}
	body := `{"request_id":"request_002","text":"quantum zebras"}`
	first := request("POST", path, created.Capability, body, 200)
	if !bytes.Contains(first, []byte(`"status":"needs_clarification"`)) {
		t.Fatal("unsupported question was not a clarification")
	}
	stop()
	start("serve")
	if replay := request("POST", path, created.Capability, body, 200); !bytes.Equal(first, replay) {
		t.Fatal("CLI restart changed replay")
	}
	if history := request("GET", path, created.Capability, "", 200); !bytes.Contains(history, []byte(`"seq":2`)) || bytes.Contains(history, []byte(`"seq":3`)) {
		t.Fatal("CLI restart lost or duplicated whole pair", string(history))
	}
}

func TestContractCLIEndpointOverrideGuard(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range []struct {
		name  string
		value string
	}{
		{"relative", "example.org/v1/chat/completions"},
		{"http elsewhere", "http://example.org/v1/chat/completions"},
		{"userinfo", "https://user:pass@example.org/v1/chat/completions"},
		{"fragment", "https://example.org/v1/chat/completions#results"},
		{"query", "https://example.org/v1/chat/completions?api=1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cmd := cliCommand(ctx, []string{
				"STORE_MODE=sqlite",
				"SQLITE_PATH=" + filepath.Join(dir, tc.name, "assistant.db"),
				"OPENAI_BASE_URL=" + tc.value,
			}, "serve")
			out, err := cmd.CombinedOutput()
			if err == nil || !strings.Contains(string(out), "invalid OpenAI endpoint configuration") {
				t.Fatalf("override %q did not fail fast: err=%v output=%s", tc.value, err, out)
			}
		})
	}
}

// importFile writes a reviewed JSONL import file for the CLI tests.
func importFile(t *testing.T, dir, name string, lines ...string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

// importSummaryJSON extracts the bounded JSON summary from CLI output.
func importSummaryJSON(t *testing.T, out []byte) map[string]any {
	t.Helper()
	start := bytes.IndexByte(out, '{')
	end := bytes.LastIndexByte(out, '}')
	if start < 0 || end < start {
		t.Fatalf("no JSON summary in output %s", out)
	}
	var summary map[string]any
	if err := json.Unmarshal(out[start:end+1], &summary); err != nil {
		t.Fatalf("invalid JSON summary %s: %v", out[start:end+1], err)
	}
	return summary
}

// The import command refuses unsafe invocations before any storage or network
// work happens.
func TestContractCLIImportGuards(t *testing.T) {
	dir := t.TempDir()
	records := importFile(t, dir, "records.jsonl",
		`{"id":"faq-one","title":"First reviewed FAQ","text":"First reviewed answer body.","approved":true}`,
		`{"id":"faq-two","title":"Second reviewed FAQ","text":"Second reviewed answer body.","approved":false}`)
	var connections atomic.Int32
	fixture := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("guarded command contacted remote database")
	}))
	fixture.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}
	fixture.StartTLS()
	defer fixture.Close()
	local := []string{"STORE_MODE=sqlite", "SQLITE_PATH=" + filepath.Join(dir, "assistant.db")}
	remote := []string{"STORE_MODE=turso", "TURSO_DATABASE_URL=" + fixture.URL, "TURSO_AUTH_TOKEN=fixture-token"}
	for _, tc := range []struct {
		name string
		env  []string
		args []string
		want string
	}{
		{"import without file", local, []string{"import-faq"}, "import file required"},
		{"import with seed", local, []string{"import-faq", "--file", records, "--seed-demo"}, "invalid command or flags"},
		{"import with positional file", local, []string{"import-faq", records}, "invalid command or flags"},
		{"serve with remote", remote, []string{"serve", "--allow-remote"}, "invalid command or flags"},
		{"unknown command", local, []string{"unknown"}, "invalid command or flags"},
		{"remote import without authorization", remote, []string{"import-faq", "--file", records}, "remote import requires import-faq --allow-remote"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := cliCommand(ctx, tc.env, tc.args...)
			out, err := cmd.CombinedOutput()
			if err == nil || !strings.Contains(string(out), tc.want) {
				t.Fatalf("guard: err=%v output=%s", err, out)
			}
		})
	}
	if connections.Load() != 0 {
		t.Fatalf("guarded commands contacted the remote database %d times", connections.Load())
	}
}

// A preview reports the plan and writes nothing; an applied import stores the
// reviewed corpus and prints the bounded summary.
func TestContractCLIImportPreviewAndApply(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "assistant.db")
	records := importFile(t, dir, "records.jsonl",
		`{"id":"faq-one","title":"First reviewed FAQ","text":"First reviewed answer body.","approved":true}`,
		`{"id":"faq-two","title":"Second reviewed FAQ","text":"Second reviewed answer body.","approved":false}`)
	env := []string{"STORE_MODE=sqlite", "SQLITE_PATH=" + dbPath}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	out, err := cliCommand(ctx, env, "import-faq", "--file", records, "--preview").CombinedOutput()
	if err != nil {
		t.Fatalf("preview failed: err=%v output=%s", err, out)
	}
	summary := importSummaryJSON(t, out)
	if summary["mode"] != "preview" {
		t.Fatalf("unexpected preview summary %s", out)
	}
	plan, ok := summary["plan"].(map[string]any)
	if !ok || plan["inserted"] != float64(1) || plan["rejected"] != float64(1) {
		t.Fatalf("unexpected preview plan %s", out)
	}
	if bytes.Contains(out, []byte("answer body")) {
		t.Fatal("preview printed record text")
	}
	store, closeStore, err := chat.OpenSQLite(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	docs, err := store.ApprovedKnowledge(context.Background())
	if err != nil || len(docs) != 0 {
		t.Fatalf("preview wrote rows %+v %v", docs, err)
	}
	if err = closeStore(); err != nil {
		t.Fatal(err)
	}
	// An approved record without the explicit approval decision is refused.
	out, err = cliCommand(ctx, env, "import-faq", "--file", records).CombinedOutput()
	if err == nil || !strings.Contains(string(out), "import rejected") {
		t.Fatalf("approved record applied without --approve: err=%v output=%s", err, out)
	}
	store, closeStore, err = chat.OpenSQLite(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if docs, err = store.ApprovedKnowledge(context.Background()); err != nil || len(docs) != 0 {
		t.Fatalf("refused import wrote rows %+v %v", docs, err)
	}
	if err = closeStore(); err != nil {
		t.Fatal(err)
	}
	out, err = cliCommand(ctx, env, "import-faq", "--file", records, "--approve").CombinedOutput()
	if err != nil {
		t.Fatalf("import failed: err=%v output=%s", err, out)
	}
	summary = importSummaryJSON(t, out)
	if summary["mode"] != "applied" {
		t.Fatalf("unexpected applied summary %s", out)
	}
	result, ok := summary["result"].(map[string]any)
	if !ok || result["inserted"] != float64(2) || result["updated"] != float64(0) || result["unchanged"] != float64(0) {
		t.Fatalf("unexpected applied result %s", out)
	}
	if bytes.Contains(out, []byte("answer body")) {
		t.Fatal("import summary printed record text")
	}
	store, closeStore, err = chat.OpenSQLite(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = closeStore() }()
	docs, err = store.ApprovedKnowledge(context.Background())
	if err != nil || len(docs) != 1 || docs[0].ID != "faq-one" {
		t.Fatalf("imported corpus %+v %v", docs, err)
	}
}

// The local seed and migration commands keep their previous behavior.
func TestContractCLILocalSeedAndMigrateUnchanged(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "assistant.db")
	env := []string{"STORE_MODE=sqlite", "SQLITE_PATH=" + dbPath}
	for _, command := range []string{"migrate", "seed-demo"} {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := cliCommand(ctx, env, command)
		out, err := cmd.CombinedOutput()
		cancel()
		if err != nil || !strings.Contains(string(out), "explicit local seed or migration completed") {
			t.Fatalf("%s changed: err=%v output=%s", command, err, out)
		}
	}
	store, closeStore, err := chat.OpenSQLite(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = closeStore() }()
	docs, err := store.ApprovedKnowledge(context.Background())
	if err != nil || len(docs) != 10 {
		t.Fatalf("demo seed corpus %d %v", len(docs), err)
	}
}
