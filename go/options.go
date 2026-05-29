package afi

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type quoteOptions struct {
	tokenIn     string
	tokenOut    string
	amountIn    string
	slippage    float64
	maxHops     int
	network     Network
	priceBase   string
	dexs        []Dex
	blockNumber string
	rpcUrls     []RpcUrlInfo
	err         error
}

func defaultOptions() *quoteOptions {
	return &quoteOptions{
		slippage: 0.5,
		maxHops:  2,
		network:  NetworkBase,
	}
}

// QuoteOption is a functional option for GetQuote and Swap.
type QuoteOption func(*quoteOptions)

func resolveAddr(t any) (string, error) {
	switch v := t.(type) {
	case Token:
		return v.Address.Hex(), nil
	case common.Address:
		return v.Hex(), nil
	default:
		return "", fmt.Errorf("unsupported type %T: use common.Address or afi.Token", t)
	}
}

// From sets the token pair and input amount.
// tokenIn and tokenOut accept common.Address or afi.Token.
func From(tokenIn, tokenOut any, amountIn string) QuoteOption {
	return func(o *quoteOptions) {
		if o.err != nil {
			return
		}
		in, err := resolveAddr(tokenIn)
		if err != nil {
			o.err = fmt.Errorf("tokenIn: %w", err)
			return
		}
		out, err := resolveAddr(tokenOut)
		if err != nil {
			o.err = fmt.Errorf("tokenOut: %w", err)
			return
		}
		o.tokenIn = in
		o.tokenOut = out
		o.amountIn = amountIn
	}
}

// WithSlippage sets slippage tolerance percentage (default: 0.5).
func WithSlippage(v float64) QuoteOption {
	return func(o *quoteOptions) { o.slippage = v }
}

// WithMaxHops sets the maximum number of route hops (default: 2).
func WithMaxHops(n int) QuoteOption {
	return func(o *quoteOptions) { o.maxHops = n }
}

// OnNetwork sets the target network (default: NetworkBase).
func OnNetwork(n Network) QuoteOption {
	return func(o *quoteOptions) { o.network = n }
}

// WithPriceBase sets the base asset for price fields.
// When set, Quote.TokenInBasePrice and Quote.TokenOutBasePrice are populated.
func WithPriceBase(base string) QuoteOption {
	return func(o *quoteOptions) { o.priceBase = base }
}

// WithDexs restricts routing to the specified DEX protocols.
func WithDexs(dexs ...Dex) QuoteOption {
	return func(o *quoteOptions) { o.dexs = dexs }
}

// WithBlockNumber sets the block number for the quote (default: "latest").
func WithBlockNumber(n string) QuoteOption {
	return func(o *quoteOptions) { o.blockNumber = n }
}

// WithRpcUrls sets custom RPC endpoints for the quoter API.
func WithRpcUrls(urls ...RpcUrlInfo) QuoteOption {
	return func(o *quoteOptions) { o.rpcUrls = urls }
}
