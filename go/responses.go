package afi

import "encoding/json"

// Route is one candidate token path returned by GetRoutes (/command "routes").
type Route struct {
	Path []string `json:"path"`
}

// PathQuote is the priced multi-hop route returned by FindPath (/command
// "path") for an explicit token path. Steps is the combined hex-encoded params
// ready for Afi.swap; Hops carries the per-leg breakdown.
type PathQuote struct {
	Network       string       `json:"network"`
	Path          []string     `json:"path"`
	TokenIn       string       `json:"tokenIn"`
	TokenOut      string       `json:"tokenOut"`
	AmountIn      string       `json:"amountIn"`
	AmountInRaw   string       `json:"amountInRaw"`
	AmountOut     string       `json:"amountOut"`
	AmountOutRaw  string       `json:"amountOutRaw"`
	MinOut        string       `json:"minOut"`
	MinOutRaw     string       `json:"minOutRaw"`
	TokenInPrice  string       `json:"tokenInPrice"`
	TokenOutPrice string       `json:"tokenOutPrice"`
	BlockNumber   string       `json:"blockNumber"`
	Slippage      float64      `json:"slippage"`
	Steps         string       `json:"steps"`
	Hops          []RouteQuote `json:"hops"`
}

// LiquidationResult is the executable route returned by Liquidate
// (/liquidation-call): repay the debt and swap the seized collateral back.
// Steps is the combined hex-encoded params; Hops is the aave + swap legs.
type LiquidationResult struct {
	TokenIn     string          `json:"tokenIn"`
	TokenOut    string          `json:"tokenOut"`
	AmountIn    string          `json:"amountIn"`
	AmountOut   string          `json:"amountOut"`
	Profit      string          `json:"profit"`
	BlockNumber json.RawMessage `json:"blockNumber,omitempty"`
	Slippage    float64         `json:"slippage"`
	Steps       string          `json:"steps"`
	Hops        []RouteQuote    `json:"hops"`
}

// AavePosition is an open Aave position eligible for liquidation, returned by
// GetLiquidationCandidates (/aave).
type AavePosition struct {
	User             string           `json:"user"`
	DebtToken        string           `json:"debtToken"`
	DebtTokenAddress string           `json:"debtTokenAddress"`
	DebtAToken       string           `json:"debtAToken"`
	Decimals         int              `json:"decimals"`
	DebtAmountRaw    string           `json:"debtAmountRaw"`
	DebtAmount       string           `json:"debtAmount"`
	Collaterals      []AaveCollateral `json:"collaterals"`
}

// AaveCollateral is one collateral leg of an AavePosition.
type AaveCollateral struct {
	Token        string `json:"token"`
	TokenAddress string `json:"tokenAddress"`
	AToken       string `json:"aToken"`
	BalanceRaw   string `json:"balanceRaw"`
	Balance      string `json:"balance"`
	Decimals     int    `json:"decimals"`
}
