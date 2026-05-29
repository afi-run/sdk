package afi

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestDefaultOptions(t *testing.T) {
	o := defaultOptions()
	if o.slippage != 0.5 {
		t.Errorf("slippage = %v, want 0.5", o.slippage)
	}
	if o.maxHops != 2 {
		t.Errorf("maxHops = %d, want 2", o.maxHops)
	}
	if o.network != NetworkBase {
		t.Errorf("network = %q, want %q", o.network, NetworkBase)
	}
}

func TestResolveAddr(t *testing.T) {
	addr := common.HexToAddress("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913")

	t.Run("accepts common.Address", func(t *testing.T) {
		got, err := resolveAddr(addr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != addr.Hex() {
			t.Errorf("got %q, want %q", got, addr.Hex())
		}
	})

	t.Run("accepts Token", func(t *testing.T) {
		tok := Token{Address: addr, Symbol: "USDC", Decimals: 6, Active: true}
		got, err := resolveAddr(tok)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != addr.Hex() {
			t.Errorf("got %q, want %q", got, addr.Hex())
		}
	})

	t.Run("rejects unsupported type", func(t *testing.T) {
		_, err := resolveAddr("0xabc")
		if err == nil {
			t.Fatal("expected error for string, got nil")
		}
		if !strings.Contains(err.Error(), "unsupported type") {
			t.Errorf("error = %q, want it to mention \"unsupported type\"", err.Error())
		}
	})

	t.Run("rejects int", func(t *testing.T) {
		_, err := resolveAddr(42)
		if err == nil {
			t.Fatal("expected error for int, got nil")
		}
	})
}

func TestFrom(t *testing.T) {
	usdc := common.HexToAddress("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913")
	weth := common.HexToAddress("0x4200000000000000000000000000000000000006")

	t.Run("populates tokenIn, tokenOut, amountIn", func(t *testing.T) {
		o := defaultOptions()
		From(usdc, weth, "1000")(o)
		if o.err != nil {
			t.Fatalf("unexpected error: %v", o.err)
		}
		if o.tokenIn != usdc.Hex() {
			t.Errorf("tokenIn = %q, want %q", o.tokenIn, usdc.Hex())
		}
		if o.tokenOut != weth.Hex() {
			t.Errorf("tokenOut = %q, want %q", o.tokenOut, weth.Hex())
		}
		if o.amountIn != "1000" {
			t.Errorf("amountIn = %q, want \"1000\"", o.amountIn)
		}
	})

	t.Run("accepts Token values", func(t *testing.T) {
		o := defaultOptions()
		in := Token{Address: usdc, Symbol: "USDC", Decimals: 6}
		out := Token{Address: weth, Symbol: "WETH", Decimals: 18}
		From(in, out, "2.5")(o)
		if o.err != nil {
			t.Fatalf("unexpected error: %v", o.err)
		}
		if o.tokenIn != usdc.Hex() || o.tokenOut != weth.Hex() {
			t.Errorf("addresses not resolved from Token: in=%q out=%q", o.tokenIn, o.tokenOut)
		}
	})

	t.Run("records error for invalid tokenIn", func(t *testing.T) {
		o := defaultOptions()
		From("bad", weth, "1")(o)
		if o.err == nil {
			t.Fatal("expected err for invalid tokenIn, got nil")
		}
		if !strings.Contains(o.err.Error(), "tokenIn") {
			t.Errorf("err = %q, want it to mention \"tokenIn\"", o.err.Error())
		}
	})

	t.Run("records error for invalid tokenOut", func(t *testing.T) {
		o := defaultOptions()
		From(usdc, 123, "1")(o)
		if o.err == nil {
			t.Fatal("expected err for invalid tokenOut, got nil")
		}
		if !strings.Contains(o.err.Error(), "tokenOut") {
			t.Errorf("err = %q, want it to mention \"tokenOut\"", o.err.Error())
		}
	})

	t.Run("is a no-op when err is already set", func(t *testing.T) {
		o := defaultOptions()
		From("bad", weth, "1")(o) // sets err
		prev := o.err
		From(usdc, weth, "999")(o) // should not overwrite
		if o.err != prev {
			t.Errorf("err overwritten: got %v, want %v", o.err, prev)
		}
		if o.tokenIn != "" || o.amountIn != "" {
			t.Errorf("fields populated despite prior err: tokenIn=%q amountIn=%q", o.tokenIn, o.amountIn)
		}
	})
}

func TestWithSlippage(t *testing.T) {
	o := defaultOptions()
	WithSlippage(1.25)(o)
	if o.slippage != 1.25 {
		t.Errorf("slippage = %v, want 1.25", o.slippage)
	}
}

func TestWithMaxHops(t *testing.T) {
	o := defaultOptions()
	WithMaxHops(5)(o)
	if o.maxHops != 5 {
		t.Errorf("maxHops = %d, want 5", o.maxHops)
	}
}

func TestOnNetwork(t *testing.T) {
	o := defaultOptions()
	OnNetwork(NetworkBSC)(o)
	if o.network != NetworkBSC {
		t.Errorf("network = %q, want %q", o.network, NetworkBSC)
	}
}

func TestWithPriceBase(t *testing.T) {
	o := defaultOptions()
	WithPriceBase("USDC")(o)
	if o.priceBase != "USDC" {
		t.Errorf("priceBase = %q, want USDC", o.priceBase)
	}
}

func TestWithDexs(t *testing.T) {
	o := defaultOptions()
	WithDexs(DexUniV3, DexAerodrome)(o)
	if len(o.dexs) != 2 {
		t.Fatalf("len(dexs) = %d, want 2", len(o.dexs))
	}
	if o.dexs[0] != DexUniV3 || o.dexs[1] != DexAerodrome {
		t.Errorf("dexs = %v, want [uni-v3, aerodrome]", o.dexs)
	}
}

func TestWithBlockNumber(t *testing.T) {
	o := defaultOptions()
	WithBlockNumber("0x1234")(o)
	if o.blockNumber != "0x1234" {
		t.Errorf("blockNumber = %q, want 0x1234", o.blockNumber)
	}
}

func TestWithRpcUrls(t *testing.T) {
	o := defaultOptions()
	WithRpcUrls(
		RpcUrlInfo{URL: "https://rpc.example.com"},
		RpcUrlInfo{URL: "https://rpc2.example.com", Account: 1, IPC: true},
	)(o)
	if len(o.rpcUrls) != 2 {
		t.Fatalf("len(rpcUrls) = %d, want 2", len(o.rpcUrls))
	}
	if o.rpcUrls[0].URL != "https://rpc.example.com" {
		t.Errorf("rpcUrls[0].URL = %q", o.rpcUrls[0].URL)
	}
	if !o.rpcUrls[1].IPC || o.rpcUrls[1].Account != 1 {
		t.Errorf("rpcUrls[1] = %+v, want IPC=true Account=1", o.rpcUrls[1])
	}
}
