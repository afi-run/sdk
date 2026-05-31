package afi

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
)

// StateOverride is the JSON-RPC eth_call stateOverride payload format.
// Maps an address to per-account overrides (balance, code, storage, etc).
type StateOverride map[common.Address]AccountOverride

// AccountOverride applies to a single address. Only StateDiff is used by the
// quoter simulator (to fake a token balance for the quoter contract).
type AccountOverride struct {
	StateDiff map[common.Hash]common.Hash `json:"stateDiff,omitempty"`
	// Balance, Code, Nonce, State omitted — quoter only needs StateDiff.
}

// MarshalJSON converts the map[Hash]Hash to a JSON object of {hex: hex} pairs
// — the format expected by eth_call's stateOverride parameter.
func (a AccountOverride) MarshalJSON() ([]byte, error) {
	out := struct {
		StateDiff map[string]string `json:"stateDiff,omitempty"`
	}{}
	if len(a.StateDiff) > 0 {
		out.StateDiff = make(map[string]string, len(a.StateDiff))
		for k, v := range a.StateDiff {
			out.StateDiff[k.Hex()] = v.Hex()
		}
	}
	return json.Marshal(out)
}

// BuildBalanceOverride returns a state_override that gives `holder` a virtual
// balance of `amount` of `token` on `chainID`.
//
// Returns an error if the token's balance slot is not in the static map and
// auto-detection is disabled (callers should use DetectBalanceSlot to seed
// the map for unknown tokens).
func BuildBalanceOverride(chainID int64, token, holder common.Address, amount *big.Int) (StateOverride, error) {
	slot, ok := LookupBalanceSlot(chainID, token)
	if !ok {
		return nil, fmt.Errorf("afi: no static balance slot for token %s on chain %d (call DetectBalanceSlot to seed)", token.Hex(), chainID)
	}
	derivedSlot := mappingSlot(holder, slot)
	return StateOverride{
		token: {
			StateDiff: map[common.Hash]common.Hash{
				derivedSlot: common.BigToHash(amount),
			},
		},
	}, nil
}

// mappingSlot returns keccak256(key . slot) — the storage location of a single
// mapping entry. `key` is left-padded to 32 bytes, slot is right-aligned uint.
func mappingSlot(key common.Address, slot uint64) common.Hash {
	var buf [64]byte
	// key padded left
	copy(buf[12:32], key.Bytes())
	// slot padded left
	slotBig := new(big.Int).SetUint64(slot).Bytes()
	copy(buf[64-len(slotBig):], slotBig)
	return crypto.Keccak256Hash(buf[:])
}

// DetectBalanceSlot brute-forces slots 0..maxSlot trying each one until a
// state_override produces a matching balanceOf reading. Slow (1 eth_call per
// slot tested) but works for any ERC20 once.
//
// On success, registers the slot in the static map so subsequent calls hit fast.
func DetectBalanceSlot(ctx context.Context, rpcClient *rpc.Client, chainID int64, token common.Address, maxSlot uint64) (uint64, error) {
	const sentinelAmount = uint64(0xdeadbeefdeadbeef)
	holder := common.HexToAddress("0x0000000000000000000000000000000000001234")
	want := new(big.Int).SetUint64(sentinelAmount)

	balanceOfData := append(
		[]byte{0x70, 0xa0, 0x82, 0x31}, // balanceOf(address) selector
		common.LeftPadBytes(holder.Bytes(), 32)...,
	)

	for slot := uint64(0); slot <= maxSlot; slot++ {
		override := StateOverride{
			token: {
				StateDiff: map[common.Hash]common.Hash{
					mappingSlot(holder, slot): common.BigToHash(want),
				},
			},
		}

		var raw string
		err := rpcClient.CallContext(ctx, &raw,
			"eth_call",
			map[string]interface{}{
				"to":   token.Hex(),
				"data": "0x" + common.Bytes2Hex(balanceOfData),
			},
			"latest",
			override,
		)
		if err != nil {
			continue
		}

		got := new(big.Int).SetBytes(common.FromHex(raw))
		if got.Cmp(want) == 0 {
			RegisterBalanceSlot(chainID, token, slot)
			return slot, nil
		}
	}
	return 0, fmt.Errorf("afi: balance slot not detected for token %s within %d slots", token.Hex(), maxSlot)
}
