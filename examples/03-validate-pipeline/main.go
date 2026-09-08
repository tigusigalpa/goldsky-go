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
		Name: "ethereum-blocks-example",
		Definition: goldsky.PipelineDefinition{
			Sources: map[string]any{
				"ethereum_blocks": map[string]any{
					"type":         "dataset",
					"dataset_name": "ethereum.raw_blocks",
					"version":      "1.0.0",
					"start_at":     "latest",
				},
			},
			Transforms: map[string]any{
				"handle_block": map[string]any{
					"type":        "handler",
					"from":        "ethereum_blocks",
					"primary_key": "id",
					"url":         "https://example.com/handle-block",
				},
			},
			Sinks: map[string]any{
				"discard": map[string]any{
					"type": "blackhole",
					"from": "handle_block",
				},
			},
		},
	})
	if err != nil {
		log.Fatalf("validate: %v", err)
	}

	fmt.Printf("Pipeline definition valid: %t\n", result.Valid)
	for _, e := range result.Errors {
		fmt.Printf("  error (%s): %s\n", e.Field, e.Message)
	}
	for _, w := range result.Warnings {
		fmt.Printf("  warning (%s): %s\n", w.Field, w.Message)
	}
}
