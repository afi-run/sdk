package afi

import (
	"strings"
	"testing"
)

const (
	testHash = "0xabc123"
	testAddr = "0xdeadbeef00000000000000000000000000000000"
)

func TestTxURL_KnownNetworks(t *testing.T) {
	cases := []struct {
		network Network
		want    string
	}{
		{NetworkBase, "https://basescan.org/tx/" + testHash},
		{NetworkBSC, "https://bscscan.com/tx/" + testHash},
		{NetworkArbitrum, "https://arbiscan.io/tx/" + testHash},
		{NetworkEthereum, "https://etherscan.io/tx/" + testHash},
		{NetworkUnichain, "https://uniscan.xyz/tx/" + testHash},
	}
	for _, c := range cases {
		t.Run(string(c.network), func(t *testing.T) {
			got, err := TxURL(testHash, c.network)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestTxURL_CustomExplorer(t *testing.T) {
	got, err := TxURL(testHash, NetworkBase, "https://my.explorer///")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != "https://my.explorer/tx/"+testHash {
		t.Errorf("custom explorer not honored / not trimmed: %q", got)
	}
}

func TestTxURL_UnknownNetwork(t *testing.T) {
	_, err := TxURL(testHash, Network("polkadot"))
	if err == nil || !strings.Contains(err.Error(), "no explorer URL") {
		t.Errorf("expected explorer-missing error, got: %v", err)
	}
}

func TestAddressURL(t *testing.T) {
	got, err := AddressURL(testAddr, NetworkBSC)
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://bscscan.com/address/"+testAddr {
		t.Errorf("got %q", got)
	}
}
