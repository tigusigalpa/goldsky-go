// Example: call eth_blockNumber over the Edge HTTPS JSON-RPC data plane.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	goldsky "github.com/tigusigalpa/goldsky-go"
)

func main() {
	edgeKey := os.Getenv("GOLDSKY_EDGE_API_KEY")
	if edgeKey == "" {
		log.Fatal("GOLDSKY_EDGE_API_KEY is not set; this is a separate secret from GOLDSKY_API_KEY")
	}
	apiKey := os.Getenv("GOLDSKY_API_KEY")
	if apiKey == "" {
		log.Fatal("GOLDSKY_API_KEY is not set; NewClient requires the REST project token even when this example only calls Edge RPC")
	}
	chainID := int64(1) // Ethereum mainnet; see client.Catalogs.EdgeNetworks() for the list

	client, err := goldsky.NewClient(apiKey, goldsky.WithEdgeAPIKey(edgeKey))
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var block string
	if err := client.RPC.Call(ctx, chainID, "eth_blockNumber", nil, &block); err != nil {
		log.Fatalf("rpc call: %v", err)
	}

	n, err := strconv.ParseUint(strings.TrimPrefix(block, "0x"), 16, 64)
	if err != nil {
		fmt.Printf("raw result: %s\n", block)
	} else {
		fmt.Printf("latest block: %d\n", n)
	}
}
