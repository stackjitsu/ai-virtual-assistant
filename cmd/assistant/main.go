// assistant composes the backend and embedded widget. It never loads a .env file.
package main

import (
	"context"
	"flag"
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

func main() {
	// Only a running command reads credential environment variables. Tests/builds
	// never execute main and have no ambient-service or credential dependency.
	command := "serve"
	args := os.Args[1:]
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		command, args = args[0], args[1:]
	}
	flags := flag.NewFlagSet("assistant "+command, flag.ExitOnError)
	seed := flags.Bool("seed-demo", false, "explicitly seed fictional sample knowledge (local SQLite only)")
	allowRemote := flags.Bool("allow-remote", false, "authorize remote schema migration, only with migrate command")
	_ = flags.Parse(args)
	if flags.NArg() != 0 || (command != "serve" && command != "seed-demo" && command != "migrate") || (*allowRemote && command != "migrate") {
		log.Fatal("invalid command or flags")
	}
	mode := env("STORE_MODE", "sqlite")
	var store *chat.SQLStore
	var err error
	if mode == "sqlite" {
		var closeStore func() error
		store, closeStore, err = chat.OpenSQLite(env("SQLITE_PATH", "./data/assistant.db"))
		if err != nil {
			log.Fatal("local storage could not be opened")
		}
		defer func() { _ = closeStore() }()
	} else if mode == "turso" {
		if *seed || command == "seed-demo" {
			log.Fatal("demo seed is local-only")
		}
		store, err = chat.OpenTurso(os.Getenv("TURSO_DATABASE_URL"), os.Getenv("TURSO_AUTH_TOKEN"))
		if err != nil {
			log.Fatal("invalid remote storage configuration")
		}
	} else {
		log.Fatal("invalid STORE_MODE")
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
	handler, err := chat.NewHandler(store, chat.NewOpenAI(os.Getenv("OPENAI_API_KEY"), model), chat.HTTPConfig{AssistantOrigin: origin, ParentOrigins: parents}, assets)
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
