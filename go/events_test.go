package afi

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// synthLog builds a *types.Log for `eventName` in `parsed`, drawing topics from
// `indexedVals` (in declaration order of the indexed args) and packing the rest
// of the values as ABI data.
func synthLog(t *testing.T, parsed abi.ABI, emitter common.Address, eventName string, indexedVals []interface{}, nonIndexedVals ...interface{}) *types.Log {
	t.Helper()
	ev, ok := parsed.Events[eventName]
	if !ok {
		t.Fatalf("event %q not found in ABI", eventName)
	}
	// Topic 0 is the event signature hash.
	topics := []common.Hash{ev.ID}
	idxArgs := abi.Arguments{}
	for _, a := range ev.Inputs {
		if a.Indexed {
			idxArgs = append(idxArgs, a)
		}
	}
	if len(idxArgs) != len(indexedVals) {
		t.Fatalf("event %q: indexed count mismatch (have %d, got %d)", eventName, len(idxArgs), len(indexedVals))
	}
	for i, v := range indexedVals {
		topics = append(topics, toTopic(t, idxArgs[i].Type, v))
	}
	// Non-indexed args → data.
	data, err := ev.Inputs.NonIndexed().Pack(nonIndexedVals...)
	if err != nil {
		t.Fatalf("event %q: pack non-indexed: %v", eventName, err)
	}
	return &types.Log{
		Address: emitter,
		Topics:  topics,
		Data:    data,
	}
}

func toTopic(t *testing.T, ty abi.Type, v interface{}) common.Hash {
	t.Helper()
	switch ty.T {
	case abi.AddressTy:
		addr := v.(common.Address)
		var h common.Hash
		copy(h[12:], addr.Bytes())
		return h
	case abi.UintTy, abi.IntTy:
		// Indexed integers are left-padded to 32 bytes.
		var n *big.Int
		switch x := v.(type) {
		case *big.Int:
			n = x
		case uint16:
			n = new(big.Int).SetUint64(uint64(x))
		case uint8:
			n = new(big.Int).SetUint64(uint64(x))
		case uint64:
			n = new(big.Int).SetUint64(x)
		case int64:
			n = big.NewInt(x)
		default:
			t.Fatalf("toTopic: unsupported int type %T", v)
		}
		return common.BigToHash(n)
	case abi.StringTy:
		// Indexed strings are stored as keccak(value) — none of our events use this.
		return crypto.Keccak256Hash([]byte(v.(string)))
	default:
		t.Fatalf("toTopic: unsupported type %v", ty)
		return common.Hash{}
	}
}

func TestParseSwapExecuted(t *testing.T) {
	from := common.HexToAddress("0x1111111111111111111111111111111111111111")
	assetIn := common.HexToAddress("0x2222222222222222222222222222222222222222")
	assetOut := common.HexToAddress("0x3333333333333333333333333333333333333333")
	amountIn := big.NewInt(1_000_000)
	amountOut := big.NewInt(990_000)
	log := synthLog(t, afiParsedABI, common.Address{}, "SwapExecuted",
		[]interface{}{from, assetIn, assetOut},
		amountIn, amountOut,
	)
	evs, err := ParseSwapExecuted([]*types.Log{log})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(evs) != 1 {
		t.Fatalf("expected 1 event, got %d", len(evs))
	}
	got := evs[0]
	if got.From != from || got.AssetIn != assetIn || got.AssetOut != assetOut {
		t.Errorf("address fields mismatch: %+v", got)
	}
	if got.AmountIn.Cmp(amountIn) != 0 || got.AmountOut.Cmp(amountOut) != 0 {
		t.Errorf("amount fields mismatch: in=%s out=%s", got.AmountIn, got.AmountOut)
	}
}

func TestParseFeeCollected(t *testing.T) {
	tok := common.HexToAddress("0x4444444444444444444444444444444444444444")
	amt := big.NewInt(125)
	log := synthLog(t, afiParsedABI, common.Address{}, "FeeCollected",
		[]interface{}{tok}, amt,
	)
	evs, err := ParseFeeCollected([]*types.Log{log})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(evs) != 1 || evs[0].Token != tok || evs[0].Amount.Cmp(amt) != 0 {
		t.Errorf("FeeCollected mismatch: %+v", evs)
	}
}

func TestParseAfiTreasuryUpdated(t *testing.T) {
	tr := common.HexToAddress("0x5555555555555555555555555555555555555555")
	log := synthLog(t, afiParsedABI, common.Address{}, "TreasuryUpdated",
		[]interface{}{tr},
	)
	evs, err := ParseAfiTreasuryUpdated([]*types.Log{log})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(evs) != 1 || evs[0].Treasury != tr {
		t.Errorf("TreasuryUpdated mismatch: %+v", evs)
	}
}

func TestParseFeeBpsUpdated(t *testing.T) {
	log := synthLog(t, afiParsedABI, common.Address{}, "FeeBpsUpdated",
		nil, uint16(150),
	)
	evs, err := ParseFeeBpsUpdated([]*types.Log{log})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(evs) != 1 || evs[0].FeeBps != 150 {
		t.Errorf("FeeBpsUpdated mismatch: %+v", evs)
	}
}

func TestParseUserFeeBpsSet(t *testing.T) {
	user := common.HexToAddress("0x6666666666666666666666666666666666666666")
	log := synthLog(t, afiParsedABI, common.Address{}, "UserFeeBpsSet",
		[]interface{}{user}, uint16(77),
	)
	evs, err := ParseUserFeeBpsSet([]*types.Log{log})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(evs) != 1 || evs[0].User != user || evs[0].FeeBps != 77 {
		t.Errorf("UserFeeBpsSet mismatch: %+v", evs)
	}
}

func TestParseUserFeeBpsCleared(t *testing.T) {
	user := common.HexToAddress("0x7777777777777777777777777777777777777777")
	log := synthLog(t, afiParsedABI, common.Address{}, "UserFeeBpsCleared",
		[]interface{}{user},
	)
	evs, err := ParseUserFeeBpsCleared([]*types.Log{log})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(evs) != 1 || evs[0].User != user {
		t.Errorf("UserFeeBpsCleared mismatch: %+v", evs)
	}
}

func TestParseFlashLoanRequested(t *testing.T) {
	asset := common.HexToAddress("0x8888888888888888888888888888888888888888")
	amount := big.NewInt(5_000_000)
	log := synthLog(t, nmrParsedABI, common.Address{}, "FlashLoanRequested",
		[]interface{}{asset}, amount,
	)
	evs, err := ParseFlashLoanRequested([]*types.Log{log})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(evs) != 1 || evs[0].Asset != asset || evs[0].Amount.Cmp(amount) != 0 {
		t.Errorf("FlashLoanRequested mismatch: %+v", evs)
	}
}

func TestParseFlashLoanExecuted(t *testing.T) {
	asset := common.HexToAddress("0x9999999999999999999999999999999999999999")
	amount := big.NewInt(10_000_000)
	premium := big.NewInt(5_000)
	profit := big.NewInt(12_345)
	log := synthLog(t, nmrParsedABI, common.Address{}, "FlashLoanExecuted",
		[]interface{}{asset}, amount, premium, profit,
	)
	evs, err := ParseFlashLoanExecuted([]*types.Log{log})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(evs) != 1 {
		t.Fatalf("expected 1 event, got %d", len(evs))
	}
	g := evs[0]
	if g.Asset != asset || g.Amount.Cmp(amount) != 0 || g.Premium.Cmp(premium) != 0 || g.Profit.Cmp(profit) != 0 {
		t.Errorf("FlashLoanExecuted mismatch: %+v", g)
	}
}

func TestParseFlashLoanFailed(t *testing.T) {
	asset := common.HexToAddress("0xaaaa888888888888888888888888888888888888")
	amount := big.NewInt(1_000_000)
	reason := "insufficient out"
	log := synthLog(t, nmrParsedABI, common.Address{}, "FlashLoanFailed",
		[]interface{}{asset}, amount, reason,
	)
	evs, err := ParseFlashLoanFailed([]*types.Log{log})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(evs) != 1 || evs[0].Asset != asset || evs[0].Amount.Cmp(amount) != 0 || evs[0].Reason != reason {
		t.Errorf("FlashLoanFailed mismatch: %+v", evs)
	}
}

func TestParseProfitSwept(t *testing.T) {
	asset := common.HexToAddress("0xbbbb888888888888888888888888888888888888")
	to := common.HexToAddress("0xcccc888888888888888888888888888888888888")
	amount := big.NewInt(7_777)
	log := synthLog(t, nmrParsedABI, common.Address{}, "ProfitSwept",
		[]interface{}{asset, to}, amount,
	)
	evs, err := ParseProfitSwept([]*types.Log{log})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(evs) != 1 || evs[0].Asset != asset || evs[0].To != to || evs[0].Amount.Cmp(amount) != 0 {
		t.Errorf("ProfitSwept mismatch: %+v", evs)
	}
}

func TestParseProfitShareUpdated(t *testing.T) {
	log := synthLog(t, nmrParsedABI, common.Address{}, "ProfitShareUpdated",
		nil, uint8(42),
	)
	evs, err := ParseProfitShareUpdated([]*types.Log{log})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(evs) != 1 || evs[0].ProfitShare != 42 {
		t.Errorf("ProfitShareUpdated mismatch: %+v", evs)
	}
}

func TestParseSwapExecutedIgnoresUnrelatedLogs(t *testing.T) {
	// Random topic; not a SwapExecuted log.
	log := &types.Log{
		Topics: []common.Hash{crypto.Keccak256Hash([]byte("Unrelated()"))},
	}
	evs, err := ParseSwapExecuted([]*types.Log{log})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(evs) != 0 {
		t.Errorf("expected 0 events for unrelated topic, got %d", len(evs))
	}
}

func TestParseNMRTreasuryUpdated(t *testing.T) {
	tr := common.HexToAddress("0xdddd888888888888888888888888888888888888")
	log := synthLog(t, nmrParsedABI, common.Address{}, "TreasuryUpdated",
		[]interface{}{tr},
	)
	evs, err := ParseNMRTreasuryUpdated([]*types.Log{log})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(evs) != 1 || evs[0].Treasury != tr {
		t.Errorf("NMR TreasuryUpdated mismatch: %+v", evs)
	}
}
