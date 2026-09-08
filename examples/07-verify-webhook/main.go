// Example: verify a goldsky-webhook-secret header in constant time.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	goldsky "github.com/tigusigalpa/goldsky-go"
)

func main() {
	expected := os.Getenv("GOLDSKY_WEBHOOK_SECRET")
	if expected == "" {
		log.Fatal("GOLDSKY_WEBHOOK_SECRET is not set; use the secret returned once at webhook creation")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !goldsky.VerifyWebhookRequest(r, expected) {
			http.Error(w, "invalid webhook secret", http.StatusUnauthorized)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid JSON payload", http.StatusBadRequest)
			return
		}
		// Enqueue the event here; keep webhook responses fast and idempotent.
		fmt.Printf("accepted webhook with %d top-level fields\n", len(payload))
		w.WriteHeader(http.StatusNoContent)
	})

	addr := os.Getenv("GOLDSKY_WEBHOOK_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	errCh := make(chan error, 1)
	go func() { errCh <- server.ListenAndServe() }()

	fmt.Printf("listening on http://%s/webhook; press Ctrl+C to stop\n", addr)
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}
}
