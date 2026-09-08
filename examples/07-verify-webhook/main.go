// Example: verify a goldsky-webhook-secret header in constant time.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	goldsky "github.com/tigusigalpa/goldsky-go"
)

func main() {
	expected := os.Getenv("GOLDSKY_WEBHOOK_SECRET")
	if expected == "" {
		log.Fatal("GOLDSKY_WEBHOOK_SECRET is not set; use the secret returned once at webhook creation")
	}

	// In a real handler you would use http.HandlerFunc with goldsky.VerifyWebhookRequest.
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if !goldsky.VerifyWebhookRequest(r, expected) {
			http.Error(w, "invalid webhook secret", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "verified")
	})

	addr := os.Getenv("GOLDSKY_WEBHOOK_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	server := &http.Server{
		Addr:              addr,
		Handler:           http.DefaultServeMux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	fmt.Printf("listening on %s; POST to /webhook with header %s\n", addr, goldsky.WebhookSecretHeader)
	log.Fatal(server.ListenAndServe())
}
