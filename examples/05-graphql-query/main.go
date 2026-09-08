// Example: query a private Subgraph GraphQL endpoint.
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
	projectID := os.Getenv("GOLDSKY_PROJECT_ID")
	if projectID == "" {
		log.Fatal("GOLDSKY_PROJECT_ID is not set; use the project id from your Goldsky dashboard")
	}
	subgraphName := os.Getenv("GOLDSKY_SUBGRAPH_NAME")
	if subgraphName == "" {
		log.Fatal("GOLDSKY_SUBGRAPH_NAME is not set")
	}
	versionOrTag := os.Getenv("GOLDSKY_SUBGRAPH_VERSION")
	if versionOrTag == "" {
		log.Fatal("GOLDSKY_SUBGRAPH_VERSION is not set; provide a deployed version or stable tag such as prod")
	}

	client, err := goldsky.NewClient(apiKey)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.GraphQL.QueryPrivate(ctx, projectID, subgraphName, versionOrTag, goldsky.GraphQLRequest{
		Query: "{ _meta { block { number } } }",
	})
	if err != nil {
		log.Fatalf("graphql query: %v", err)
	}
	if resp.HasErrors() {
		for _, e := range resp.Errors {
			fmt.Printf("graphql error: %s\n", e.Message)
		}
		os.Exit(1)
	}

	fmt.Printf("status=%d data=%s\n", resp.Status, string(resp.Data))
}
