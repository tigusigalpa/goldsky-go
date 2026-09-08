// Example: create a pipeline. This is a MUTATION and is guarded by
// GOLDSKY_RUN_MUTATIONS=1 so it cannot run by accident.
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
	if os.Getenv("GOLDSKY_RUN_MUTATIONS") != "1" {
		log.Fatal("refusing to run a mutation: set GOLDSKY_RUN_MUTATIONS=1 to create a pipeline")
	}

	apiKey := os.Getenv("GOLDSKY_API_KEY")
	if apiKey == "" {
		log.Fatal("GOLDSKY_API_KEY is not set; create a project API token in the Goldsky dashboard")
	}

	client, err := goldsky.NewClient(apiKey)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	name := os.Getenv("GOLDSKY_PIPELINE_NAME")
	if name == "" {
		name = "ethereum-blocks-example"
	}

	p, err := client.Pipelines.Create(ctx, goldsky.CreatePipelineRequest{
		Name:         name,
		ResourceSize: "s",
		Description:  "Example pipeline created with goldsky-go",
		Definition: goldsky.PipelineDefinition{
			Sources: map[string]any{
				"ethereum_blocks": map[string]any{
					"type":         "dataset",
					"dataset_name": "ethereum.raw_blocks",
					"version":      "1.0.0",
					"start_at":     "latest",
				},
			},
			Transforms: map[string]any{},
			Sinks: map[string]any{
				"discard": map[string]any{
					"type": "blackhole",
					"from": "ethereum_blocks",
				},
			},
		},
	})
	if err != nil {
		log.Fatalf("create pipeline: %v", err)
	}

	fmt.Printf("created pipeline %s (status=%s)\n", p.Name, p.Status)
}
