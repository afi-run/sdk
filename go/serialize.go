package afi

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// JSON serialization helpers.
//
// Default *big.Int marshaling writes raw decimal numbers, which lose precision
// once they exceed JavaScript's safe integer range. The Marshal/Unmarshal pairs
// here serialize *big.Int as base-10 strings so the payload round-trips cleanly
// across language boundaries.

type hopJSON struct {
	TokenIn       common.Address `json:"tokenIn"`
	TokenOut      common.Address `json:"tokenOut"`
	AmountIn      string         `json:"amountIn"`
	AmountOut     string         `json:"amountOut"`
	MinOut        string         `json:"minOut"`
	AmountInWei   string         `json:"amountInWei"`
	AmountOutWei  string         `json:"amountOutWei"`
	MinOutWei     string         `json:"minOutWei"`
	TokenInPrice  string         `json:"tokenInPrice"`
	TokenOutPrice string         `json:"tokenOutPrice"`
	Slippage      float64        `json:"slippage"`
	Type          string         `json:"type"`
	Kind          string         `json:"kind"`
	RouteID       int            `json:"routeId"`
	Weight        float64        `json:"weight"`
}

type quoteJSON struct {
	TokenIn           common.Address   `json:"tokenIn"`
	TokenOut          common.Address   `json:"tokenOut"`
	AmountIn          string           `json:"amountIn"`
	AmountOut         string           `json:"amountOut"`
	MinOut            string           `json:"minOut"`
	AmountInWei       string           `json:"amountInWei"`
	AmountOutWei      string           `json:"amountOutWei"`
	MinOutWei         string           `json:"minOutWei"`
	Steps             string           `json:"steps"`
	Path              []common.Address `json:"path"`
	Hops              []hopJSON        `json:"hops"`
	Slippage          float64          `json:"slippage"`
	FeeBps            uint16           `json:"feeBps"`
	TokenInPrice      string           `json:"tokenInPrice"`
	TokenOutPrice     string           `json:"tokenOutPrice"`
	TokenInBasePrice  string           `json:"tokenInBasePrice,omitempty"`
	TokenOutBasePrice string           `json:"tokenOutBasePrice,omitempty"`
	CreatedAt         int64            `json:"createdAt"`
	Network           Network          `json:"network"`
	MaxHops           int              `json:"maxHops"`
	PriceBase         string           `json:"priceBase,omitempty"`
	Dexs              []Dex            `json:"dexs,omitempty"`
}

func bigIntString(b *big.Int) string {
	if b == nil {
		return "0"
	}
	return b.String()
}

func parseBigInt(s string) (*big.Int, error) {
	b, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return nil, fmt.Errorf("invalid bigint string: %q", s)
	}
	return b, nil
}

// MarshalJSON serializes Quote with *big.Int fields as base-10 strings.
func (q *Quote) MarshalJSON() ([]byte, error) {
	hops := make([]hopJSON, len(q.Hops))
	for i, h := range q.Hops {
		hops[i] = hopJSON{
			TokenIn: h.TokenIn, TokenOut: h.TokenOut,
			AmountIn: h.AmountIn, AmountOut: h.AmountOut, MinOut: h.MinOut,
			AmountInWei: bigIntString(h.AmountInWei), AmountOutWei: bigIntString(h.AmountOutWei), MinOutWei: bigIntString(h.MinOutWei),
			TokenInPrice: h.TokenInPrice, TokenOutPrice: h.TokenOutPrice,
			Slippage: h.Slippage, Type: h.Type, Kind: h.Kind, RouteID: h.RouteID, Weight: h.Weight,
		}
	}
	return json.Marshal(quoteJSON{
		TokenIn: q.TokenIn, TokenOut: q.TokenOut,
		AmountIn: q.AmountIn, AmountOut: q.AmountOut, MinOut: q.MinOut,
		AmountInWei: bigIntString(q.AmountInWei), AmountOutWei: bigIntString(q.AmountOutWei), MinOutWei: bigIntString(q.MinOutWei),
		Steps: "0x" + common.Bytes2Hex(q.Steps), Path: q.Path, Hops: hops,
		Slippage: q.Slippage, FeeBps: q.FeeBps,
		TokenInPrice: q.TokenInPrice, TokenOutPrice: q.TokenOutPrice,
		TokenInBasePrice: q.TokenInBasePrice, TokenOutBasePrice: q.TokenOutBasePrice,
		CreatedAt: q.CreatedAt,
		Network:   q.Network, MaxHops: q.MaxHops, PriceBase: q.PriceBase, Dexs: q.Dexs,
	})
}

// UnmarshalJSON reverses MarshalJSON.
func (q *Quote) UnmarshalJSON(data []byte) error {
	var j quoteJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	amountInWei, err := parseBigInt(j.AmountInWei)
	if err != nil {
		return fmt.Errorf("amountInWei: %w", err)
	}
	amountOutWei, err := parseBigInt(j.AmountOutWei)
	if err != nil {
		return fmt.Errorf("amountOutWei: %w", err)
	}
	minOutWei, err := parseBigInt(j.MinOutWei)
	if err != nil {
		return fmt.Errorf("minOutWei: %w", err)
	}
	hops := make([]Hop, len(j.Hops))
	for i, h := range j.Hops {
		a, errA := parseBigInt(h.AmountInWei)
		b, errB := parseBigInt(h.AmountOutWei)
		m, errM := parseBigInt(h.MinOutWei)
		if errA != nil || errB != nil || errM != nil {
			return fmt.Errorf("hop %d bigint: %v/%v/%v", i, errA, errB, errM)
		}
		hops[i] = Hop{
			TokenIn: h.TokenIn, TokenOut: h.TokenOut,
			AmountIn: h.AmountIn, AmountOut: h.AmountOut, MinOut: h.MinOut,
			AmountInWei: a, AmountOutWei: b, MinOutWei: m,
			TokenInPrice: h.TokenInPrice, TokenOutPrice: h.TokenOutPrice,
			Slippage: h.Slippage, Type: h.Type, Kind: h.Kind, RouteID: h.RouteID, Weight: h.Weight,
		}
	}
	*q = Quote{
		TokenIn: j.TokenIn, TokenOut: j.TokenOut,
		AmountIn: j.AmountIn, AmountOut: j.AmountOut, MinOut: j.MinOut,
		AmountInWei: amountInWei, AmountOutWei: amountOutWei, MinOutWei: minOutWei,
		Steps: common.FromHex(j.Steps), Path: j.Path, Hops: hops,
		Slippage: j.Slippage, FeeBps: j.FeeBps,
		TokenInPrice: j.TokenInPrice, TokenOutPrice: j.TokenOutPrice,
		TokenInBasePrice: j.TokenInBasePrice, TokenOutBasePrice: j.TokenOutBasePrice,
		CreatedAt: j.CreatedAt,
		Network:   j.Network, MaxHops: j.MaxHops, PriceBase: j.PriceBase, Dexs: j.Dexs,
	}
	return nil
}

type swapResultJSON struct {
	TxHash            common.Hash    `json:"txHash"`
	BlockNumber       uint64         `json:"blockNumber"`
	AmountIn          string         `json:"amountIn"`
	AmountOut         string         `json:"amountOut"`
	TokenIn           common.Address `json:"tokenIn"`
	TokenOut          common.Address `json:"tokenOut"`
	GasUsed           uint64         `json:"gasUsed"`
	EffectiveGasPrice string         `json:"effectiveGasPrice,omitempty"`
	FeeWei            string         `json:"feeWei,omitempty"`
	FeeETH            string         `json:"feeEth,omitempty"`
}

func (r *SwapResult) MarshalJSON() ([]byte, error) {
	return json.Marshal(swapResultJSON{
		TxHash: r.TxHash, BlockNumber: r.BlockNumber,
		AmountIn: bigIntString(r.AmountIn), AmountOut: bigIntString(r.AmountOut),
		TokenIn: r.TokenIn, TokenOut: r.TokenOut, GasUsed: r.GasUsed,
		EffectiveGasPrice: bigIntString(r.EffectiveGasPrice),
		FeeWei:            bigIntString(r.FeeWei),
		FeeETH:            r.FeeETH,
	})
}

func (r *SwapResult) UnmarshalJSON(data []byte) error {
	var j swapResultJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	amountIn, err := parseBigInt(j.AmountIn)
	if err != nil {
		return fmt.Errorf("amountIn: %w", err)
	}
	amountOut, err := parseBigInt(j.AmountOut)
	if err != nil {
		return fmt.Errorf("amountOut: %w", err)
	}
	out := SwapResult{
		TxHash: j.TxHash, BlockNumber: j.BlockNumber,
		AmountIn: amountIn, AmountOut: amountOut,
		TokenIn: j.TokenIn, TokenOut: j.TokenOut, GasUsed: j.GasUsed,
		FeeETH: j.FeeETH,
	}
	if j.EffectiveGasPrice != "" {
		if v, err := parseBigInt(j.EffectiveGasPrice); err == nil {
			out.EffectiveGasPrice = v
		}
	}
	if j.FeeWei != "" {
		if v, err := parseBigInt(j.FeeWei); err == nil {
			out.FeeWei = v
		}
	}
	*r = out
	return nil
}

type tokenInfoJSON struct {
	Address   common.Address `json:"address"`
	Symbol    string         `json:"symbol"`
	Name      string         `json:"name"`
	Decimals  uint8          `json:"decimals"`
	Owner     common.Address `json:"owner,omitempty"`
	Balance   string         `json:"balance,omitempty"`
	Allowance string         `json:"allowance,omitempty"`
}

func (t *TokenInfo) MarshalJSON() ([]byte, error) {
	j := tokenInfoJSON{
		Address: t.Address, Symbol: t.Symbol, Name: t.Name, Decimals: t.Decimals,
		Owner: t.Owner,
	}
	if t.Balance != nil {
		j.Balance = t.Balance.String()
	}
	if t.Allowance != nil {
		j.Allowance = t.Allowance.String()
	}
	return json.Marshal(j)
}

func (t *TokenInfo) UnmarshalJSON(data []byte) error {
	var j tokenInfoJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	out := TokenInfo{
		Address: j.Address, Symbol: j.Symbol, Name: j.Name, Decimals: j.Decimals,
		Owner: j.Owner,
	}
	if j.Balance != "" {
		b, err := parseBigInt(j.Balance)
		if err != nil {
			return fmt.Errorf("balance: %w", err)
		}
		out.Balance = b
	}
	if j.Allowance != "" {
		a, err := parseBigInt(j.Allowance)
		if err != nil {
			return fmt.Errorf("allowance: %w", err)
		}
		out.Allowance = a
	}
	*t = out
	return nil
}
