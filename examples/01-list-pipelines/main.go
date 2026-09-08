// Example: construct a client and list one page of pipelines.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	goldsky "github.com/tigusigalpa/goldsky-go"
)

func main() {
	apiKey := os.Getenv("GOLDSKY_API_KEY")
	if apiKey == "" {
		log.Fatal("GOLDSKY_API_KEY is not set; create a project API token in the Goldsky dashboard")
	}

	client, err := goldsky.NewClient(apiKey, goldsky.WithTimeout(30*time.Second))
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	page, err := client.Pipelines.List(ctx, goldsky.ListPipelinesOptions{PageSize: 50})
	if err != nil {
		log.Fatalf("list pipelines: %v", err)
	}

	fmt.Printf("Pipelines (page size %d, more=%t):\n", len(page.Data), page.HasMore())
	for _, p := range page.Data {
		fmt.Printf("  - %s  status=%s\n", p.Name, p.Status)
	}
}
