// Example: distinguish API failures from network and decoding failures.
package main

import (
	"context"
	"errors"
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
	name := os.Getenv("GOLDSKY_PIPELINE_NAME")
	if name == "" {
		log.Fatal("GOLDSKY_PIPELINE_NAME is not set; provide the pipeline you want to inspect")
	}

	client, err := goldsky.NewClient(apiKey)
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pipeline, err := client.Pipelines.Get(ctx, name)
	if err == nil {
		fmt.Printf("%s is %s\n", pipeline.Name, pipeline.Status)
		return
	}

	if problem := goldsky.AsProblem(err); problem != nil {
		switch {
		case problem.IsNotFound():
			log.Fatalf("pipeline %q does not exist", name)
		case problem.IsRateLimited():
			if seconds, ok := problem.RetryAfter(); ok {
				log.Fatalf("rate limited; retry after about %d seconds", seconds)
			}
		}
		log.Fatalf("Goldsky rejected the request (status %d, type %s): %s", problem.Status, problem.Type, problem.Detail)
	}

	var transport *goldsky.TransportError
	if errors.As(err, &transport) {
		log.Fatalf("request could not be completed: %v", transport)
	}
	log.Fatal(err)
}
