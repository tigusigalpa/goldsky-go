// Package goldsky provides a Go client for the Goldsky REST control plane,
// Subgraph GraphQL endpoints, Edge JSON-RPC, and webhook verification.
//
// Use NewClient for control-plane or private GraphQL operations that require a
// project API token. Use NewDataClient for public GraphQL and Edge RPC-only
// applications. A Client is safe to reuse across goroutines; individual pagers
// are stateful and should be consumed by one goroutine.
package goldsky
