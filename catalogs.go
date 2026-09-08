package goldsky

import "context"

// CatalogService lists supported chains, networks, and Edge Data sources.
// These endpoints are also reachable through the Subgraph and Edge services
// for convenience; this service groups the catalog reads together.
type CatalogService struct {
	client *Client
}

// SupportedSubgraphChains lists supported deployment chains. See
// https://api.goldsky.com/api/v1/docs#tag/Catalogs/operation/listSubgraphChains
func (s *CatalogService) SupportedSubgraphChains(ctx context.Context) (SubgraphChainsResponse, error) {
	resp, err := s.client.do(ctx, "GET", []string{"subgraphs", "supported-chains"}, requestOptions{})
	if err != nil {
		return SubgraphChainsResponse{}, err
	}
	var out SubgraphChainsResponse
	if err := decodeJSON(resp.body, &out); err != nil {
		return SubgraphChainsResponse{}, &TransportError{Op: "listSubgraphChains", Err: err}
	}
	return out, nil
}

// EdgeNetworks lists supported Edge networks. See
// https://api.goldsky.com/api/v1/docs#tag/Catalogs/operation/listEdgeNetworks
func (s *CatalogService) EdgeNetworks(ctx context.Context) (EdgeNetworksResponse, error) {
	resp, err := s.client.do(ctx, "GET", []string{"edge", "networks"}, requestOptions{})
	if err != nil {
		return EdgeNetworksResponse{}, err
	}
	var out EdgeNetworksResponse
	if err := decodeJSON(resp.body, &out); err != nil {
		return EdgeNetworksResponse{}, &TransportError{Op: "listEdgeNetworks", Err: err}
	}
	return out, nil
}

// EdgeSources lists Edge Data sources. See
// https://api.goldsky.com/api/v1/docs#tag/Catalogs/operation/listEdgeSources
func (s *CatalogService) EdgeSources(ctx context.Context) (EdgeSourcesResponse, error) {
	resp, err := s.client.do(ctx, "GET", []string{"edge", "sources"}, requestOptions{})
	if err != nil {
		return EdgeSourcesResponse{}, err
	}
	var out EdgeSourcesResponse
	if err := decodeJSON(resp.body, &out); err != nil {
		return EdgeSourcesResponse{}, &TransportError{Op: "listEdgeSources", Err: err}
	}
	return out, nil
}
