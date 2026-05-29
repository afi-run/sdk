package afi

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

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
}

// TxReceipt holds the on-chain confirmation details of a submitted transaction.
type TxReceipt struct {
	BlockNumber uint64
	GasUsed     uint64
}

// PendingTx represents a submitted transaction that can be waited on for confirmation.
// TxHash is available immediately; call Wait() to block until the tx is mined.
type PendingTx struct {
	TxHash string
	waitFn func(ctx context.Context) (*TxReceipt, error)
}

// Wait blocks until the transaction is confirmed and returns the receipt.
func (p *PendingTx) Wait(ctx context.Context) (*TxReceipt, error) {
	return p.waitFn(ctx)
}

// PendingSwap represents a submitted swap transaction.
// TxHash is available immediately; call Wait() to block until confirmed and get the SwapResult.
type PendingSwap struct {
	TxHash string
	waitFn func(ctx context.Context) (*SwapResult, error)
}

// Wait blocks until the swap is confirmed and returns the result parsed from the SwapExecuted event.
func (p *PendingSwap) Wait(ctx context.Context) (*SwapResult, error) {
	return p.waitFn(ctx)
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
}
