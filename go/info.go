package afi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type infoResponse struct {
	Status string                `json:"status"`
	Data   map[string][]rawToken `json:"data"`
}

type rawToken struct {
	Address  string `json:"address"`
	Symbol   string `json:"symbol"`
	Decimals uint8  `json:"decimals"`
	Active   bool   `json:"active"`
}

// fetchTokens fetches available tokens from the AFI info endpoint for the given network.
func fetchTokens(ctx context.Context, network Network, infoURL string) ([]Token, error) {
	return fetchTokensFrom(ctx, network, infoURL)
}

// fetchTokensFrom allows injecting a custom URL — used by tests.
func fetchTokensFrom(ctx context.Context, network Network, url string) ([]Token, error) {
	u := url + "?network=" + string(network)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("build info request: %w", err)
	}

	httpClient := &http.Client{Timeout: 15 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("info request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("info HTTP %d", resp.StatusCode)
	}

	var result infoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode info: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("no %s tokens in info response", network)
	}

	networkTokens, ok := result.Data[string(network)]
	if !ok || len(networkTokens) == 0 {
		return nil, fmt.Errorf("no %s tokens in info response", network)
	}

	tokens := make([]Token, len(networkTokens))
	for i, t := range networkTokens {
		tokens[i] = Token{
			Address:  common.HexToAddress(t.Address),
			Symbol:   t.Symbol,
			Decimals: t.Decimals,
			Active:   t.Active,
		}
	}
	return tokens, nil
}
