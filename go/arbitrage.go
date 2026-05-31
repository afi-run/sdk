package afi

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// RouteQuote is one route returned by the afi-rpc /arbitrage endpoint — a
// single-DEX quote for tokenIn → tokenOut, with the encoded hop ready to run
// through Afi.swap. Use QuoteFromRoute to turn it into an executable Quote.
type RouteQuote struct {
	Network       string  `json:"network"`
	Type          string  `json:"type"`
	Kind          string  `json:"kind"`
	TokenIn       string  `json:"tokenIn"`
	TokenOut      string  `json:"tokenOut"`
	TokenInPrice  string  `json:"tokenInPrice"`
	TokenOutPrice string  `json:"tokenOutPrice"`
	AmountIn      string  `json:"amountIn"`
	AmountInRaw   string  `json:"amountInRaw"`
	AmountOut     string  `json:"amountOut"`
	AmountOutRaw  string  `json:"amountOutRaw"`
	MinOut        string  `json:"minOut"`
	MinOutRaw     string  `json:"minOutRaw"`
	Slippage      float64 `json:"slippage"`
	Weight        float64 `json:"weight"`
	BlockNumber   string  `json:"blockNumber"`
	Fee           int     `json:"fee,omitempty"`
	RouteID       int     `json:"routeId"`
	// StepData is the opaque per-hop calldata for the DEX adapter, hex-encoded.
	StepData string `json:"stepData"`
}

// Profit returns AmountOutRaw − AmountInRaw in base units. For a cycle quote
// (TokenIn == TokenOut) this is the realized gross profit; for a directional
// quote it is meaningless. Returns nil if either raw amount fails to parse.
func (r RouteQuote) Profit() *big.Int {
	in, ok1 := new(big.Int).SetString(r.AmountInRaw, 10)
	out, ok2 := new(big.Int).SetString(r.AmountOutRaw, 10)
	if !ok1 || !ok2 {
		return nil
	}
	return new(big.Int).Sub(out, in)
}

// QuoteFromRoute hydrates an executable Quote from a RouteQuote: it wraps the
// route's single hop ({RouteID, StepData}) into Afi.swap params via EncodeSteps
// and copies over the token/amount fields. The resulting Quote can be passed
// straight to Client.ExecuteSwap / Client.Simulate.
//
// minOutOverride, when non-nil, replaces the route's MinOutRaw — handy for a
// self-funded cycle where the caller wants to floor the output at
// principal + threshold so an unprofitable cycle reverts on-chain.
func QuoteFromRoute(r RouteQuote, minOutOverride *big.Int) (*Quote, error) {
	amountIn, ok := new(big.Int).SetString(r.AmountInRaw, 10)
	if !ok {
		return nil, fmt.Errorf("QuoteFromRoute: bad amountInRaw %q", r.AmountInRaw)
	}
	amountOut, _ := new(big.Int).SetString(r.AmountOutRaw, 10)

	minOut := minOutOverride
	if minOut == nil {
		var ok bool
		if minOut, ok = new(big.Int).SetString(r.MinOutRaw, 10); !ok {
			return nil, fmt.Errorf("QuoteFromRoute: bad minOutRaw %q", r.MinOutRaw)
		}
	}

	if r.RouteID < 0 || r.RouteID > 0xFFFF {
		return nil, fmt.Errorf("QuoteFromRoute: routeId %d out of uint16 range", r.RouteID)
	}
	params, err := EncodeSteps([]Step{{ID: uint16(r.RouteID), Data: common.FromHex(r.StepData)}})
	if err != nil {
		return nil, fmt.Errorf("QuoteFromRoute: encode steps: %w", err)
	}

	return &Quote{
		TokenIn:      common.HexToAddress(r.TokenIn),
		TokenOut:     common.HexToAddress(r.TokenOut),
		AmountIn:     r.AmountIn,
		AmountOut:    r.AmountOut,
		MinOut:       r.MinOut,
		AmountInWei:  amountIn,
		AmountOutWei: amountOut,
		MinOutWei:    minOut,
		Steps:        params,
		Slippage:     r.Slippage,
		Network:      Network(r.Network),
	}, nil
}
