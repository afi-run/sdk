package afi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// doHTTPRequest centralises the typed-error mapping for HTTP responses.
// Returns:
//   - *NetworkError    transport / dial / decode failure
//   - *BadRequestError  4xx (caller payload bad)
//   - *ServerError      5xx (service down — retry-safe)
//   - the raw body on 2xx success
func doHTTPRequest(ctx context.Context, method, url string, payload []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(payload))
	if err != nil {
		return nil, &NetworkError{Err: fmt.Errorf("build request: %w", err)}
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, &NetworkError{Err: err}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return body, nil
	case resp.StatusCode >= 500:
		return nil, &ServerError{Status: resp.StatusCode, Body: string(body)}
	default:
		// 3xx is not expected from these JSON APIs — bucket with 4xx so the caller surfaces it.
		return nil, &BadRequestError{Status: resp.StatusCode, Body: string(body)}
	}
}

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
	TokenIn           string      `json:"tokenIn"`
	TokenOut          string      `json:"tokenOut"`
	AmountIn          string      `json:"amountIn"`
	AmountOut         string      `json:"amountOut"`
	MinOut            string      `json:"minOut"`
	AmountInRaw       string      `json:"amountInRaw"`
	AmountOutRaw      string      `json:"amountOutRaw"`
	MinOutRaw         string      `json:"minOutRaw"`
	Steps             string      `json:"steps"`
	Slippage          float64     `json:"slippage"`
	Path              []string    `json:"path"`
	Hops              []quoterHop `json:"hops"`
	TokenInPrice      string      `json:"tokenInPrice"`
	TokenOutPrice     string      `json:"tokenOutPrice"`
	TokenInBasePrice  string      `json:"tokenInBasePrice"`
	TokenOutBasePrice string      `json:"tokenOutBasePrice"`
}

type quoterResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	// Data is an object on success and may be a string on error.
	Data json.RawMessage `json:"data"`
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

	respBody, err := doHTTPRequest(ctx, http.MethodPost, quoterURL, payload)
	if err != nil {
		// Surface the typed error directly — callers can still match QUOTE_FAILED via
		// errors.As but get the precise BadRequest / Server / Network bucket too.
		return nil, err
	}

	var result quoterResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
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
		CreatedAt:         timeNowMS(),
		Network:           opts.network,
		MaxHops:           opts.maxHops,
		PriceBase:         opts.priceBase,
		Dexs:              opts.dexs,
	}, nil
}

// ─── Generic JSON endpoint helpers ───────────────────────────────────────────

// commandRequest wraps a /command payload with the action discriminator.
type commandRequest struct {
	Action string                 `json:"action"`
	Body   map[string]interface{} `json:"-"`
}

// marshal merges Action + Body into a single JSON object.
func (cr commandRequest) marshal() ([]byte, error) {
	m := make(map[string]interface{}, len(cr.Body)+1)
	for k, v := range cr.Body {
		m[k] = v
	}
	m["action"] = cr.Action
	return json.Marshal(m)
}

// postCommand POSTs to /command with the given action + payload, decoding the
// service envelope ({status, data, time}) and unmarshalling `data` into out.
func (c *Client) postCommand(ctx context.Context, action string, body map[string]interface{}, out interface{}) error {
	payload, err := commandRequest{Action: action, Body: body}.marshal()
	if err != nil {
		return ErrQuote(fmt.Sprintf("marshal: %v", err))
	}
	raw, err := doHTTPRequest(ctx, http.MethodPost, c.apiURL+"/command", payload)
	if err != nil {
		return err
	}
	return decodeEnvelope(raw, out)
}

// decodeEnvelope unwraps the standard afi-rpc response envelope
// ({status, data, time}): on a non-success status it returns the error message,
// otherwise it unmarshals the `data` field into out (a no-op when out is nil or
// data is empty).
func decodeEnvelope(raw []byte, out interface{}) error {
	var env struct {
		Status string          `json:"status"`
		Data   json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return ErrQuote(fmt.Sprintf("decode envelope: %v", err))
	}
	if env.Status != "" && env.Status != "success" {
		var msg string
		_ = json.Unmarshal(env.Data, &msg)
		if msg == "" {
			msg = env.Status
		}
		return ErrQuote(msg)
	}
	if out == nil || len(env.Data) == 0 {
		return nil
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return ErrQuote(fmt.Sprintf("decode data: %v", err))
	}
	return nil
}

// postForData POSTs req to url and decodes the response envelope's data into out.
func postForData(ctx context.Context, url string, req, out interface{}) error {
	payload, err := json.Marshal(req)
	if err != nil {
		return ErrQuote(fmt.Sprintf("marshal: %v", err))
	}
	raw, err := doHTTPRequest(ctx, http.MethodPost, url, payload)
	if err != nil {
		return err
	}
	return decodeEnvelope(raw, out)
}

// ───────────────── HTTP endpoint wrappers ─────────────────

// ArbitrageRequest is the body of POST /arbitrage. Use raw fields (tokenIn,
// tokenOut, amountIn, network) — the RPC service interprets the shape. For a
// self-funded cycle set tokenIn == tokenOut.
type ArbitrageRequest map[string]interface{}

// FindArbitrage hits POST /arbitrage and returns the candidate routes, decoding
// the service envelope ({status, data, time}). Each RouteQuote is an executable
// single-DEX route — feed the most profitable one to QuoteFromRoute.
func (c *Client) FindArbitrage(ctx context.Context, req ArbitrageRequest) ([]RouteQuote, error) {
	var routes []RouteQuote
	if err := postForData(ctx, c.apiURL+"/arbitrage", req, &routes); err != nil {
		return nil, err
	}
	return routes, nil
}

// PathRequest is the body of POST /command action="path".
type PathRequest map[string]interface{}

// FindPath hits POST /command action="path" and returns the priced multi-hop
// route for an explicit token path.
func (c *Client) FindPath(ctx context.Context, req PathRequest) (*PathQuote, error) {
	var resp PathQuote
	if err := c.postCommand(ctx, "path", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// RoutesRequest is the body of POST /command action="routes".
type RoutesRequest map[string]interface{}

// GetRoutes hits POST /command action="routes" and returns the candidate token
// paths between tokenIn and tokenOut.
func (c *Client) GetRoutes(ctx context.Context, req RoutesRequest) ([]Route, error) {
	var routes []Route
	if err := c.postCommand(ctx, "routes", req, &routes); err != nil {
		return nil, err
	}
	return routes, nil
}

// LiquidationCandidatesRequest is the body of POST /aave.
type LiquidationCandidatesRequest map[string]interface{}

// GetLiquidationCandidates hits POST /aave and returns open Aave positions
// eligible for liquidation.
func (c *Client) GetLiquidationCandidates(ctx context.Context, req LiquidationCandidatesRequest) ([]AavePosition, error) {
	var positions []AavePosition
	if err := postForData(ctx, c.apiURL+"/aave", req, &positions); err != nil {
		return nil, err
	}
	return positions, nil
}

// LiquidateRequest is the body of POST /liquidation-call.
type LiquidateRequest map[string]interface{}

// Liquidate hits POST /liquidation-call and returns the executable route that
// repays the debt and swaps the seized collateral back.
func (c *Client) Liquidate(ctx context.Context, req LiquidateRequest) (*LiquidationResult, error) {
	var resp LiquidationResult
	if err := postForData(ctx, c.apiURL+"/liquidation-call", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// PriceQuoteRequest is the body of POST /command action="price".
type PriceQuoteRequest map[string]interface{}

// PriceQuote hits POST /command action="price" and returns the per-DEX quotes
// for the pair (same shape as FindArbitrage).
func (c *Client) PriceQuote(ctx context.Context, req PriceQuoteRequest) ([]RouteQuote, error) {
	var quotes []RouteQuote
	if err := c.postCommand(ctx, "price", req, &quotes); err != nil {
		return nil, err
	}
	return quotes, nil
}

// DexQuoteRequest is the body of POST /command action=dex.
type DexQuoteRequest map[string]interface{}

// QuoteDex hits POST /command action=dex (e.g. "uniV3", "aerodrome") and returns
// that DEX's quotes for the pair.
func (c *Client) QuoteDex(ctx context.Context, dex string, req DexQuoteRequest) ([]RouteQuote, error) {
	var quotes []RouteQuote
	if err := c.postCommand(ctx, dex, req, &quotes); err != nil {
		return nil, err
	}
	return quotes, nil
}
