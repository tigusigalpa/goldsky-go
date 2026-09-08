// Example: validate a pipeline definition without creating it.
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

	client, err := goldsky.NewClient(apiKey)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := client.Pipelines.Validate(ctx, goldsky.ValidatePipelineRequest{
		Name: "example-pipeline",
		Definition: goldsky.PipelineDefinition{
			Sources:    map[string]any{"src": map[string]any{"type": "ethereum"}},
			Transforms: map[string]any{},
			Sinks:      map[string]any{"out": map[string]any{"type": "postgres"}},
		},
	})
	if err != nil {
		log.Fatalf("validate: %v", err)
	}

	fmt.Printf("valid=%t\n", result.Valid)
	for _, e := range result.Errors {
		fmt.Printf("  error: %s\n", e.Message)
	}
	for _, w := range result.Warnings {
		fmt.Printf("  warning: %s\n", w.Message)
	}
}
