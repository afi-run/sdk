package afi

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func timeNowMS() int64 { return time.Now().UnixMilli() }

// Network represents a supported blockchain network.
type Network string

const (
	NetworkBase     Network = "base"
	NetworkBSC      Network = "bsc"
	NetworkArbitrum Network = "arbitrum"
	NetworkEthereum Network = "ethereum"
	NetworkUnichain Network = "unichain"
)

// Dex represents a supported DEX protocol.
type Dex string

const (
	DexUniV3     Dex = "uni-v3"
	DexUniV4     Dex = "uni-v4"
	DexCakeV3    Dex = "cake-v3"
	DexAerodrome Dex = "aerodrome"
	DexBalancer  Dex = "balancer"
	DexCurve128  Dex = "curve128"
	DexCurve256  Dex = "curve256"
	DexFluid     Dex = "fluid"
)

// RpcUrlInfo specifies a custom RPC endpoint for the quoter API.
type RpcUrlInfo struct {
	URL     string `json:"url"`
	Account int    `json:"account,omitempty"`
	IPC     bool   `json:"ipc,omitempty"`
}

// Config is the required configuration to create a Client.
type Config struct {
	// RPC endpoint for Base network (required).
	RPCURL string
	// Private key hex string with or without 0x prefix (optional — required only for Approve, SubmitSwap, ExecuteSwap, Swap).
	PrivateKey string
	// GasBufferPercent adds a percentage on top of the estimated gas for write txs (approve, swap).
	// Zero (or unset) falls back to DefaultGasBufferPercent (15%). Use 0 explicitly via WithGasBuffer(0) per call.
	GasBufferPercent uint
	// Logger receives a LogEvent for each significant SDK operation. Optional.
	Logger Logger
}

// LogEvent describes a single SDK operation for diagnostics.
type LogEvent struct {
	// Kind is "rpc" for chain calls or "api" for AFI HTTP API calls.
	Kind string
	// Method is the operation name (e.g. "GetQuote", "ExecuteSwap", "TokenInfo").
	Method string
	// DurationMs is the wall-clock duration of the operation.
	DurationMs int64
	// OK is true when the operation returned without error.
	OK bool
	// Err is non-nil when OK is false.
	Err error
}

// Logger is the callback signature for diagnostic events.
type Logger func(event LogEvent)

// WaitForTxOptions controls how WaitForTx polls and times out.
type WaitForTxOptions struct {
	// Confirmations is the minimum number of confirmations required (default: 1).
	Confirmations uint64
	// TimeoutMs aborts the wait after the given duration in milliseconds (default: no timeout).
	TimeoutMs int64
	// PollIntervalMs overrides the polling interval (default: 500ms).
	PollIntervalMs int64
}

// ExecuteOptions controls behavior of ExecuteSwap / Swap.
type ExecuteOptions struct {
	// Confirmations is the minimum number of confirmations to wait for after submission (default: 1).
	Confirmations uint64
	// TimeoutMs is the timeout for the confirmation wait (default: no timeout).
	TimeoutMs int64
	// Nonce is an explicit nonce for the swap tx. When nil, the SDK uses the
	// managed nonce (if UseManagedNonce was called) or queries the RPC.
	Nonce *uint64
}

// TxStatus is the result of a non-blocking transaction status check.
type TxStatus string

const (
	TxStatusPending TxStatus = "pending"
	TxStatusSuccess TxStatus = "success"
	TxStatusFailed  TxStatus = "failed"
	TxStatusUnknown TxStatus = "unknown"
)

// TokenPrice is the live exchange rate for a token pair.
type TokenPrice struct {
	// Price is how many units of tokenOut you get for 1 unit of tokenIn.
	Price string
	// Inverse is how many units of tokenIn you get for 1 unit of tokenOut.
	Inverse string
}

// PreflightProblem is a single blocking issue identified by Preflight.
type PreflightProblem struct {
	Code    string
	Message string
}

// PreflightReport is the result of Client.Preflight.
type PreflightReport struct {
	// CanExecute is true when no blocking problems were found. ExecuteSwap
	// will still auto-handle approval if NeedsApproval is true.
	CanExecute bool
	// NeedsApproval is true when an approve tx is needed before the swap can execute.
	NeedsApproval bool
	// Problems holds blocking issues — empty when CanExecute is true.
	Problems []PreflightProblem
	// Balance is the current tokenIn balance of the wallet.
	Balance *big.Int
	// Allowance is the current allowance granted to the AFI router.
	Allowance *big.Int
}

// SwapCostEstimate is the projected gas cost for executing a quote.
type SwapCostEstimate struct {
	// Gas is the estimated gas units (before the SDK buffer).
	Gas uint64
	// GasWithBuffer is Gas multiplied by (1 + gasBufferPercent/100).
	GasWithBuffer uint64
	// GasPriceWei is the maxFeePerGas the SDK would use (baseFee*2 + tip).
	GasPriceWei *big.Int
	// TotalWei is GasWithBuffer * GasPriceWei.
	TotalWei *big.Int
	// TotalETH is TotalWei formatted as ETH (18 decimals).
	TotalETH string
}

// HealthCheck is the result of a Health() probe.
type HealthCheck struct {
	RPC HealthEndpoint
	API HealthEndpoint
}

// HealthEndpoint describes the result of probing a single endpoint.
type HealthEndpoint struct {
	OK         bool
	DurationMs int64
	// Detail carries the chain ID for RPC and the HTTP status for API.
	Detail string
	Err    error
}

// TxReceipt holds the on-chain confirmation details of a submitted transaction.
type TxReceipt struct {
	BlockNumber uint64
	GasUsed     uint64
	// EffectiveGasPrice is the wei/gas price the chain charged. Nil when the receipt didn't expose it.
	EffectiveGasPrice *big.Int
	// FeeWei is GasUsed * EffectiveGasPrice.
	FeeWei *big.Int
	// FeeETH is FeeWei formatted in ETH (18 decimals).
	FeeETH string
}

// PendingTx represents a submitted transaction that can be waited on for confirmation.
// TxHash is available immediately; call Wait() to block until the tx is mined.
type PendingTx struct {
	TxHash string
	waitFn func(ctx context.Context, opts WaitForTxOptions) (*TxReceipt, error)
}

// Wait blocks until the transaction is confirmed and returns the receipt.
// Pass WaitForTxOptions to require more than 1 confirmation or set a timeout.
func (p *PendingTx) Wait(ctx context.Context, opts ...WaitForTxOptions) (*TxReceipt, error) {
	var o WaitForTxOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return p.waitFn(ctx, o)
}

// PendingSwap represents a submitted swap transaction.
// TxHash is available immediately; call Wait() to block until confirmed and get the SwapResult.
type PendingSwap struct {
	TxHash string
	waitFn func(ctx context.Context, opts WaitForTxOptions) (*SwapResult, error)
}

// Wait blocks until the swap is confirmed and returns the result parsed from the SwapExecuted event.
// Pass WaitForTxOptions to require more than 1 confirmation or set a timeout.
func (p *PendingSwap) Wait(ctx context.Context, opts ...WaitForTxOptions) (*SwapResult, error) {
	var o WaitForTxOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return p.waitFn(ctx, o)
}

// Token represents a supported token on Base.
type Token struct {
	Address  common.Address
	Symbol   string
	Decimals uint8
	Active   bool
}

// Hop describes a single step in a multi-hop route returned by the quoter.
type Hop struct {
	TokenIn      common.Address
	TokenOut     common.Address
	AmountIn     string
	AmountOut    string
	MinOut       string
	AmountInWei  *big.Int
	AmountOutWei *big.Int
	MinOutWei    *big.Int
	TokenInPrice  string
	TokenOutPrice string
	Slippage     float64
	// Type is the pool protocol, e.g. "v3", "v2".
	Type    string
	// Kind is the routing engine, e.g. "cake".
	Kind    string
	RouteID int
	Weight  float64
}

// Quote holds the result of a quoter request and the data needed for on-chain execution.
type Quote struct {
	TokenIn      common.Address
	TokenOut     common.Address
	AmountIn     string
	AmountOut    string
	MinOut       string
	// AmountInWei is the exact amount to approve and pass to swap().
	AmountInWei *big.Int
	// AmountOutWei is the estimated output (informational).
	AmountOutWei *big.Int
	// MinOutWei is the minimum output after slippage — never bypassed.
	MinOutWei *big.Int
	// Steps is the encoded route bytes passed as params to Afi.swap().
	Steps []byte
	// Path is the token path for the route.
	Path     []common.Address
	Hops     []Hop
	Slippage float64
	// FeeBps is the current protocol fee read from the contract.
	FeeBps uint16
	// TokenInPrice is the USD price of TokenIn as reported by the quoter.
	TokenInPrice string
	// TokenOutPrice is the USD price of TokenOut as reported by the quoter.
	TokenOutPrice string
	// TokenInBasePrice is the price of TokenIn in the priceBase asset.
	// Present only when WithPriceBase is used.
	TokenInBasePrice string
	// TokenOutBasePrice is the price of TokenOut in the priceBase asset.
	// Present only when WithPriceBase is used.
	TokenOutBasePrice string
	// CreatedAt is the unix-millisecond timestamp when the quote was fetched.
	CreatedAt int64
	// Network the quote was fetched against — needed by RefreshQuote.
	Network Network
	// MaxHops used in the routing — needed by RefreshQuote.
	MaxHops int
	// PriceBase asset used (if any) — needed by RefreshQuote.
	PriceBase string
	// Dexs restriction used (if any) — needed by RefreshQuote.
	Dexs []Dex
}

// IsStale returns true when the quote is older than maxAgeSec seconds.
func (q *Quote) IsStale(maxAgeSec int64) bool {
	if q.CreatedAt == 0 {
		return false
	}
	return (timeNowMS()-q.CreatedAt)/1000 > maxAgeSec
}

// TokenInfo holds metadata + (optionally) balance/allowance for a token, fetched in one multicall.
type TokenInfo struct {
	Address   common.Address
	Symbol    string
	Name      string
	Decimals  uint8
	// Balance is the balance of Owner, when Owner was provided.
	Balance *big.Int
	// Allowance is the allowance granted by Owner to the AFI contract, when Owner was provided.
	Allowance *big.Int
	// Owner is the address used for Balance/Allowance lookups. Zero address when not queried.
	Owner common.Address
}

// SwapResult holds the outcome of an executed swap.
type SwapResult struct {
	TxHash      common.Hash
	BlockNumber uint64
	// AmountIn is the actual input from the SwapExecuted event.
	AmountIn *big.Int
	// AmountOut is the actual output from the SwapExecuted event.
	AmountOut *big.Int
	TokenIn   common.Address
	TokenOut  common.Address
	GasUsed   uint64
	// EffectiveGasPrice is the wei/gas price the chain charged.
	EffectiveGasPrice *big.Int
	// FeeWei is GasUsed * EffectiveGasPrice.
	FeeWei *big.Int
	// FeeETH is FeeWei formatted in ETH (18 decimals).
	FeeETH string
}
