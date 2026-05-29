package afi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type quoterRequest struct {
	Network     string       `json:"network"`
	TokenIn     string       `json:"tokenIn"`
	TokenOut    string       `json:"tokenOut"`
	AmountIn    string       `json:"amountIn"`
	Slippage    float64      `json:"slippage"`
	MaxHops     int          `json:"maxHops"`
	Show        bool         `json:"show"`
	PriceBase   string       `json:"priceBase,omitempty"`
	Dexs        []string     `json:"dexs,omitempty"`
	BlockNumber string       `json:"blockNumber,omitempty"`
	RpcUrls     []RpcUrlInfo `json:"rpcUrls,omitempty"`
}

type quoterHop struct {
	TokenIn       string  `json:"tokenIn"`
	TokenOut      string  `json:"tokenOut"`
	AmountIn      string  `json:"amountIn"`
	AmountOut     string  `json:"amountOut"`
	MinOut        string  `json:"minOut"`
	AmountInRaw   string  `json:"amountInRaw"`
	AmountOutRaw  string  `json:"amountOutRaw"`
	MinOutRaw     string  `json:"minOutRaw"`
	TokenInPrice  string  `json:"tokenInPrice"`
	TokenOutPrice string  `json:"tokenOutPrice"`
	Slippage      float64 `json:"slippage"`
	Type          string  `json:"type"`
	Kind          string  `json:"kind"`
	RouteID       int     `json:"routeId"`
	Weight        float64 `json:"weight"`
}

type quoterResponseData struct {
	TokenIn          string      `json:"tokenIn"`
	TokenOut         string      `json:"tokenOut"`
	AmountIn         string      `json:"amountIn"`
	AmountOut        string      `json:"amountOut"`
	MinOut           string      `json:"minOut"`
	AmountInRaw      string      `json:"amountInRaw"`
	AmountOutRaw     string      `json:"amountOutRaw"`
	MinOutRaw        string      `json:"minOutRaw"`
	Steps            string      `json:"steps"`
	Slippage         float64     `json:"slippage"`
	Path             []string    `json:"path"`
	Hops             []quoterHop `json:"hops"`
	TokenInPrice     string      `json:"tokenInPrice"`
	TokenOutPrice    string      `json:"tokenOutPrice"`
	TokenInBasePrice  string      `json:"tokenInBasePrice"`
	TokenOutBasePrice string      `json:"tokenOutBasePrice"`
}

type quoterResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	// Data is an object on success and may be a string on error.
	Data    json.RawMessage `json:"data"`
}

func formatAmount(amount *big.Int, decimals uint8) string {
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	whole := new(big.Int).Div(amount, divisor)
	frac := new(big.Int).Mod(amount, divisor)

	if frac.Sign() == 0 {
		return whole.String()
	}

	fracStr := frac.String()
	for len(fracStr) < int(decimals) {
		fracStr = "0" + fracStr
	}
	fracStr = strings.TrimRight(fracStr, "0")
	return whole.String() + "." + fracStr
}

// fetchQuote is called by the client, passing the resolved options and quoter URL.
func fetchQuote(ctx context.Context, opts *quoteOptions, feeBps uint16, quoterURL string) (*Quote, error) {
	return fetchQuoteFrom(ctx, opts, feeBps, quoterURL)
}

// fetchQuoteFrom is the testable version — same params but quoterURL is injectable.
func fetchQuoteFrom(ctx context.Context, opts *quoteOptions, feeBps uint16, quoterURL string) (*Quote, error) {
	body := quoterRequest{
		Network:  string(opts.network),
		TokenIn:  opts.tokenIn,
		TokenOut: opts.tokenOut,
		AmountIn: opts.amountIn,
		Slippage: opts.slippage,
		MaxHops:  opts.maxHops,
		Show:     true,
	}
	if opts.priceBase != "" {
		body.PriceBase = opts.priceBase
	}
	if len(opts.dexs) > 0 {
		dexStrs := make([]string, len(opts.dexs))
		for i, d := range opts.dexs {
			dexStrs[i] = string(d)
		}
		body.Dexs = dexStrs
	}
	if opts.blockNumber != "" {
		body.BlockNumber = opts.blockNumber
	}
	if len(opts.rpcUrls) > 0 {
		body.RpcUrls = opts.rpcUrls
	}

	// json.Marshal cannot fail for a struct with only basic scalar fields
	payload, _ := json.Marshal(body)

	httpClient := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, quoterURL, bytes.NewReader(payload))
	if err != nil {
		return nil, ErrQuote(fmt.Sprintf("build request: %v", err))
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, ErrQuote(fmt.Sprintf("network: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrQuote(fmt.Sprintf("HTTP %d", resp.StatusCode))
	}

	var result quoterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, ErrQuote(fmt.Sprintf("decode: %v", err))
	}

	if result.Status != "success" {
		msg := result.Message
		if msg == "" {
			// data may carry the error as a plain string
			var dataStr string
			if json.Unmarshal(result.Data, &dataStr) == nil && dataStr != "" {
				msg = dataStr
			}
		}
		if msg == "" {
			msg = "unknown error from quoter"
		}
		return nil, ErrQuote(msg)
	}

	var d quoterResponseData
	if err := json.Unmarshal(result.Data, &d); err != nil {
		return nil, ErrQuote(fmt.Sprintf("decode data: %v", err))
	}

	if d.MinOutRaw == "" || d.MinOutRaw == "0" {
		return nil, ErrQuote("received zero minOut — rejected for safety")
	}

	amountInWei := new(big.Int)
	amountInWei.SetString(d.AmountInRaw, 10)

	amountOutWei := new(big.Int)
	amountOutWei.SetString(d.AmountOutRaw, 10)

	minOutWei := new(big.Int)
	minOutWei.SetString(d.MinOutRaw, 10)

	steps := common.FromHex(d.Steps)

	path := make([]common.Address, len(d.Path))
	for i, p := range d.Path {
		path[i] = common.HexToAddress(p)
	}

	hops := make([]Hop, len(d.Hops))
	for i, h := range d.Hops {
		amountInWeiHop := new(big.Int)
		amountInWeiHop.SetString(h.AmountInRaw, 10)
		amountOutWeiHop := new(big.Int)
		amountOutWeiHop.SetString(h.AmountOutRaw, 10)
		minOutWeiHop := new(big.Int)
		minOutWeiHop.SetString(h.MinOutRaw, 10)
		hops[i] = Hop{
			TokenIn:       common.HexToAddress(h.TokenIn),
			TokenOut:      common.HexToAddress(h.TokenOut),
			AmountIn:      h.AmountIn,
			AmountOut:     h.AmountOut,
			MinOut:        h.MinOut,
			AmountInWei:   amountInWeiHop,
			AmountOutWei:  amountOutWeiHop,
			MinOutWei:     minOutWeiHop,
			TokenInPrice:  h.TokenInPrice,
			TokenOutPrice: h.TokenOutPrice,
			Slippage:      h.Slippage,
			Type:          h.Type,
			Kind:          h.Kind,
			RouteID:       h.RouteID,
			Weight:        h.Weight,
		}
	}

	return &Quote{
		TokenIn:           common.HexToAddress(d.TokenIn),
		TokenOut:          common.HexToAddress(d.TokenOut),
		AmountIn:          d.AmountIn,
		AmountOut:         d.AmountOut,
		MinOut:            d.MinOut,
		AmountInWei:       amountInWei,
		AmountOutWei:      amountOutWei,
		MinOutWei:         minOutWei,
		Steps:             steps,
		Path:              path,
		Hops:              hops,
		Slippage:          d.Slippage,
		FeeBps:            feeBps,
		TokenInPrice:      d.TokenInPrice,
		TokenOutPrice:     d.TokenOutPrice,
		TokenInBasePrice:  d.TokenInBasePrice,
		TokenOutBasePrice: d.TokenOutBasePrice,
	}, nil
}
