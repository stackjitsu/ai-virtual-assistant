// assistant composes the backend and embedded widget. It never loads a .env file.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"ai-virtual-assistant/internal/chat"
	"ai-virtual-assistant/web"
)

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// importSummary is the bounded, deterministic result of an import command. It
// reports counts and record identifiers only; record text is never printed.
type importSummary struct {
	Mode   string             `json:"mode"`
	Plan   *chat.ImportPlan   `json:"plan,omitempty"`
	Result *chat.ImportResult `json:"result,omitempty"`
}

// printImportSummary writes the import summary as one deterministic JSON
// document on standard output.
func printImportSummary(summary importSummary) {
	encoded, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		log.Fatal("import summary failed")
	}
	fmt.Println(string(encoded))
}

// parsePublicDemo deliberately accepts only the documented lowercase values.
func parsePublicDemo(value string) (bool, error) {
	switch value {
	case "true":
		return true, nil
	case "", "false":
		return false, nil
	default:
		return false, errors.New("invalid ASSISTANT_PUBLIC_DEMO")
	}
}

func main() {
	// Only a running command reads credential environment variables. Tests/builds
	// never execute main and have no ambient-service or credential dependency.
	command := "serve"
	args := os.Args[1:]
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		command, args = args[0], args[1:]
	}
	flags := flag.NewFlagSet("assistant "+command, flag.ExitOnError)
	seed := flags.Bool("seed-demo", false, "explicitly seed fictional Jitihada FAQ knowledge (local SQLite only)")
	allowRemote := flags.Bool("allow-remote", false, "authorize remote schema migration or remote knowledge import")
	file := flags.String("file", "", "JSONL knowledge import file")
	preview := flags.Bool("preview", false, "plan the knowledge import without writing anything")
	approve := flags.Bool("approve", false, "permit approved:true records in the knowledge import")
	_ = flags.Parse(args)
	if flags.NArg() != 0 || (command != "serve" && command != "seed-demo" && command != "migrate" && command != "import-faq") || (*allowRemote && command != "migrate" && command != "import-faq") || (command == "import-faq" && *seed) {
		log.Fatal("invalid command or flags")
	}
	publicDemo, policyErr := parsePublicDemo(os.Getenv("ASSISTANT_PUBLIC_DEMO"))
	if policyErr != nil {
		log.Fatal("invalid ASSISTANT_PUBLIC_DEMO")
	}
	mode := env("STORE_MODE", "sqlite")
	if mode != "sqlite" && mode != "turso" {
		log.Fatal("invalid STORE_MODE")
	}
	var records []chat.ImportRecord
	if command == "import-faq" {
		if *file == "" {
			log.Fatal("import file required")
		}
		// A remote import is refused before any network I/O happens.
		if mode == "turso" && !*allowRemote {
			log.Fatal("remote import requires import-faq --allow-remote")
		}
		var err error
		records, err = chat.ReadImportFile(*file)
		if err != nil {
			log.Fatal("import file is invalid")
		}
	}
	var store *chat.SQLStore
	var err error
	if mode == "sqlite" {
		var closeStore func() error
		store, closeStore, err = chat.OpenSQLite(env("SQLITE_PATH", "./data/assistant.db"))
		if err != nil {
			log.Fatal("local storage could not be opened")
		}
		defer func() { _ = closeStore() }()
	} else {
		if *seed || command == "seed-demo" {
			log.Fatal("demo seed is local-only")
		}
		store, err = chat.OpenTurso(os.Getenv("TURSO_DATABASE_URL"), os.Getenv("TURSO_AUTH_TOKEN"))
		if err != nil {
			log.Fatal("invalid remote storage configuration")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	if mode == "sqlite" || command == "migrate" {
		if mode == "turso" && !*allowRemote {
			log.Fatal("remote migration requires migrate --allow-remote")
		}
		if store.Migrate(ctx) != nil {
			log.Fatal("schema migration failed")
		}
	}
	if command == "seed-demo" || *seed {
		if store.SeedDemo(ctx) != nil {
			log.Fatal("local demo seed failed")
		}
	}
	if command == "import-faq" {
		// An import may read and write up to 100 records over the network, so
		// it gets its own bounded deadline instead of the start-up deadline.
		cancel()
		importCtx, importCancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer importCancel()
		// A preview never writes: it only reports what the import would do.
		if *preview {
			plan, err := store.PlanImport(importCtx, records, *approve)
			if err != nil {
				log.Fatal("import plan failed")
			}
			printImportSummary(importSummary{Mode: "preview", Plan: &plan})
			return
		}
		result, err := store.ImportKnowledge(importCtx, records, *approve)
		if err != nil {
			if errors.Is(err, chat.ErrInvalid) {
				log.Fatal("import rejected")
			}
			log.Fatal("import failed")
		}
		printImportSummary(importSummary{Mode: "applied", Result: &result})
		return
	}
	cancel()
	if command != "serve" {
		log.Print("explicit local seed or migration completed")
		return
	}
	origin := env("ASSISTANT_ORIGIN", "http://127.0.0.1:8080")
	parents := strings.Fields(env("WIDGET_PARENT_ORIGINS", "http://localhost:8080 http://127.0.0.1:8080"))
	assets, err := fs.Sub(web.Assets, "assets")
	if err != nil {
		log.Fatal("embedded widget unavailable")
	}
	model := env("OPENAI_MODEL", "gpt-4.1-mini")
	if len(model) > 128 || strings.ContainsAny(model, "\r\n\t ") {
		log.Fatal("invalid model configuration")
	}
	// OPENAI_BASE_URL optionally names the complete chat-completions URL the
	// provider POSTs exactly; blank or missing keeps the fixed OpenAI default.
	openai := chat.NewOpenAI(os.Getenv("OPENAI_API_KEY"), model)
	endpoint, err := chat.ValidateOpenAIEndpoint(os.Getenv("OPENAI_BASE_URL"))
	if err != nil {
		log.Fatal("invalid OpenAI endpoint configuration")
	}
	if endpoint != "" {
		openai = chat.NewOpenAIEndpoint(os.Getenv("OPENAI_API_KEY"), model, endpoint)
	}
	handler, err := chat.NewHandler(store, openai, chat.HTTPConfig{PublicDemo: publicDemo, AssistantOrigin: origin, ParentOrigins: parents}, assets)
	if err != nil {
		log.Fatal("invalid HTTP origin configuration")
	}
	addr := env("LISTEN_ADDR", "127.0.0.1:8080")
	if _, _, err = net.SplitHostPort(addr); err != nil {
		log.Fatal("invalid listen address")
	}
	server := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 25 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-signalCtx.Done()
		shutdownCtx, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		_ = server.Shutdown(shutdownCtx)
	}()
	log.Print("assistant listening; AI may be wrong; use no sensitive data; demo knowledge is fictional")
	if err = server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("HTTP server stopped unexpectedly")
	}
}
