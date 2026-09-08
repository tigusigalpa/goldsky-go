// Example: fetch the chain ID and latest block in one Edge RPC batch.
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
		log.Fatal("GOLDSKY_EDGE_API_KEY is not set")
	}
	chainID := int64(1)
	if raw := os.Getenv("GOLDSKY_CHAIN_ID"); raw != "" {
		var err error
		chainID, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || chainID <= 0 {
			log.Fatalf("GOLDSKY_CHAIN_ID must be a positive decimal integer, got %q", raw)
		}
	}

	client, err := goldsky.NewDataClient(goldsky.WithEdgeAPIKey(edgeKey))
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var remoteChainID, latestBlock string
	responses, err := client.RPC.Batch(ctx, chainID, []goldsky.RPCBatchCall{
		{Method: "eth_chainId", Params: []any{}, Result: &remoteChainID},
		{Method: "eth_blockNumber", Params: []any{}, Result: &latestBlock},
	})
	if err != nil {
		log.Fatalf("batch RPC: %v", err)
	}
	for _, response := range responses {
		if response.Error != nil {
			log.Fatalf("RPC call %d failed: %v", response.ID, response.Error)
		}
	}

	fmt.Printf("chain=%d latest_block=%d\n", mustParseHex(remoteChainID), mustParseHex(latestBlock))
}

func mustParseHex(value string) uint64 {
	n, err := strconv.ParseUint(strings.TrimPrefix(value, "0x"), 16, 64)
	if err != nil {
		log.Fatalf("invalid hex RPC result %q: %v", value, err)
	}
	return n
}
