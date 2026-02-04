package main

import (
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brunobiangulo/goreason"
)

func main() {
	configPath := flag.String("config", "", "Path to config file (JSON)")
	addr := flag.String("addr", ":8080", "Listen address")
	flag.Parse()

	// Structured JSON logging.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg := goreason.DefaultConfig()
	if *configPath != "" {
		f, err := os.Open(*configPath)
		if err != nil {
			slog.Error("opening config", "error", err)
			os.Exit(1)
		}
		if err := json.NewDecoder(f).Decode(&cfg); err != nil {
			f.Close()
			slog.Error("parsing config", "error", err)
			os.Exit(1)
		}
		f.Close()
	}

	// Override from environment variables.
	if v := os.Getenv("GOREASON_DB_PATH"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("GOREASON_CHAT_BASE_URL"); v != "" {
		cfg.Chat.BaseURL = v
	}
	if v := os.Getenv("GOREASON_EMBED_BASE_URL"); v != "" {
		cfg.Embedding.BaseURL = v
	}
	if v := os.Getenv("GOREASON_CHAT_API_KEY"); v != "" {
		cfg.Chat.APIKey = v
	}
	if v := os.Getenv("GOREASON_EMBED_API_KEY"); v != "" {
		cfg.Embedding.APIKey = v
	}
	if v := os.Getenv("GOREASON_CHAT_MODEL"); v != "" {
		cfg.Chat.Model = v
	}
	if v := os.Getenv("GOREASON_EMBED_MODEL"); v != "" {
		cfg.Embedding.Model = v
	}
	if v := os.Getenv("GOREASON_CHAT_PROVIDER"); v != "" {
		cfg.Chat.Provider = v
	}
	if v := os.Getenv("GOREASON_EMBED_PROVIDER"); v != "" {
		cfg.Embedding.Provider = v
	}

	// Fallback: check well-known provider env vars for API keys.
	if cfg.Chat.APIKey == "" {
		switch cfg.Chat.Provider {
		case "openai":
			cfg.Chat.APIKey = os.Getenv("OPENAI_API_KEY")
		case "groq":
			cfg.Chat.APIKey = os.Getenv("GROQ_API_KEY")
		}
	}
	if cfg.Embedding.APIKey == "" {
		switch cfg.Embedding.Provider {
		case "openai":
			cfg.Embedding.APIKey = os.Getenv("OPENAI_API_KEY")
		case "groq":
			cfg.Embedding.APIKey = os.Getenv("GROQ_API_KEY")
		}
	}

	apiKey := os.Getenv("GOREASON_API_KEY")
	corsOrigins := os.Getenv("GOREASON_CORS_ORIGINS")

	engine, err := goreason.New(cfg)
	if err != nil {
		slog.Error("creating engine", "error", err)
		os.Exit(1)
	}
	defer engine.Close()

	h := newHandler(engine)
	mux := http.NewServeMux()

	mux.HandleFunc("POST /ingest", h.handleIngest)
	mux.HandleFunc("POST /query", h.handleQuery)
	mux.HandleFunc("POST /update", h.handleUpdate)
	mux.HandleFunc("POST /update-all", h.handleUpdateAll)
	mux.HandleFunc("DELETE /documents/{id}", h.handleDeleteDocument)
	mux.HandleFunc("GET /documents", h.handleListDocuments)
	mux.HandleFunc("GET /health", h.handleHealth)

	// Middleware chain: recovery -> cors -> auth -> logging -> mux
	var handler http.Handler = mux
	handler = logMiddleware(handler)
	handler = authMiddleware(apiKey, handler)
	handler = corsMiddleware(corsOrigins, handler)
	handler = recoveryMiddleware(handler)

	srv := &http.Server{
		Addr:         *addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // streaming responses (ingest can be long)
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown on SIGTERM/SIGINT.
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("server starting", "addr", *addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-done
	slog.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}

	slog.Info("server stopped")
}
