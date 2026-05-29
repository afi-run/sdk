// Package afi provides a client for executing token swaps on Base via the AFI Protocol.
package afi

import (
	"context"
	"crypto/ecdsa"
	"errors"
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
	apiURL string // base URL for AFI API calls
}

// NewClient creates a new AFI client from the given config.
// PrivateKey is optional — omit it to create a read-only client.
func NewClient(cfg Config) (*Client, error) {
	if cfg.RPCURL == "" {
		return nil, fmt.Errorf("RPCURL is required")
	}

	eth, err := ethclient.Dial(cfg.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("connect to RPC: %w", err)
	}

	var key *ecdsa.PrivateKey
	if cfg.PrivateKey != "" {
		key, err = privateKeyFromHex(cfg.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
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
		apiURL: APIBaseURL,
	}, nil
}

func (c *Client) quoterURL() string { return c.apiURL + "/quoter" }
func (c *Client) infoURL() string   { return c.apiURL + "/info" }

// SetApiURL changes the base URL used for API calls (default: https://rpc.afi.run).
// Returns c for chaining.
func (c *Client) SetApiURL(url string) *Client {
	c.apiURL = url
	return c
}

// Connect sets the private key on an existing client, enabling signing operations.
func (c *Client) Connect(privateKey string) error {
	key, err := privateKeyFromHex(privateKey)
	if err != nil {
		return fmt.Errorf("parse private key: %w", err)
	}
	c.key = key
	return nil
}

// requireSigner returns ErrNoSigner if no private key has been set.
func (c *Client) requireSigner() error {
	if c.key == nil {
		return ErrNoSigner()
	}
	return nil
}

// Address returns the wallet address derived from the configured private key.
// Returns the zero address if no private key is set.
func (c *Client) Address() common.Address {
	if c.key == nil {
		return common.Address{}
	}
	return crypto.PubkeyToAddress(c.key.PublicKey)
}

// GetTokens returns tokens available for swapping.
// Pass a network to filter (e.g. NetworkBase, NetworkBSC).
func (c *Client) GetTokens(ctx context.Context, network ...Network) ([]Token, error) {
	net := NetworkBase
	if len(network) > 0 {
		net = network[0]
	}
	return fetchTokens(ctx, net, c.infoURL())
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

// GetQuote fetches a price quote.
//
//	quote, err := client.GetQuote(ctx,
//	    afi.From(usdc, weth, "1000"),
//	    afi.WithSlippage(0.5),
//	)
func (c *Client) GetQuote(ctx context.Context, opts ...QuoteOption) (*Quote, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}
	if o.err != nil {
		return nil, o.err
	}
	if o.tokenIn == "" || o.tokenOut == "" || o.amountIn == "" {
		return nil, fmt.Errorf("From() is required")
	}
	// use client rpcURL as default RPC if no custom rpcUrls provided
	if len(o.rpcUrls) == 0 {
		o.rpcUrls = []RpcUrlInfo{{URL: c.rpcURL}}
	}
	feeBps, err := c.GetFeeBps(ctx)
	if err != nil {
		return nil, fmt.Errorf("get feeBps: %w", err)
	}
	return fetchQuote(ctx, o, feeBps, c.quoterURL())
}

// Approve approves exactly amountWei of token to the AFI contract.
// Returns a PendingTx (with TxHash immediately) or nil if the allowance was already sufficient.
// Called automatically by ExecuteSwap() — only use this directly for custom flows.
func (c *Client) Approve(ctx context.Context, token common.Address, amountWei *big.Int) (*PendingTx, error) {
	if err := c.requireSigner(); err != nil {
		return nil, err
	}
	return submitApprove(ctx, c.eth, c.erc20, c, token, c.Address(), amountWei)
}

// Simulate dry-runs the swap via eth_call to check if it would succeed.
// Returns (true, nil) if the simulation passes, (false, nil) if it reverts,
// or (false, err) for unexpected errors.
// An optional log function receives the revert message when simulation fails.
func (c *Client) Simulate(ctx context.Context, q *Quote, log ...func(string)) (bool, error) {
	if err := c.requireSigner(); err != nil {
		return false, err
	}
	from := c.Address()
	if err := simulateSwap(ctx, c.eth, c.afiABI, q, from); err != nil {
		var afiErr *AfiError
		if errors.As(err, &afiErr) && afiErr.Code == "SIMULATION_FAILED" {
			if len(log) > 0 && log[0] != nil {
				log[0](afiErr.Message)
			}
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// SubmitSwap sends the swap transaction without waiting for it to be mined.
// Returns a PendingSwap whose TxHash is immediately available.
// Call Wait() on the result to block until confirmed and get the SwapResult.
func (c *Client) SubmitSwap(ctx context.Context, q *Quote) (*PendingSwap, error) {
	if err := c.requireSigner(); err != nil {
		return nil, err
	}
	return submitSwap(ctx, c, c.afiABI, q)
}

// ExecuteSwap executes a swap from a pre-fetched quote.
//
// Flow: balance check → approve (exact) → simulate → swap → wait for confirmation
//
// Use this after reviewing a quote from GetQuote().
// Returns an error before sending any tx if the swap would revert.
func (c *Client) ExecuteSwap(ctx context.Context, q *Quote) (*SwapResult, error) {
	if err := c.requireSigner(); err != nil {
		return nil, err
	}
	from := c.Address()

	if err := assertSufficientBalance(ctx, c.eth, c.erc20, q.TokenIn, from, q.AmountInWei); err != nil {
		return nil, err
	}

	if _, err := ensureExactApproval(ctx, c.eth, c.erc20, c, q.TokenIn, from, q.AmountInWei); err != nil {
		return nil, err
	}

	var simulationMsg string
	ok, err := c.Simulate(ctx, q, func(msg string) { simulationMsg = msg })
	if err != nil {
		return nil, err
	}
	if !ok {
		if simulationMsg == "" {
			simulationMsg = "simulation failed"
		}
		return nil, ErrSimulation(simulationMsg)
	}

	pending, err := submitSwap(ctx, c, c.afiABI, q)
	if err != nil {
		return nil, err
	}
	return pending.Wait(ctx)
}

// Swap is the convenience method: equivalent to GetQuote(...) then ExecuteSwap(quote).
func (c *Client) Swap(ctx context.Context, opts ...QuoteOption) (*SwapResult, error) {
	quote, err := c.GetQuote(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return c.ExecuteSwap(ctx, quote)
}

// Close closes the underlying RPC connection.
func (c *Client) Close() { c.eth.Close() }
