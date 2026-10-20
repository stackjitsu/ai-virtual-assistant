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
