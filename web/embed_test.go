package web

import (
	"errors"
	"io/fs"
	"testing"
)

func TestContractEmbeddedAssets(t *testing.T) {
	assets, err := fs.Sub(Assets, "assets")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"loader.js", "loader.css", "widget.html", "widget.js", "widget-core.mjs",
		"protocol.mjs", "widget.css", "sample-host.html",
	} {
		data, err := fs.ReadFile(assets, name)
		if err != nil || len(data) == 0 {
			t.Fatalf("embedded browser asset %q missing/empty: %v", name, err)
		}
	}
	if _, err := fs.Stat(Assets, "tests"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("tests must not be public embedded assets: %v", err)
	}
}
