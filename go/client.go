// Package afi provides a client for executing token swaps on Base via the AFI Protocol.
package afi

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Client executes swaps on Base via the AFI Protocol.
type Client struct {
	eth          *ethclient.Client
	key          *ecdsa.PrivateKey
	afiABI       abi.ABI
	erc20        abi.ABI
	multicall3   abi.ABI
	rpcURL       string
	apiURL       string // base URL for AFI API calls
	gasBufferPct uint
	logger       Logger

	tokensMu    sync.Mutex
	tokensCache map[Network][]Token

	chainIDMu sync.Mutex
	chainID   *big.Int

	metaCache *metadataCache

	nonceMu      sync.Mutex
	managedNonce bool
	localNonce   uint64
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

	mcABI, err := abi.JSON(strings.NewReader(multicall3ABIJSON))
	if err != nil {
		return nil, fmt.Errorf("parse multicall3 abi: %w", err)
	}

	gasBuffer := cfg.GasBufferPercent
	if gasBuffer == 0 {
		gasBuffer = DefaultGasBufferPercent
	}

	return &Client{
		eth:          eth,
		key:          key,
		afiABI:       afiABI,
		erc20:        erc20ABI,
		multicall3:   mcABI,
		rpcURL:       cfg.RPCURL,
		apiURL:       APIBaseURL,
		gasBufferPct: gasBuffer,
		logger:       cfg.Logger,
		tokensCache:  make(map[Network][]Token),
		metaCache:    newMetadataCache(),
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

// SetGasBufferPercent overrides the default gas buffer applied on top of estimated gas
// for write txs (approve, swap). Returns c for chaining.
func (c *Client) SetGasBufferPercent(percent uint) *Client {
	c.gasBufferPct = percent
	return c
}

// GasBufferPercent returns the configured gas buffer percentage.
func (c *Client) GasBufferPercent() uint { return c.gasBufferPct }

// SetLogger attaches (or replaces) the diagnostic logger. Pass nil to disable.
// Returns c for chaining.
func (c *Client) SetLogger(logger Logger) *Client {
	c.logger = logger
	return c
}

// ChainID returns the chain ID reported by the RPC, cached on first call.
// Use this to detect mismatches between the configured network and the RPC URL.
func (c *Client) ChainID(ctx context.Context) (*big.Int, error) {
	c.chainIDMu.Lock()
	if c.chainID != nil {
		id := new(big.Int).Set(c.chainID)
		c.chainIDMu.Unlock()
		return id, nil
	}
	c.chainIDMu.Unlock()

	id, err := c.eth.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("read chain id: %w", err)
	}

	c.chainIDMu.Lock()
	c.chainID = new(big.Int).Set(id)
	c.chainIDMu.Unlock()
	return id, nil
}

// DetectNetwork returns the Network the RPC is connected to, or an empty string
// if the chain ID doesn't match any known network.
func (c *Client) DetectNetwork(ctx context.Context) (Network, error) {
	id, err := c.ChainID(ctx)
	if err != nil {
		return "", err
	}
	for net, cid := range NetworkChainIDs {
		if id.Int64() == cid {
			return net, nil
		}
	}
	return "", nil
}

// WaitForTx blocks until the transaction `hash` reaches the requested number of confirmations
// (defaults to 1). Use this for hashes obtained outside the SDK (e.g. persisted from a prior run).
func (c *Client) WaitForTx(ctx context.Context, hash string, opts ...WaitForTxOptions) (*TxReceipt, error) {
	o := WaitForTxOptions{Confirmations: 1, PollIntervalMs: 500}
	if len(opts) > 0 {
		if opts[0].Confirmations > 0 {
			o.Confirmations = opts[0].Confirmations
		}
		if opts[0].PollIntervalMs > 0 {
			o.PollIntervalMs = opts[0].PollIntervalMs
		}
		o.TimeoutMs = opts[0].TimeoutMs
	}

	if o.TimeoutMs > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(o.TimeoutMs)*time.Millisecond)
		defer cancel()
	}

	hashH := common.HexToHash(hash)
	for {
		receipt, err := c.eth.TransactionReceipt(ctx, hashH)
		if err == nil {
			if o.Confirmations <= 1 {
				return receiptToTxReceipt(receipt), nil
			}
			head, err := c.eth.HeaderByNumber(ctx, nil)
			if err != nil {
				return nil, fmt.Errorf("get head: %w", err)
			}
			confs := head.Number.Uint64() - receipt.BlockNumber.Uint64() + 1
			if confs >= o.Confirmations {
				return receiptToTxReceipt(receipt), nil
			}
		} else if !errors.Is(err, ethereum.NotFound) {
			return nil, fmt.Errorf("get receipt: %w", err)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(o.PollIntervalMs) * time.Millisecond):
		}
	}
}

// receiptToTxReceipt converts a chain receipt into the SDK TxReceipt with fee info.
func receiptToTxReceipt(r *types.Receipt) *TxReceipt {
	eff, feeWei, feeEth := feeFromReceipt(r)
	return &TxReceipt{
		BlockNumber:       r.BlockNumber.Uint64(),
		GasUsed:           r.GasUsed,
		EffectiveGasPrice: eff,
		FeeWei:            feeWei,
		FeeETH:            feeEth,
	}
}

// logged wraps a function call with logger emission. The logger is invoked even when fn returns an error.
func (c *Client) logged(method string, fn func() error) error {
	if c.logger == nil {
		return fn()
	}
	start := timeNowMS()
	err := fn()
	c.logger(LogEvent{
		Kind:       "rpc",
		Method:     method,
		DurationMs: timeNowMS() - start,
		OK:         err == nil,
		Err:        err,
	})
	return err
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
// Pass a network to filter (e.g. NetworkBase, NetworkBSC). Results are cached per-network —
// use ClearTokensCache to force a refresh.
func (c *Client) GetTokens(ctx context.Context, network ...Network) ([]Token, error) {
	net := NetworkBase
	if len(network) > 0 {
		net = network[0]
	}
	c.tokensMu.Lock()
	if cached, ok := c.tokensCache[net]; ok {
		c.tokensMu.Unlock()
		return cached, nil
	}
	c.tokensMu.Unlock()

	tokens, err := fetchTokens(ctx, net, c.infoURL())
	if err != nil {
		return nil, err
	}

	c.tokensMu.Lock()
	c.tokensCache[net] = tokens
	c.tokensMu.Unlock()
	return tokens, nil
}

// ClearTokensCache invalidates the in-memory GetTokens cache.
// Pass a network to clear just that one, or omit to clear all.
func (c *Client) ClearTokensCache(network ...Network) {
	c.tokensMu.Lock()
	defer c.tokensMu.Unlock()
	if len(network) > 0 {
		delete(c.tokensCache, network[0])
		return
	}
	c.tokensCache = make(map[Network][]Token)
}

// FindToken looks up a token by symbol (case-insensitive) on `network` (default: Base).
// Uses the GetTokens cache. Returns nil when no active token matches.
func (c *Client) FindToken(ctx context.Context, symbol string, network ...Network) (*Token, error) {
	tokens, err := c.GetTokens(ctx, network...)
	if err != nil {
		return nil, err
	}
	target := strings.ToLower(symbol)
	for i := range tokens {
		if strings.ToLower(tokens[i].Symbol) == target {
			return &tokens[i], nil
		}
	}
	return nil, nil
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

// TokenInfo fetches symbol/name/decimals — plus balance and allowance against the AFI contract
// when owner is non-zero — in a single multicall.
// Pass common.Address{} as owner to skip balance/allowance.
func (c *Client) TokenInfo(ctx context.Context, token, owner common.Address) (*TokenInfo, error) {
	return fetchTokenInfo(ctx, c.eth, c.erc20, c.multicall3, token, owner, c.metaCache)
}

// MyTokenInfo is shorthand for TokenInfo using the connected wallet as owner.
// Requires a signer.
func (c *Client) MyTokenInfo(ctx context.Context, token common.Address) (*TokenInfo, error) {
	if err := c.requireSigner(); err != nil {
		return nil, err
	}
	return fetchTokenInfo(ctx, c.eth, c.erc20, c.multicall3, token, c.Address(), c.metaCache)
}

// TokenInfoBatch fetches metadata (+ optional balance/allowance) for N tokens in
// a single multicall. Pass common.Address{} as owner to skip balance/allowance.
// Order of the result matches the input.
func (c *Client) TokenInfoBatch(ctx context.Context, tokens []common.Address, owner common.Address) ([]*TokenInfo, error) {
	return fetchTokenInfoBatch(ctx, c.eth, c.erc20, c.multicall3, tokens, owner, c.metaCache)
}

// ClearTokenMetadataCache wipes the in-memory symbol/name/decimals cache.
// Useful when proxying through a custom RPC that may serve stale token metadata.
func (c *Client) ClearTokenMetadataCache() { c.metaCache.clear() }

// Multicall is a generic Multicall3 wrapper. Bundles N read calls into a single
// RPC round-trip. Each call's AllowFailure controls whether an individual revert
// tanks the batch.
//
// Use this to do arbitrary contract reads beyond what the SDK exposes — pool
// prices, custom DEX state, your own contracts, etc.
func (c *Client) Multicall(ctx context.Context, calls []Multicall3Call) ([]Multicall3Result, error) {
	return aggregate3(ctx, c.eth, c.multicall3, calls)
}

// Revoke sends approve(AFI, 0) for `token`, zeroing the router's allowance.
// Returns nil when the allowance is already zero. Use as a post-swap security cleanup.
func (c *Client) Revoke(ctx context.Context, token common.Address) (*PendingTx, error) {
	if err := c.requireSigner(); err != nil {
		return nil, err
	}
	return submitRevoke(ctx, c.eth, c.erc20, c, token, c.Address())
}

// RefreshQuote re-fetches a quote using the same logical request as a stale one.
// All quote parameters (network, slippage, maxHops, priceBase, dexs) are reused —
// only the live route, prices and timestamps are refreshed.
//
//	if quote.IsStale(30) {
//	    quote, _ = client.RefreshQuote(ctx, quote)
//	}
func (c *Client) RefreshQuote(ctx context.Context, q *Quote) (*Quote, error) {
	opts := []QuoteOption{
		From(q.TokenIn, q.TokenOut, q.AmountIn),
		WithSlippage(q.Slippage),
	}
	if q.Network != "" {
		opts = append(opts, OnNetwork(q.Network))
	}
	if q.MaxHops > 0 {
		opts = append(opts, WithMaxHops(q.MaxHops))
	}
	if q.PriceBase != "" {
		opts = append(opts, WithPriceBase(q.PriceBase))
	}
	if len(q.Dexs) > 0 {
		opts = append(opts, WithDexs(q.Dexs...))
	}
	return c.GetQuote(ctx, opts...)
}

// EncodeSwap is a Client-bound convenience for the standalone EncodeSwap helper.
func (c *Client) EncodeSwap(q *Quote) (*EncodedTx, error) { return EncodeSwap(q) }

// EncodeApprove is a Client-bound convenience for the standalone EncodeApprove helper.
func (c *Client) EncodeApprove(token common.Address, amountWei *big.Int) (*EncodedTx, error) {
	return EncodeApprove(token, amountWei)
}

// EncodeRevoke is a Client-bound convenience for the standalone EncodeRevoke helper.
func (c *Client) EncodeRevoke(token common.Address) (*EncodedTx, error) { return EncodeRevoke(token) }

// ─── Nonce management ────────────────────────────────────────────────────────

// GetNonce reads the pending nonce of the connected wallet from the RPC.
func (c *Client) GetNonce(ctx context.Context) (uint64, error) {
	if err := c.requireSigner(); err != nil {
		return 0, err
	}
	return c.eth.PendingNonceAt(ctx, c.Address())
}

// UseManagedNonce enables managed-nonce mode. The SDK fetches the current
// pending nonce once and then maintains a local counter incremented atomically
// for each write tx. Use this for bots that submit multiple swaps in parallel
// without waiting between them.
func (c *Client) UseManagedNonce(ctx context.Context) error {
	n, err := c.GetNonce(ctx)
	if err != nil {
		return err
	}
	c.nonceMu.Lock()
	defer c.nonceMu.Unlock()
	c.localNonce = n
	c.managedNonce = true
	return nil
}

// DisableManagedNonce reverts to per-call RPC queries.
func (c *Client) DisableManagedNonce() {
	c.nonceMu.Lock()
	defer c.nonceMu.Unlock()
	c.managedNonce = false
	c.localNonce = 0
}

// ResetManagedNonce re-syncs the local counter from the chain.
func (c *Client) ResetManagedNonce(ctx context.Context) error {
	c.nonceMu.Lock()
	enabled := c.managedNonce
	c.nonceMu.Unlock()
	if !enabled {
		return nil
	}
	n, err := c.GetNonce(ctx)
	if err != nil {
		return err
	}
	c.nonceMu.Lock()
	c.localNonce = n
	c.nonceMu.Unlock()
	return nil
}

// allocateNonce returns the nonce to use for the next write tx.
func (c *Client) allocateNonce(ctx context.Context, override *uint64, from common.Address) (uint64, error) {
	if override != nil {
		return *override, nil
	}
	c.nonceMu.Lock()
	if c.managedNonce {
		n := c.localNonce
		c.localNonce++
		c.nonceMu.Unlock()
		return n, nil
	}
	c.nonceMu.Unlock()
	return c.eth.PendingNonceAt(ctx, from)
}

// ─── Tx introspection ────────────────────────────────────────────────────────

// GetTxStatus is a non-blocking status check for an arbitrary tx hash.
func (c *Client) GetTxStatus(ctx context.Context, hash string) (TxStatus, error) {
	h := common.HexToHash(hash)
	receipt, err := c.eth.TransactionReceipt(ctx, h)
	if err == nil {
		if receipt.Status == types.ReceiptStatusSuccessful {
			return TxStatusSuccess, nil
		}
		return TxStatusFailed, nil
	}
	if !errors.Is(err, ethereum.NotFound) {
		return "", err
	}
	if _, _, err := c.eth.TransactionByHash(ctx, h); err == nil {
		return TxStatusPending, nil
	} else if errors.Is(err, ethereum.NotFound) {
		return TxStatusUnknown, nil
	} else {
		return "", err
	}
}

// ─── Token pricing ───────────────────────────────────────────────────────────

// TokenPriceOptions controls GetTokenPrice behavior.
type TokenPriceOptions struct {
	Network  Network // default: NetworkBase
	Amount   string  // default: "1"
	Slippage float64 // default: 0.5
}

// GetTokenPrice fetches the live exchange rate for a token pair via the quoter.
func (c *Client) GetTokenPrice(
	ctx context.Context,
	tokenIn, tokenOut common.Address,
	opts ...TokenPriceOptions,
) (*TokenPrice, error) {
	o := TokenPriceOptions{Network: NetworkBase, Amount: "1", Slippage: 0.5}
	if len(opts) > 0 {
		v := opts[0]
		if v.Network != "" {
			o.Network = v.Network
		}
		if v.Amount != "" {
			o.Amount = v.Amount
		}
		if v.Slippage > 0 {
			o.Slippage = v.Slippage
		}
	}
	q, err := c.GetQuote(ctx,
		From(tokenIn, tokenOut, o.Amount),
		WithSlippage(o.Slippage),
		OnNetwork(o.Network),
	)
	if err != nil {
		return nil, err
	}
	return &TokenPrice{Price: q.TokenInPrice, Inverse: q.TokenOutPrice}, nil
}

// ─── Preflight ───────────────────────────────────────────────────────────────

// Preflight runs balance + allowance + simulation checks without sending any tx.
// Use it to drive a UI "ready to swap" indicator before the user commits.
func (c *Client) Preflight(ctx context.Context, q *Quote) (*PreflightReport, error) {
	if err := c.requireSigner(); err != nil {
		return nil, err
	}
	owner := c.Address()
	balance, err := getBalance(ctx, c.eth, c.erc20, q.TokenIn, owner)
	if err != nil {
		return nil, err
	}
	allowance, err := getAllowance(ctx, c.eth, c.erc20, q.TokenIn, owner)
	if err != nil {
		return nil, err
	}

	report := &PreflightReport{
		Balance:       balance,
		Allowance:     allowance,
		NeedsApproval: allowance.Cmp(q.AmountInWei) < 0,
	}
	if balance.Cmp(q.AmountInWei) < 0 {
		report.Problems = append(report.Problems, PreflightProblem{
			Code:    "INSUFFICIENT_BALANCE",
			Message: fmt.Sprintf("have %s, need %s", balance, q.AmountInWei),
		})
	}

	// Skip simulate when balance or allowance is missing — it would just duplicate noise.
	if balance.Cmp(q.AmountInWei) >= 0 && !report.NeedsApproval {
		if err := simulateSwap(ctx, c.eth, c.afiABI, q, owner); err != nil {
			var afiErr *AfiError
			if errors.As(err, &afiErr) && afiErr.Code == "SIMULATION_FAILED" {
				report.Problems = append(report.Problems, PreflightProblem{
					Code: "SIMULATION_FAILED", Message: afiErr.Message,
				})
			} else {
				return nil, err
			}
		}
	}

	report.CanExecute = len(report.Problems) == 0
	return report, nil
}

// GetETHBalance returns the native ETH balance of owner (defaults to the connected wallet).
func (c *Client) GetETHBalance(ctx context.Context, owner ...common.Address) (*big.Int, error) {
	addr, err := c.resolveOwner(owner...)
	if err != nil {
		return nil, err
	}
	return c.eth.BalanceAt(ctx, addr, nil)
}

// TxURL returns the explorer URL for a transaction hash on `network` (default: Base).
func (c *Client) TxURL(hash string, network ...Network) (string, error) {
	return TxURL(hash, networkOrBase(network...))
}

// AddressURL returns the explorer URL for an address on `network` (default: Base).
func (c *Client) AddressURL(address string, network ...Network) (string, error) {
	return AddressURL(address, networkOrBase(network...))
}

func networkOrBase(network ...Network) Network {
	if len(network) > 0 {
		return network[0]
	}
	return NetworkBase
}

// GetAllowance reads how much `token` the AFI contract is allowed to spend on behalf of owner.
// If owner is the zero address, the connected wallet is used (requires a signer).
func (c *Client) GetAllowance(ctx context.Context, token common.Address, owner ...common.Address) (*big.Int, error) {
	addr, err := c.resolveOwner(owner...)
	if err != nil {
		return nil, err
	}
	return getAllowance(ctx, c.eth, c.erc20, token, addr)
}

// HasAllowance returns true when the AFI contract already has at least `amountWei` of allowance
// from owner (or the connected wallet if owner is omitted). Use this to skip Approve().
func (c *Client) HasAllowance(ctx context.Context, token common.Address, amountWei *big.Int, owner ...common.Address) (bool, error) {
	current, err := c.GetAllowance(ctx, token, owner...)
	if err != nil {
		return false, err
	}
	return current.Cmp(amountWei) >= 0, nil
}

// GetBalance reads the ERC20 balance of owner (or the connected wallet if omitted).
func (c *Client) GetBalance(ctx context.Context, token common.Address, owner ...common.Address) (*big.Int, error) {
	addr, err := c.resolveOwner(owner...)
	if err != nil {
		return nil, err
	}
	return getBalance(ctx, c.eth, c.erc20, token, addr)
}

func (c *Client) resolveOwner(owner ...common.Address) (common.Address, error) {
	if len(owner) > 0 && owner[0] != (common.Address{}) {
		return owner[0], nil
	}
	if err := c.requireSigner(); err != nil {
		return common.Address{}, err
	}
	return c.Address(), nil
}

// GetQuote fetches a price quote.
//
//	quote, err := client.GetQuote(ctx,
//	    afi.From(usdc, weth, "1000"),
//	    afi.WithSlippage(0.5),
//	)
func (c *Client) GetQuote(ctx context.Context, opts ...QuoteOption) (*Quote, error) {
	var quote *Quote
	err := c.logged("GetQuote", func() error {
		o := defaultOptions()
		for _, opt := range opts {
			opt(o)
		}
		if o.err != nil {
			return o.err
		}
		if o.tokenIn == "" || o.tokenOut == "" || o.amountIn == "" {
			return fmt.Errorf("From() is required")
		}
		if len(o.rpcUrls) == 0 {
			o.rpcUrls = []RpcUrlInfo{{URL: c.rpcURL}}
		}
		feeBps, err := c.GetFeeBps(ctx)
		if err != nil {
			return fmt.Errorf("get feeBps: %w", err)
		}
		q, err := fetchQuote(ctx, o, feeBps, c.quoterURL())
		if err != nil {
			return err
		}
		quote = q
		return nil
	})
	return quote, err
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

// Simulate dry-runs the swap via eth_call. Returns nil when the swap would succeed,
// or an *AfiError with Code "SIMULATION_FAILED" carrying the revert reason.
func (c *Client) Simulate(ctx context.Context, q *Quote) error {
	if err := c.requireSigner(); err != nil {
		return err
	}
	return simulateSwap(ctx, c.eth, c.afiABI, q, c.Address())
}

// SubmitSwap sends the swap transaction without waiting for it to be mined.
// Returns a PendingSwap whose TxHash is immediately available.
// Call Wait() on the result to block until confirmed and get the SwapResult.
func (c *Client) SubmitSwap(ctx context.Context, q *Quote) (*PendingSwap, error) {
	if err := c.requireSigner(); err != nil {
		return nil, err
	}
	return submitSwap(ctx, c, c.afiABI, q, c.gasBufferPct, nil)
}

// ExecuteSwap executes a swap from a pre-fetched quote.
//
// Flow: balance check → approve (exact) → simulate → swap → wait for confirmation
//
// Use this after reviewing a quote from GetQuote().
// Returns an error before sending any tx if the swap would revert.
// Pass ExecuteOptions to require more than 1 confirmation or set a timeout.
func (c *Client) ExecuteSwap(ctx context.Context, q *Quote, opts ...ExecuteOptions) (*SwapResult, error) {
	var eo ExecuteOptions
	if len(opts) > 0 {
		eo = opts[0]
	}
	waitOpts := WaitForTxOptions{Confirmations: eo.Confirmations, TimeoutMs: eo.TimeoutMs}

	var result *SwapResult
	err := c.logged("ExecuteSwap", func() error {
		if err := c.requireSigner(); err != nil {
			return err
		}
		from := c.Address()
		if err := assertSufficientBalance(ctx, c.eth, c.erc20, c.multicall3, q.TokenIn, from, q.AmountInWei); err != nil {
			return err
		}
		if _, err := ensureExactApproval(ctx, c.eth, c.erc20, c, q.TokenIn, from, q.AmountInWei); err != nil {
			return err
		}
		if err := c.Simulate(ctx, q); err != nil {
			return err
		}
		pending, err := submitSwap(ctx, c, c.afiABI, q, c.gasBufferPct, eo.Nonce)
		if err != nil {
			return err
		}
		r, err := pending.Wait(ctx, waitOpts)
		if err != nil {
			return err
		}
		result = r
		return nil
	})
	return result, err
}

// Swap is the convenience method: equivalent to GetQuote(...) then ExecuteSwap(quote).
// Use ExecuteSwap directly if you need ExecuteOptions (confirmations, timeout).
func (c *Client) Swap(ctx context.Context, opts ...QuoteOption) (*SwapResult, error) {
	quote, err := c.GetQuote(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return c.ExecuteSwap(ctx, quote)
}

// EstimateSwapCost projects the gas cost of executing the quote (without sending a tx).
// Requires a signer because eth_estimateGas needs a from address.
func (c *Client) EstimateSwapCost(ctx context.Context, q *Quote) (*SwapCostEstimate, error) {
	if err := c.requireSigner(); err != nil {
		return nil, err
	}
	from := c.Address()

	input, err := c.afiABI.Pack("swap", q.TokenIn, q.AmountInWei, q.TokenOut, q.MinOutWei, q.Steps)
	if err != nil {
		return nil, fmt.Errorf("pack swap: %w", err)
	}
	gas, err := c.eth.EstimateGas(ctx, ethereum.CallMsg{From: from, To: &AfiAddress, Data: input})
	if err != nil {
		return nil, fmt.Errorf("estimate gas: %w", err)
	}
	gasWithBuf := applyGasBuffer(gas, c.gasBufferPct)

	tip, err := c.eth.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, fmt.Errorf("gas tip: %w", err)
	}
	head, err := c.eth.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("header: %w", err)
	}
	maxFee := new(big.Int).Add(new(big.Int).Mul(head.BaseFee, big.NewInt(2)), tip)
	totalWei := new(big.Int).Mul(maxFee, new(big.Int).SetUint64(gasWithBuf))

	return &SwapCostEstimate{
		Gas:           gas,
		GasWithBuffer: gasWithBuf,
		GasPriceWei:   maxFee,
		TotalWei:      totalWei,
		TotalETH:      formatAmount(totalWei, 18),
	}, nil
}

// Health probes both the RPC (chainId) and the AFI API (/info?network=base) and
// reports the per-endpoint status. Useful at startup to fail fast.
func (c *Client) Health(ctx context.Context) HealthCheck {
	type endpointResult struct {
		ok       bool
		duration int64
		detail   string
		err      error
	}

	rpcCh := make(chan endpointResult, 1)
	apiCh := make(chan endpointResult, 1)

	go func() {
		start := timeNowMS()
		id, err := c.eth.ChainID(ctx)
		r := endpointResult{ok: err == nil, duration: timeNowMS() - start, err: err}
		if err == nil {
			r.detail = fmt.Sprintf("chainId=%s", id.String())
		}
		rpcCh <- r
	}()

	go func() {
		start := timeNowMS()
		_, err := fetchTokens(ctx, NetworkBase, c.infoURL())
		r := endpointResult{ok: err == nil, duration: timeNowMS() - start, err: err}
		if err == nil {
			r.detail = "ok"
		}
		apiCh <- r
	}()

	rpc := <-rpcCh
	api := <-apiCh

	return HealthCheck{
		RPC: HealthEndpoint{OK: rpc.ok, DurationMs: rpc.duration, Detail: rpc.detail, Err: rpc.err},
		API: HealthEndpoint{OK: api.ok, DurationMs: api.duration, Detail: api.detail, Err: api.err},
	}
}

// Close closes the underlying RPC connection.
func (c *Client) Close() { c.eth.Close() }
