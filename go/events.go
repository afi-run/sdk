package afi

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ---------- Afi events ----------

// SwapExecutedEvent is emitted by Afi on every successful swap.
type SwapExecutedEvent struct {
	From      common.Address
	AssetIn   common.Address
	AmountIn  *big.Int
	AssetOut  common.Address
	AmountOut *big.Int
	Raw       *types.Log
}

// FeeCollectedEvent is emitted by Afi when a non-zero fee is deducted.
type FeeCollectedEvent struct {
	Token  common.Address
	Amount *big.Int
	Raw    *types.Log
}

// TreasuryUpdatedEvent is emitted by Afi.
type TreasuryUpdatedEvent struct {
	Treasury common.Address
	Raw      *types.Log
}

// FeeBpsUpdatedEvent is emitted by Afi when the default fee changes.
type FeeBpsUpdatedEvent struct {
	FeeBps uint16
	Raw    *types.Log
}

// UserFeeBpsSetEvent is emitted by Afi when a user-specific fee override is written.
type UserFeeBpsSetEvent struct {
	User   common.Address
	FeeBps uint16
	Raw    *types.Log
}

// UserFeeBpsClearedEvent is emitted by Afi when a user-specific override is cleared.
type UserFeeBpsClearedEvent struct {
	User common.Address
	Raw  *types.Log
}

// ---------- internal helpers ----------

// decodeEventLog locates `name` in `parsed`, verifies the topic[0] hash, and
// unpacks indexed topics + non-indexed data into `out`.
func decodeEventLog(parsed abi.ABI, name string, log *types.Log, out interface{}) error {
	ev, ok := parsed.Events[name]
	if !ok {
		return fmt.Errorf("event %q not in ABI", name)
	}
	if len(log.Topics) == 0 || log.Topics[0] != ev.ID {
		return fmt.Errorf("event %q: topic0 mismatch", name)
	}
	// Non-indexed (data) fields.
	if len(log.Data) > 0 {
		if err := parsed.UnpackIntoInterface(out, name, log.Data); err != nil {
			return fmt.Errorf("event %q: unpack data: %w", name, err)
		}
	}
	// Indexed (topic) fields.
	var indexed abi.Arguments
	for _, arg := range ev.Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	if len(indexed) > 0 {
		if err := abi.ParseTopics(out, indexed, log.Topics[1:]); err != nil {
			return fmt.Errorf("event %q: parse topics: %w", name, err)
		}
	}
	return nil
}

// matchesEvent returns true if `log` could be the named event from `parsed`.
func matchesEvent(parsed abi.ABI, name string, log *types.Log) bool {
	ev, ok := parsed.Events[name]
	if !ok {
		return false
	}
	return len(log.Topics) > 0 && log.Topics[0] == ev.ID
}

// ---------- Afi parsers ----------

// ParseSwapExecuted decodes all SwapExecuted events from `logs` (Afi-emitted).
func ParseSwapExecuted(logs []*types.Log) ([]SwapExecutedEvent, error) {
	var out []SwapExecutedEvent
	for _, l := range logs {
		if !matchesEvent(afiParsedABI, "SwapExecuted", l) {
			continue
		}
		var tmp struct {
			From      common.Address
			AssetIn   common.Address
			AmountIn  *big.Int
			AssetOut  common.Address
			AmountOut *big.Int
		}
		if err := decodeEventLog(afiParsedABI, "SwapExecuted", l, &tmp); err != nil {
			return nil, err
		}
		out = append(out, SwapExecutedEvent{
			From: tmp.From, AssetIn: tmp.AssetIn, AmountIn: tmp.AmountIn,
			AssetOut: tmp.AssetOut, AmountOut: tmp.AmountOut, Raw: l,
		})
	}
	return out, nil
}

// ParseFeeCollected decodes all Afi FeeCollected events.
func ParseFeeCollected(logs []*types.Log) ([]FeeCollectedEvent, error) {
	var out []FeeCollectedEvent
	for _, l := range logs {
		if !matchesEvent(afiParsedABI, "FeeCollected", l) {
			continue
		}
		var tmp struct {
			Token  common.Address
			Amount *big.Int
		}
		if err := decodeEventLog(afiParsedABI, "FeeCollected", l, &tmp); err != nil {
			return nil, err
		}
		out = append(out, FeeCollectedEvent{Token: tmp.Token, Amount: tmp.Amount, Raw: l})
	}
	return out, nil
}

// ParseAfiTreasuryUpdated decodes Afi.TreasuryUpdated events. Use only on logs
// known to be Afi-emitted (filter by log.Address).
func ParseAfiTreasuryUpdated(logs []*types.Log) ([]TreasuryUpdatedEvent, error) {
	return parseTreasuryUpdated(afiParsedABI, logs)
}

func parseTreasuryUpdated(parsed abi.ABI, logs []*types.Log) ([]TreasuryUpdatedEvent, error) {
	var out []TreasuryUpdatedEvent
	for _, l := range logs {
		if !matchesEvent(parsed, "TreasuryUpdated", l) {
			continue
		}
		var tmp struct {
			Treasury common.Address
		}
		if err := decodeEventLog(parsed, "TreasuryUpdated", l, &tmp); err != nil {
			return nil, err
		}
		out = append(out, TreasuryUpdatedEvent{Treasury: tmp.Treasury, Raw: l})
	}
	return out, nil
}

// ParseFeeBpsUpdated decodes all Afi FeeBpsUpdated events.
func ParseFeeBpsUpdated(logs []*types.Log) ([]FeeBpsUpdatedEvent, error) {
	var out []FeeBpsUpdatedEvent
	for _, l := range logs {
		if !matchesEvent(afiParsedABI, "FeeBpsUpdated", l) {
			continue
		}
		var tmp struct {
			FeeBps uint16
		}
		if err := decodeEventLog(afiParsedABI, "FeeBpsUpdated", l, &tmp); err != nil {
			return nil, err
		}
		out = append(out, FeeBpsUpdatedEvent{FeeBps: tmp.FeeBps, Raw: l})
	}
	return out, nil
}

// ParseUserFeeBpsSet decodes all Afi UserFeeBpsSet events.
func ParseUserFeeBpsSet(logs []*types.Log) ([]UserFeeBpsSetEvent, error) {
	var out []UserFeeBpsSetEvent
	for _, l := range logs {
		if !matchesEvent(afiParsedABI, "UserFeeBpsSet", l) {
			continue
		}
		var tmp struct {
			User   common.Address
			FeeBps uint16
		}
		if err := decodeEventLog(afiParsedABI, "UserFeeBpsSet", l, &tmp); err != nil {
			return nil, err
		}
		out = append(out, UserFeeBpsSetEvent{User: tmp.User, FeeBps: tmp.FeeBps, Raw: l})
	}
	return out, nil
}

// ParseUserFeeBpsCleared decodes all Afi UserFeeBpsCleared events.
func ParseUserFeeBpsCleared(logs []*types.Log) ([]UserFeeBpsClearedEvent, error) {
	var out []UserFeeBpsClearedEvent
	for _, l := range logs {
		if !matchesEvent(afiParsedABI, "UserFeeBpsCleared", l) {
			continue
		}
		var tmp struct {
			User common.Address
		}
		if err := decodeEventLog(afiParsedABI, "UserFeeBpsCleared", l, &tmp); err != nil {
			return nil, err
		}
		out = append(out, UserFeeBpsClearedEvent{User: tmp.User, Raw: l})
	}
	return out, nil
}
