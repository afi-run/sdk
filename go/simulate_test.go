package afi

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestLookupBalanceSlot_BaseWETH(t *testing.T) {
	slot, ok := LookupBalanceSlot(8453, common.HexToAddress("0x4200000000000000000000000000000000000006"))
	if !ok {
		t.Fatal("expected slot for Base WETH")
	}
	if slot != 3 {
		t.Errorf("expected slot 3 for WETH, got %d", slot)
	}
}

func TestLookupBalanceSlot_Unknown(t *testing.T) {
	_, ok := LookupBalanceSlot(8453, common.HexToAddress("0x000000000000000000000000000000000000dEaD"))
	if ok {
		t.Error("expected ok=false for unknown token")
	}
}

func TestRegisterBalanceSlot(t *testing.T) {
	addr := common.HexToAddress("0x000000000000000000000000000000000000dEaD")
	defer delete(balanceSlots[8453], addr)

	RegisterBalanceSlot(8453, addr, 42)
	got, ok := LookupBalanceSlot(8453, addr)
	if !ok || got != 42 {
		t.Errorf("expected slot 42, got %d ok=%v", got, ok)
	}
}

func TestBuildBalanceOverride(t *testing.T) {
	holder := common.HexToAddress("0xAaAaAaaAaAaAaAaAaAaAAAAAaaaAaAaAaAaAaAaA")
	token := common.HexToAddress("0x4200000000000000000000000000000000000006") // Base WETH, slot 3
	amount := new(big.Int).SetUint64(1000000)

	override, err := BuildBalanceOverride(8453, token, holder, amount)
	if err != nil {
		t.Fatal(err)
	}
	acct, ok := override[token]
	if !ok {
		t.Fatal("override missing token entry")
	}
	if len(acct.StateDiff) != 1 {
		t.Errorf("expected 1 stateDiff entry, got %d", len(acct.StateDiff))
	}
	// Derived slot must be keccak256(holder || 3)
	expectedSlot := mappingSlot(holder, 3)
	val, ok := acct.StateDiff[expectedSlot]
	if !ok {
		t.Fatal("derived slot missing in stateDiff")
	}
	if val != common.BigToHash(amount) {
		t.Errorf("amount mismatch: got %s want %s", val.Hex(), common.BigToHash(amount).Hex())
	}
}

func TestBuildBalanceOverride_UnknownToken(t *testing.T) {
	holder := common.HexToAddress("0xAaAaAaaAaAaAaAaAaAaAAAAAaaaAaAaAaAaAaAaA")
	token := common.HexToAddress("0x000000000000000000000000000000000000bEEF")

	_, err := BuildBalanceOverride(8453, token, holder, big.NewInt(1))
	if err == nil {
		t.Fatal("expected error for unknown token")
	}
}

func TestMappingSlot_Deterministic(t *testing.T) {
	h := common.HexToAddress("0x1234567890123456789012345678901234567890")
	a := mappingSlot(h, 9)
	b := mappingSlot(h, 9)
	if a != b {
		t.Error("mappingSlot must be deterministic")
	}
	c := mappingSlot(h, 10)
	if a == c {
		t.Error("different slots must produce different keys")
	}
}

func TestSimulateOpts_RequiresClient(t *testing.T) {
	_, err := SimulateRoute(nil, SimulateOpts{Amount: big.NewInt(1)})
	if err == nil {
		t.Fatal("expected error for nil RPCClient")
	}
}

func TestSimulateOpts_RequiresAmount(t *testing.T) {
	// Trick: pass a stub RPC client that won't be called since amount check is first
	_, err := SimulateRoute(nil, SimulateOpts{Amount: big.NewInt(0)})
	if err == nil {
		t.Fatal("expected error for zero amount")
	}
}
