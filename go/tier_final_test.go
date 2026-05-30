package afi

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestFeeFromReceipt(t *testing.T) {
	receipt := &types.Receipt{
		GasUsed:           150_000,
		EffectiveGasPrice: big.NewInt(2_000_000_000), // 2 gwei
	}
	eff, fee, eth := feeFromReceipt(receipt)
	if eff.Cmp(big.NewInt(2_000_000_000)) != 0 {
		t.Errorf("effective gas price mismatch: %s", eff)
	}
	want := new(big.Int).SetUint64(300_000_000_000_000)
	if fee.Cmp(want) != 0 {
		t.Errorf("fee mismatch: got %s, want %s", fee, want)
	}
	if eth != "0.0003" {
		t.Errorf("ETH formatted: %q, want 0.0003", eth)
	}
}

func TestFeeFromReceipt_NilGasPrice(t *testing.T) {
	receipt := &types.Receipt{GasUsed: 150_000}
	eff, fee, eth := feeFromReceipt(receipt)
	if eff.Sign() != 0 || fee.Sign() != 0 {
		t.Error("nil gas price should produce zero fee")
	}
	if eth != "0" {
		t.Errorf("ETH formatted: %q, want 0", eth)
	}
}

func TestReceiptToTxReceipt(t *testing.T) {
	receipt := &types.Receipt{
		BlockNumber:       big.NewInt(42),
		GasUsed:           21_000,
		EffectiveGasPrice: big.NewInt(1_000_000_000),
	}
	r := receiptToTxReceipt(receipt)
	if r.BlockNumber != 42 || r.GasUsed != 21_000 {
		t.Errorf("base fields mismatch: %+v", r)
	}
	if r.FeeWei.Cmp(big.NewInt(21_000_000_000_000)) != 0 {
		t.Errorf("FeeWei mismatch: %s", r.FeeWei)
	}
}

func TestNonceManagement_DefaultDisabled(t *testing.T) {
	c, _ := NewClient(Config{RPCURL: "http://localhost:1"})
	defer c.Close()
	if c.managedNonce {
		t.Error("managed nonce should default to disabled")
	}
}

func TestNonceManagement_DisableResetsState(t *testing.T) {
	c, _ := NewClient(Config{RPCURL: "http://localhost:1"})
	defer c.Close()
	// Simulate enabled mode without RPC by setting fields directly.
	c.nonceMu.Lock()
	c.managedNonce = true
	c.localNonce = 100
	c.nonceMu.Unlock()

	c.DisableManagedNonce()
	if c.managedNonce || c.localNonce != 0 {
		t.Errorf("expected disabled+zero, got managed=%v local=%d", c.managedNonce, c.localNonce)
	}
}

func TestAllocateNonce_OverrideAndManaged(t *testing.T) {
	c, _ := NewClient(Config{RPCURL: "http://localhost:1"})
	defer c.Close()

	// Override always wins.
	override := uint64(999)
	got, err := c.allocateNonce(context.Background(), &override, common.Address{})
	if err != nil {
		t.Fatal(err)
	}
	if got != 999 {
		t.Errorf("override should win, got %d", got)
	}

	// Managed mode: returns and increments local counter.
	c.nonceMu.Lock()
	c.managedNonce = true
	c.localNonce = 50
	c.nonceMu.Unlock()

	a, _ := c.allocateNonce(context.Background(), nil, common.Address{})
	b, _ := c.allocateNonce(context.Background(), nil, common.Address{})
	if a != 50 || b != 51 {
		t.Errorf("managed mode mismatch: %d, %d", a, b)
	}
}

func TestTokenPriceOptionsDefaults(t *testing.T) {
	// Trivial test that the struct can be constructed and zero values default properly.
	opts := TokenPriceOptions{}
	if opts.Amount != "" {
		t.Error("default amount should be empty")
	}
	// Defaults are applied inside GetTokenPrice — covered by integration tests with a mock quoter.
}

func TestPreflightReport_Construction(t *testing.T) {
	r := &PreflightReport{
		Balance:       big.NewInt(100),
		Allowance:     big.NewInt(50),
		NeedsApproval: true,
		Problems: []PreflightProblem{
			{Code: "INSUFFICIENT_BALANCE", Message: "have 100, need 500"},
		},
	}
	if r.CanExecute {
		t.Error("expected CanExecute=false by default with problems")
	}
}

func TestTxStatus_Constants(t *testing.T) {
	for _, s := range []TxStatus{TxStatusPending, TxStatusSuccess, TxStatusFailed, TxStatusUnknown} {
		if s == "" {
			t.Errorf("status constant empty")
		}
	}
}
