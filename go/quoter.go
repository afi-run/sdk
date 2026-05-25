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
	Network  string  `json:"network"`
	TokenIn  string  `json:"tokenIn"`
	TokenOut string  `json:"tokenOut"`
	AmountIn string  `json:"amountIn"`
	Slippage float64 `json:"slippage"`
	MaxHops  int     `json:"maxHops"`
	Show     bool    `json:"show"`
	RPCUrl   string  `json:"rpcUrl"`
}

type quoterResponseData struct {
	TokenIn      string   `json:"tokenIn"`
	TokenOut     string   `json:"tokenOut"`
	AmountInRaw  string   `json:"amountInRaw"`
	AmountOutRaw string   `json:"amountOutRaw"`
	MinOutRaw    string   `json:"minOutRaw"`
	Steps        string   `json:"steps"`
	Slippage     float64  `json:"slippage"`
	Path         []string `json:"path"`
}

type quoterResponse struct {
	Status  string              `json:"status"`
	Message string              `json:"message"`
	Data    *quoterResponseData `json:"data"`
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

func fetchQuote(ctx context.Context, params SwapParams, decimals uint8, feeBps uint16, rpcURL string) (*Quote, error) {
	return fetchQuoteFrom(ctx, params, decimals, feeBps, rpcURL, QuoterURL)
}

// fetchQuoteFrom allows injecting a custom quoter URL — used by tests.
func fetchQuoteFrom(ctx context.Context, params SwapParams, decimals uint8, feeBps uint16, rpcURL, quoterURL string) (*Quote, error) {
	body := quoterRequest{
		Network:  "base",
		TokenIn:  params.TokenIn.Hex(),
		TokenOut: params.TokenOut.Hex(),
		AmountIn: formatAmount(params.AmountIn, decimals),
		Slippage: params.Slippage,
		MaxHops:  4,
		Show:     true,
		RPCUrl:   rpcURL,
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

	if result.Status != "success" || result.Data == nil {
		msg := result.Message
		if msg == "" {
			msg = "unknown error from quoter"
		}
		return nil, ErrQuote(msg)
	}

	d := result.Data

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

	return &Quote{
		TokenIn:      common.HexToAddress(d.TokenIn),
		TokenOut:     common.HexToAddress(d.TokenOut),
		AmountInWei:  amountInWei,
		AmountOutWei: amountOutWei,
		MinOutWei:    minOutWei,
		Steps:        steps,
		Path:         path,
		Slippage:     d.Slippage,
		FeeBps:       feeBps,
	}, nil
}
