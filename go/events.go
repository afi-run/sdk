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

// TreasuryUpdatedEvent is emitted by Afi (and also by NMR — disambiguate by Raw.Address).
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

// ---------- NMR events ----------

// FlashLoanRequestedEvent is emitted by NMR before initiating an Aave flash loan.
type FlashLoanRequestedEvent struct {
	Asset  common.Address
	Amount *big.Int
	Raw    *types.Log
}

// FlashLoanExecutedEvent is emitted by NMR on a successful arbitrage cycle.
type FlashLoanExecutedEvent struct {
	Asset   common.Address
	Amount  *big.Int
	Premium *big.Int
	Profit  *big.Int
	Raw     *types.Log
}

// FlashLoanFailedEvent is emitted by NMR when the inner cycle reverts with a string reason.
type FlashLoanFailedEvent struct {
	Asset  common.Address
	Amount *big.Int
	Reason string
	Raw    *types.Log
}

// FlashLoanFailedWithDataEvent is emitted by NMR when the inner cycle reverts
// with low-level data (no decodable string reason).
type FlashLoanFailedWithDataEvent struct {
	Asset  common.Address
	Amount *big.Int
	Data   []byte
	Raw    *types.Log
}

// NMRSwapExecutedEvent is emitted by NMR on a successful cycle swap (NMR.swap).
// Note this is distinct from Afi's SwapExecuted (it carries no `from` field and
// has a different topic0).
type NMRSwapExecutedEvent struct {
	AssetIn   common.Address
	AmountIn  *big.Int
	AssetOut  common.Address
	AmountOut *big.Int
	Raw       *types.Log
}

// ProfitSweptEvent is emitted by NMR when operator pulls profit to treasury.
type ProfitSweptEvent struct {
	Asset  common.Address
	Amount *big.Int
	To     common.Address
	Raw    *types.Log
}

// ProfitShareUpdatedEvent is emitted by NMR when owner changes profit share split.
type ProfitShareUpdatedEvent struct {
	ProfitShare uint8
	Raw         *types.Log
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
// known to be Afi-emitted (filter by log.Address). NMR emits an event with the
// same name + same topic0, so do not pass mixed logs.
func ParseAfiTreasuryUpdated(logs []*types.Log) ([]TreasuryUpdatedEvent, error) {
	return parseTreasuryUpdated(afiParsedABI, logs)
}

// ParseNMRTreasuryUpdated decodes NMR.TreasuryUpdated events. Same caveat as
// ParseAfiTreasuryUpdated — filter by emitter address before calling.
func ParseNMRTreasuryUpdated(logs []*types.Log) ([]TreasuryUpdatedEvent, error) {
	return parseTreasuryUpdated(nmrParsedABI, logs)
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

// ---------- NMR parsers ----------

// ParseFlashLoanRequested decodes all NMR FlashLoanRequested events.
func ParseFlashLoanRequested(logs []*types.Log) ([]FlashLoanRequestedEvent, error) {
	var out []FlashLoanRequestedEvent
	for _, l := range logs {
		if !matchesEvent(nmrParsedABI, "FlashLoanRequested", l) {
			continue
		}
		var tmp struct {
			Asset  common.Address
			Amount *big.Int
		}
		if err := decodeEventLog(nmrParsedABI, "FlashLoanRequested", l, &tmp); err != nil {
			return nil, err
		}
		out = append(out, FlashLoanRequestedEvent{Asset: tmp.Asset, Amount: tmp.Amount, Raw: l})
	}
	return out, nil
}

// ParseFlashLoanExecuted decodes all NMR FlashLoanExecuted events.
func ParseFlashLoanExecuted(logs []*types.Log) ([]FlashLoanExecutedEvent, error) {
	var out []FlashLoanExecutedEvent
	for _, l := range logs {
		if !matchesEvent(nmrParsedABI, "FlashLoanExecuted", l) {
			continue
		}
		var tmp struct {
			Asset   common.Address
			Amount  *big.Int
			Premium *big.Int
			Profit  *big.Int
		}
		if err := decodeEventLog(nmrParsedABI, "FlashLoanExecuted", l, &tmp); err != nil {
			return nil, err
		}
		out = append(out, FlashLoanExecutedEvent{
			Asset: tmp.Asset, Amount: tmp.Amount, Premium: tmp.Premium, Profit: tmp.Profit, Raw: l,
		})
	}
	return out, nil
}

// ParseFlashLoanFailed decodes all NMR FlashLoanFailed events (string reason variant).
func ParseFlashLoanFailed(logs []*types.Log) ([]FlashLoanFailedEvent, error) {
	var out []FlashLoanFailedEvent
	for _, l := range logs {
		if !matchesEvent(nmrParsedABI, "FlashLoanFailed", l) {
			continue
		}
		var tmp struct {
			Asset  common.Address
			Amount *big.Int
			Reason string
		}
		if err := decodeEventLog(nmrParsedABI, "FlashLoanFailed", l, &tmp); err != nil {
			return nil, err
		}
		out = append(out, FlashLoanFailedEvent{
			Asset: tmp.Asset, Amount: tmp.Amount, Reason: tmp.Reason, Raw: l,
		})
	}
	return out, nil
}

// ParseFlashLoanFailedWithData decodes all NMR FlashLoanFailedWithData events
// (low-level revert data variant). Pair this with ParseFlashLoanFailed: the
// contract emits one or the other depending on whether the revert carried a
// decodable string reason.
func ParseFlashLoanFailedWithData(logs []*types.Log) ([]FlashLoanFailedWithDataEvent, error) {
	var out []FlashLoanFailedWithDataEvent
	for _, l := range logs {
		if !matchesEvent(nmrParsedABI, "FlashLoanFailedWithData", l) {
			continue
		}
		var tmp struct {
			Asset  common.Address
			Amount *big.Int
			Data   []byte
		}
		if err := decodeEventLog(nmrParsedABI, "FlashLoanFailedWithData", l, &tmp); err != nil {
			return nil, err
		}
		out = append(out, FlashLoanFailedWithDataEvent{
			Asset: tmp.Asset, Amount: tmp.Amount, Data: tmp.Data, Raw: l,
		})
	}
	return out, nil
}

// ParseNMRSwapExecuted decodes all NMR-emitted SwapExecuted events (the cycle
// swap variant, distinct from Afi's). Use ParseSwapExecuted for Afi logs.
func ParseNMRSwapExecuted(logs []*types.Log) ([]NMRSwapExecutedEvent, error) {
	var out []NMRSwapExecutedEvent
	for _, l := range logs {
		if !matchesEvent(nmrParsedABI, "SwapExecuted", l) {
			continue
		}
		var tmp struct {
			AssetIn   common.Address
			AmountIn  *big.Int
			AssetOut  common.Address
			AmountOut *big.Int
		}
		if err := decodeEventLog(nmrParsedABI, "SwapExecuted", l, &tmp); err != nil {
			return nil, err
		}
		out = append(out, NMRSwapExecutedEvent{
			AssetIn: tmp.AssetIn, AmountIn: tmp.AmountIn,
			AssetOut: tmp.AssetOut, AmountOut: tmp.AmountOut, Raw: l,
		})
	}
	return out, nil
}

// ParseProfitSwept decodes all NMR ProfitSwept events.
func ParseProfitSwept(logs []*types.Log) ([]ProfitSweptEvent, error) {
	var out []ProfitSweptEvent
	for _, l := range logs {
		if !matchesEvent(nmrParsedABI, "ProfitSwept", l) {
			continue
		}
		var tmp struct {
			Asset  common.Address
			Amount *big.Int
			To     common.Address
		}
		if err := decodeEventLog(nmrParsedABI, "ProfitSwept", l, &tmp); err != nil {
			return nil, err
		}
		out = append(out, ProfitSweptEvent{Asset: tmp.Asset, Amount: tmp.Amount, To: tmp.To, Raw: l})
	}
	return out, nil
}

// ParseProfitShareUpdated decodes all NMR ProfitShareUpdated events.
func ParseProfitShareUpdated(logs []*types.Log) ([]ProfitShareUpdatedEvent, error) {
	var out []ProfitShareUpdatedEvent
	for _, l := range logs {
		if !matchesEvent(nmrParsedABI, "ProfitShareUpdated", l) {
			continue
		}
		var tmp struct {
			ProfitShare uint8
		}
		if err := decodeEventLog(nmrParsedABI, "ProfitShareUpdated", l, &tmp); err != nil {
			return nil, err
		}
		out = append(out, ProfitShareUpdatedEvent{ProfitShare: tmp.ProfitShare, Raw: l})
	}
	return out, nil
}
