package afi

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Config is the required configuration to create a Client.
type Config struct {
	// RPC endpoint for Base network (required).
	RPCURL string
	// Private key hex string with or without 0x prefix (required).
	PrivateKey string
}

// Token represents a supported token on Base.
type Token struct {
	Address  common.Address
	Symbol   string
	Decimals uint8
	Active   bool
}

// SwapParams are the inputs for a swap operation.
type SwapParams struct {
	TokenIn  common.Address
	TokenOut common.Address
	// AmountIn in raw wei.
	AmountIn *big.Int
	// Slippage percentage, e.g. 0.5 for 0.5%.
	Slippage float64
}

// Quote holds the result of a quoter request and the data needed for on-chain execution.
type Quote struct {
	TokenIn  common.Address
	TokenOut common.Address
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
	Slippage float64
	// FeeBps is the current protocol fee read from the contract.
	FeeBps uint16
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
