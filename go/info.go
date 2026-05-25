package afi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

const infoURL = "https://rpc.afi.run/info"

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

// fetchTokens fetches available tokens from the AFI info endpoint.
func fetchTokens(ctx context.Context) ([]Token, error) {
	return fetchTokensFrom(ctx, infoURL)
}

// fetchTokensFrom allows injecting a custom URL — used by tests.
func fetchTokensFrom(ctx context.Context, url string) ([]Token, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
		return nil, fmt.Errorf("no base tokens in info response")
	}

	baseTokens, ok := result.Data["base"]
	if !ok || len(baseTokens) == 0 {
		return nil, fmt.Errorf("no base tokens in info response")
	}

	tokens := make([]Token, len(baseTokens))
	for i, t := range baseTokens {
		tokens[i] = Token{
			Address:  common.HexToAddress(t.Address),
			Symbol:   t.Symbol,
			Decimals: t.Decimals,
			Active:   t.Active,
		}
	}
	return tokens, nil
}
