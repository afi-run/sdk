package afi

import (
	"errors"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func TestNewClient_RequiresRPCURL(t *testing.T) {
	_, err := NewClient(Config{})
	if err == nil {
		t.Fatal("expected error when RPCURL is empty")
	}
	if !strings.Contains(err.Error(), "RPCURL") {
		t.Errorf("error should mention RPCURL, got: %v", err)
	}
}

func TestNewClient_DefaultsGasBufferTo15(t *testing.T) {
	c, err := NewClient(Config{RPCURL: "http://localhost:1"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	if c.GasBufferPercent() != DefaultGasBufferPercent {
		t.Errorf("GasBufferPercent() = %d, want %d", c.GasBufferPercent(), DefaultGasBufferPercent)
	}
	if DefaultGasBufferPercent != 15 {
		t.Errorf("DefaultGasBufferPercent = %d, want 15", DefaultGasBufferPercent)
	}
}

func TestNewClient_RespectsConfiguredGasBuffer(t *testing.T) {
	c, err := NewClient(Config{RPCURL: "http://localhost:1", GasBufferPercent: 25})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	if c.GasBufferPercent() != 25 {
		t.Errorf("GasBufferPercent() = %d, want 25", c.GasBufferPercent())
	}
}

func TestSetGasBufferPercent_Overrides(t *testing.T) {
	c, _ := NewClient(Config{RPCURL: "http://localhost:1"})
	defer c.Close()

	ret := c.SetGasBufferPercent(30)
	if ret != c {
		t.Error("SetGasBufferPercent should return the client for chaining")
	}
	if c.GasBufferPercent() != 30 {
		t.Errorf("GasBufferPercent() = %d, want 30", c.GasBufferPercent())
	}

	// 0 is a valid override (disables buffer)
	c.SetGasBufferPercent(0)
	if c.GasBufferPercent() != 0 {
		t.Errorf("GasBufferPercent() = %d, want 0", c.GasBufferPercent())
	}
}

func TestNewClient_InvalidPrivateKey(t *testing.T) {
	_, err := NewClient(Config{RPCURL: "http://localhost:1", PrivateKey: "not-hex"})
	if err == nil {
		t.Fatal("expected error for invalid private key")
	}
}

func TestRequireSigner_NoKey(t *testing.T) {
	c, _ := NewClient(Config{RPCURL: "http://localhost:1"})
	defer c.Close()

	err := c.requireSigner()
	if err == nil {
		t.Fatal("expected ErrNoSigner")
	}
	var afiErr *AfiError
	if !errors.As(err, &afiErr) || afiErr.Code != "NO_SIGNER" {
		t.Errorf("expected AfiError NO_SIGNER, got %v", err)
	}
}

func TestMulticall3ABI_Parses(t *testing.T) {
	a, err := abi.JSON(strings.NewReader(multicall3ABIJSON))
	if err != nil {
		t.Fatalf("parse multicall3 abi: %v", err)
	}
	if _, ok := a.Methods["aggregate3"]; !ok {
		t.Error("aggregate3 method missing from ABI")
	}
}

func TestERC20ABI_HasSymbolAndName(t *testing.T) {
	a, err := abi.JSON(strings.NewReader(erc20ABIJSON))
	if err != nil {
		t.Fatalf("parse erc20 abi: %v", err)
	}
	if _, ok := a.Methods["symbol"]; !ok {
		t.Error("symbol method missing from ERC20 ABI — required by TokenInfo multicall")
	}
	if _, ok := a.Methods["name"]; !ok {
		t.Error("name method missing from ERC20 ABI — required by TokenInfo multicall")
	}
}

func TestClient_TxURL_DefaultsToBase(t *testing.T) {
	c, _ := NewClient(Config{RPCURL: "http://localhost:1"})
	defer c.Close()

	got, err := c.TxURL("0xabc")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://basescan.org/tx/0xabc" {
		t.Errorf("got %q", got)
	}
}

func TestClient_AddressURL_RespectsNetwork(t *testing.T) {
	c, _ := NewClient(Config{RPCURL: "http://localhost:1"})
	defer c.Close()

	got, err := c.AddressURL("0xdead", NetworkBSC)
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://bscscan.com/address/0xdead" {
		t.Errorf("got %q", got)
	}
}

func TestNetworkExplorers_HasAllSupportedNetworks(t *testing.T) {
	supported := []Network{NetworkBase, NetworkBSC, NetworkArbitrum, NetworkEthereum, NetworkUnichain}
	for _, n := range supported {
		if _, ok := NetworkExplorers[n]; !ok {
			t.Errorf("NetworkExplorers missing %q", n)
		}
		if _, ok := NetworkChainIDs[n]; !ok {
			t.Errorf("NetworkChainIDs missing %q", n)
		}
	}
}
