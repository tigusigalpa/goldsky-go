// Example: walk all subgraph pages with the cancellation-aware pager.
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

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pager := client.Subgraphs.NewSubgraphPager(goldsky.ListSubgraphsOptions{PageSize: 100})
	count := 0
	for i := 0; ; i++ {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Fatalf("page %d: %v", i, err)
		}
		for _, s := range page.Data {
			count++
			fmt.Printf("  - %s/%s  status=%s health=%s synced=%t\n", s.Name, s.Version, s.Status, s.Health, s.Synced)
		}
		if !page.HasMore() {
			break
		}
	}
	fmt.Printf("Total subgraphs listed: %d\n", count)
}
