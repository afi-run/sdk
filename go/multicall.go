package afi

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Multicall3Call is a single call submitted to Multicall3.aggregate3.
type Multicall3Call struct {
	Target       common.Address `json:"target"`
	AllowFailure bool           `json:"allowFailure"`
	CallData     []byte         `json:"callData"`
}

// Multicall3Result is one entry returned by Multicall3.aggregate3.
type Multicall3Result struct {
	Success    bool   `json:"success"`
	ReturnData []byte `json:"returnData"`
}

// TokenMetadata is the immutable per-token data that the metadata cache holds.
type TokenMetadata struct {
	Symbol   string
	Name     string
	Decimals uint8
}

// metadataCache is a thread-safe address-keyed cache for TokenMetadata.
type metadataCache struct {
	mu sync.RWMutex
	m  map[common.Address]TokenMetadata
}

func newMetadataCache() *metadataCache {
	return &metadataCache{m: make(map[common.Address]TokenMetadata)}
}

func (c *metadataCache) get(token common.Address) (TokenMetadata, bool) {
	if c == nil {
		return TokenMetadata{}, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.m[token]
	return v, ok
}

func (c *metadataCache) set(token common.Address, meta TokenMetadata) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[token] = meta
}

func (c *metadataCache) clear() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m = make(map[common.Address]TokenMetadata)
}

// aggregate3 calls Multicall3.aggregate3 with the given calls.
// Each entry's AllowFailure controls whether an individual revert tanks the batch.
func aggregate3(
	ctx context.Context,
	client *ethclient.Client,
	mcABI abi.ABI,
	calls []Multicall3Call,
) ([]Multicall3Result, error) {
	if len(calls) == 0 {
		return []Multicall3Result{}, nil
	}
	input, err := mcABI.Pack("aggregate3", calls)
	if err != nil {
		return nil, fmt.Errorf("pack aggregate3: %w", err)
	}

	raw, err := client.CallContract(ctx, ethereum.CallMsg{
		To:   &Multicall3Address,
		Data: input,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("multicall aggregate3: %w", err)
	}

	var results []Multicall3Result
	if err := mcABI.UnpackIntoInterface(&results, "aggregate3", raw); err != nil {
		return nil, fmt.Errorf("unpack aggregate3: %w", err)
	}
	return results, nil
}
