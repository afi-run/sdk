// Package afi provides a client for executing token swaps on Base via the AFI Protocol.
package afi

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Client executes swaps on Base via the AFI Protocol.
type Client struct {
	eth    *ethclient.Client
	key    *ecdsa.PrivateKey
	afiABI abi.ABI
	erc20  abi.ABI
	rpcURL string
}

// NewClient creates a new AFI client from the given config.
func NewClient(cfg Config) (*Client, error) {
	if cfg.RPCURL == "" {
		return nil, fmt.Errorf("RPCURL is required")
	}
	if cfg.PrivateKey == "" {
		return nil, fmt.Errorf("PrivateKey is required")
	}

	eth, err := ethclient.Dial(cfg.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("connect to RPC: %w", err)
	}

	key, err := privateKeyFromHex(cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	afiABI, err := abi.JSON(strings.NewReader(afiABIJSON))
	if err != nil {
		return nil, fmt.Errorf("parse afi abi: %w", err)
	}

	erc20ABI, err := abi.JSON(strings.NewReader(erc20ABIJSON))
	if err != nil {
		return nil, fmt.Errorf("parse erc20 abi: %w", err)
	}

	return &Client{
		eth:    eth,
		key:    key,
		afiABI: afiABI,
		erc20:  erc20ABI,
		rpcURL: cfg.RPCURL,
	}, nil
}

// Address returns the wallet address derived from the configured private key.
func (c *Client) Address() common.Address {
	return crypto.PubkeyToAddress(c.key.PublicKey)
}

// GetTokens returns all tokens available for swapping on Base.
// Use this to discover supported tokens before building a swap.
func (c *Client) GetTokens(ctx context.Context) ([]Token, error) {
	return fetchTokens(ctx)
}

// GetFeeBps reads the current protocol fee from the contract (basis points).
func (c *Client) GetFeeBps(ctx context.Context) (uint16, error) {
	input, err := c.afiABI.Pack("feeBps")
	if err != nil {
		return 0, err
	}

	result, err := c.eth.CallContract(ctx, ethereum.CallMsg{To: &AfiAddress, Data: input}, nil)
	if err != nil {
		return 0, fmt.Errorf("read feeBps: %w", err)
	}

	out, err := c.afiABI.Unpack("feeBps", result)
	if err != nil {
		return 0, err
	}
	return out[0].(uint16), nil
}

// GetQuote fetches a quote for the given swap params from the AFI quoter service.
// Returns pricing, route path, minimum output, and the encoded steps needed for execution.
// No on-chain interaction — safe to call freely.
func (c *Client) GetQuote(ctx context.Context, params SwapParams) (*Quote, error) {
	decimals, err := getDecimals(ctx, c.eth, c.erc20, params.TokenIn)
	if err != nil {
		return nil, fmt.Errorf("get decimals: %w", err)
	}

	feeBps, err := c.GetFeeBps(ctx)
	if err != nil {
		return nil, fmt.Errorf("get feeBps: %w", err)
	}

	return fetchQuote(ctx, params, decimals, feeBps, c.rpcURL)
}

// Approve approves exactly amountWei of token to the AFI contract.
// Returns the approval tx hash, or empty string if existing allowance was already sufficient.
// Called automatically by ExecuteSwap() — only use this directly for custom flows.
func (c *Client) Approve(ctx context.Context, token common.Address, amountWei *big.Int) (string, error) {
	return ensureExactApproval(ctx, c.eth, c.erc20, c, token, c.Address(), amountWei)
}

// ExecuteSwap executes a swap from a pre-fetched quote.
//
// Flow: balance check → approve (exact) → simulate → swap
//
// Use this after reviewing a quote from GetQuote().
// Returns an error before sending any tx if the swap would revert.
func (c *Client) ExecuteSwap(ctx context.Context, q *Quote) (*SwapResult, error) {
	from := c.Address()

	if err := assertSufficientBalance(ctx, c.eth, c.erc20, q.TokenIn, from, q.AmountInWei); err != nil {
		return nil, err
	}

	if _, err := ensureExactApproval(ctx, c.eth, c.erc20, c, q.TokenIn, from, q.AmountInWei); err != nil {
		return nil, err
	}

	if err := simulateSwap(ctx, c.eth, c.afiABI, q, from); err != nil {
		return nil, err
	}

	return executeSwap(ctx, c, c.afiABI, q)
}

// Swap is the convenience method that runs the full flow in one call.
// Equivalent to: quote, _ := GetQuote(params); return ExecuteSwap(quote)
func (c *Client) Swap(ctx context.Context, params SwapParams) (*SwapResult, error) {
	quote, err := c.GetQuote(ctx, params)
	if err != nil {
		return nil, err
	}
	return c.ExecuteSwap(ctx, quote)
}

// Close closes the underlying RPC connection.
func (c *Client) Close() { c.eth.Close() }
