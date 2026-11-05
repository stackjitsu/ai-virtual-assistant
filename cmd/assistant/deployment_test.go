package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestContractCLIPublicDemoParsing(t *testing.T) {
	for _, tc := range []struct {
		value   string
		want    bool
		invalid bool
	}{
		{"", false, false}, {"false", false, false}, {"true", true, false}, {"TRUE", false, true}, {"1", false, true}, {"False", false, true}, {" true", false, true}, {"false\n", false, true}, {"yes", false, true},
	} {
		got, err := parsePublicDemo(tc.value)
		if got != tc.want || (err != nil) != tc.invalid {
			t.Fatalf("parse %q: %v %v", tc.value, got, err)
		}
	}
	// Actual CLI must reject invalid config before opening even a local database.
	dir := t.TempDir()
	db := filepath.Join(dir, "should-not-exist.db")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := cliCommand(ctx, []string{"ASSISTANT_PUBLIC_DEMO=TRUE", "SQLITE_PATH=" + db}, "serve").CombinedOutput()
	if err == nil || !strings.Contains(string(out), "invalid ASSISTANT_PUBLIC_DEMO") {
		t.Fatalf("invalid CLI: %v %s", err, out)
	}
	if _, err = os.Stat(db); !os.IsNotExist(err) {
		t.Fatal("invalid configuration opened storage")
	}
}
