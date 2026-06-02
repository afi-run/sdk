package afi

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// SwapForRequest is a single entry for batchSwapFor.
type SwapForRequest struct {
	User     common.Address
	TokenIn  common.Address
	AmountIn *big.Int
	TokenOut common.Address
	MinOut   *big.Int
	Params   []byte
}

// afiAddressForCtx resolves the deployed Afi router for the chain the RPC is
// connected to, falling back to AfiAddress when no per-network entry exists.
func (c *Client) afiAddressForCtx(ctx context.Context) (common.Address, error) {
	net, err := c.DetectNetwork(ctx)
	if err != nil {
		return common.Address{}, err
	}
	if addr, ok := AfiAddresses[net]; ok && addr != (common.Address{}) {
		return addr, nil
	}
	if AfiAddress != (common.Address{}) {
		return AfiAddress, nil
	}
	return common.Address{}, fmt.Errorf("no Afi address known for network %q", net)
}

// SwapFor encodes Afi.swapFor(user, tokenIn, amountIn, tokenOut, minOut, params)
// and submits it. Operator-only.
//
// By default a cheap precheck reads ERC20(tokenIn).allowance(user, AFI) and
// errors out before sending the tx when the allowance is short. Disable via
// WithoutAllowancePrecheck if you want the on-chain revert path instead.
func (c *Client) SwapFor(
	ctx context.Context,
	user, tokenIn common.Address,
	amountIn *big.Int,
	tokenOut common.Address,
	minOut *big.Int,
	params []byte,
	opts ...SendOption,
) (*types.Receipt, error) {
	to, err := c.afiAddressForCtx(ctx)
	if err != nil {
		return nil, err
	}
	if precheckEnabled(opts) {
		allowance, err := getAllowanceFor(ctx, c.eth, c.erc20, tokenIn, user, to)
		if err != nil {
			return nil, fmt.Errorf("SwapFor allowance precheck: %w", err)
		}
		if allowance.Cmp(amountIn) < 0 {
			return nil, ErrInsufficientAllowance(tokenIn.Hex(), user.Hex(), to.Hex(), allowance, amountIn)
		}
	}
	return c.SendContractTx(ctx, to, afiParsedABI, "swapFor",
		[]interface{}{user, tokenIn, amountIn, tokenOut, minOut, params}, opts...)
}

// BatchSwapFor encodes Afi.batchSwapFor(swaps[]) and submits it. Operator-only.
func (c *Client) BatchSwapFor(ctx context.Context, swaps []SwapForRequest, opts ...SendOption) (*types.Receipt, error) {
	to, err := c.afiAddressForCtx(ctx)
	if err != nil {
		return nil, err
	}
	// The ABI tuple component order: user, tokenIn, amountIn, tokenOut, minOut, params.
	type swapTuple struct {
		User     common.Address
		TokenIn  common.Address
		AmountIn *big.Int
		TokenOut common.Address
		MinOut   *big.Int
		Params   []byte
	}
	encoded := make([]swapTuple, len(swaps))
	for i, s := range swaps {
		encoded[i] = swapTuple{s.User, s.TokenIn, s.AmountIn, s.TokenOut, s.MinOut, s.Params}
	}
	return c.SendContractTx(ctx, to, afiParsedABI, "batchSwapFor", []interface{}{encoded}, opts...)
}

// ───────────────────────── Admin wrappers ────────────────────────────────────

// AdminPause calls Afi.pause(). Owner-only.
func (c *Client) AdminPause(ctx context.Context, opts ...SendOption) (*types.Receipt, error) {
	to, err := c.afiAddressForCtx(ctx)
	if err != nil {
		return nil, err
	}
	return c.SendContractTx(ctx, to, afiParsedABI, "pause", nil, opts...)
}

// AdminUnpause calls Afi.unpause(). Owner-only.
func (c *Client) AdminUnpause(ctx context.Context, opts ...SendOption) (*types.Receipt, error) {
	to, err := c.afiAddressForCtx(ctx)
	if err != nil {
		return nil, err
	}
	return c.SendContractTx(ctx, to, afiParsedABI, "unpause", nil, opts...)
}

// AdminSetTreasury calls Afi.setTreasury(addr). Owner-only.
func (c *Client) AdminSetTreasury(ctx context.Context, addr common.Address, opts ...SendOption) (*types.Receipt, error) {
	if addr == (common.Address{}) {
		return nil, fmt.Errorf("AdminSetTreasury: addr cannot be zero")
	}
	to, err := c.afiAddressForCtx(ctx)
	if err != nil {
		return nil, err
	}
	return c.SendContractTx(ctx, to, afiParsedABI, "setTreasury", []interface{}{addr}, opts...)
}

// AdminSetFeeBps calls Afi.setFeeBps(bps). Owner-only. Rejects values > MaxFeeBps.
func (c *Client) AdminSetFeeBps(ctx context.Context, bps uint16, opts ...SendOption) (*types.Receipt, error) {
	if bps > MaxFeeBps {
		return nil, fmt.Errorf("AdminSetFeeBps: bps %d exceeds max %d", bps, MaxFeeBps)
	}
	to, err := c.afiAddressForCtx(ctx)
	if err != nil {
		return nil, err
	}
	return c.SendContractTx(ctx, to, afiParsedABI, "setFeeBps", []interface{}{bps}, opts...)
}

// AdminSetUserFeeBps calls Afi.setUserFeeBps(user, bps). Owner-only.
func (c *Client) AdminSetUserFeeBps(ctx context.Context, user common.Address, bps uint16, opts ...SendOption) (*types.Receipt, error) {
	if bps > MaxFeeBps {
		return nil, fmt.Errorf("AdminSetUserFeeBps: bps %d exceeds max %d", bps, MaxFeeBps)
	}
	to, err := c.afiAddressForCtx(ctx)
	if err != nil {
		return nil, err
	}
	return c.SendContractTx(ctx, to, afiParsedABI, "setUserFeeBps", []interface{}{user, bps}, opts...)
}

// AdminSetUserFeeBpsBatch calls Afi.setUserFeeBpsBatch(users, bps). Owner-only.
func (c *Client) AdminSetUserFeeBpsBatch(ctx context.Context, users []common.Address, bps []uint16, opts ...SendOption) (*types.Receipt, error) {
	if len(users) != len(bps) {
		return nil, fmt.Errorf("AdminSetUserFeeBpsBatch: users (%d) and bps (%d) length mismatch", len(users), len(bps))
	}
	for i, f := range bps {
		if f > MaxFeeBps {
			return nil, fmt.Errorf("AdminSetUserFeeBpsBatch: bps[%d]=%d exceeds max %d", i, f, MaxFeeBps)
		}
	}
	to, err := c.afiAddressForCtx(ctx)
	if err != nil {
		return nil, err
	}
	return c.SendContractTx(ctx, to, afiParsedABI, "setUserFeeBpsBatch", []interface{}{users, bps}, opts...)
}

// AdminClearUserFeeBps calls Afi.clearUserFeeBps(user). Owner-only.
func (c *Client) AdminClearUserFeeBps(ctx context.Context, user common.Address, opts ...SendOption) (*types.Receipt, error) {
	to, err := c.afiAddressForCtx(ctx)
	if err != nil {
		return nil, err
	}
	return c.SendContractTx(ctx, to, afiParsedABI, "clearUserFeeBps", []interface{}{user}, opts...)
}

// AdminResetAnyUserOverride calls Afi.resetAnyUserOverride(). Owner-only.
func (c *Client) AdminResetAnyUserOverride(ctx context.Context, opts ...SendOption) (*types.Receipt, error) {
	to, err := c.afiAddressForCtx(ctx)
	if err != nil {
		return nil, err
	}
	return c.SendContractTx(ctx, to, afiParsedABI, "resetAnyUserOverride", nil, opts...)
}

// AdminAddRule calls Afi.addRule(rule). Owner-only.
func (c *Client) AdminAddRule(ctx context.Context, rule common.Address, opts ...SendOption) (*types.Receipt, error) {
	if rule == (common.Address{}) {
		return nil, fmt.Errorf("AdminAddRule: rule cannot be zero address")
	}
	to, err := c.afiAddressForCtx(ctx)
	if err != nil {
		return nil, err
	}
	return c.SendContractTx(ctx, to, afiParsedABI, "addRule", []interface{}{rule}, opts...)
}

// AdminClearRules calls Afi.clearRules(). Owner-only.
func (c *Client) AdminClearRules(ctx context.Context, opts ...SendOption) (*types.Receipt, error) {
	to, err := c.afiAddressForCtx(ctx)
	if err != nil {
		return nil, err
	}
	return c.SendContractTx(ctx, to, afiParsedABI, "clearRules", nil, opts...)
}

// AdminSetOperator calls Afi.setOperator(op, value). Owner-only.
func (c *Client) AdminSetOperator(ctx context.Context, op common.Address, value bool, opts ...SendOption) (*types.Receipt, error) {
	to, err := c.afiAddressForCtx(ctx)
	if err != nil {
		return nil, err
	}
	return c.SendContractTx(ctx, to, afiParsedABI, "setOperator", []interface{}{op, value}, opts...)
}

// AdminRescueTokens calls Afi.rescueTokens(token, value, to). Owner-only.
func (c *Client) AdminRescueTokens(ctx context.Context, token common.Address, value *big.Int, to common.Address, opts ...SendOption) (*types.Receipt, error) {
	dest, err := c.afiAddressForCtx(ctx)
	if err != nil {
		return nil, err
	}
	return c.SendContractTx(ctx, dest, afiParsedABI, "rescueTokens", []interface{}{token, value, to}, opts...)
}

// ───────────────────────── Ownable2Step helpers ──────────────────────────────

// AcceptOwnership calls Ownable2Step.acceptOwnership() on the given contract.
// The caller MUST be the pending owner — otherwise the tx reverts on-chain.
func (c *Client) AcceptOwnership(ctx context.Context, contractAddr common.Address, opts ...SendOption) (*types.Receipt, error) {
	return c.SendContractTx(ctx, contractAddr, ownable2StepABI, "acceptOwnership", nil, opts...)
}

// AcceptOwnershipBatch accepts ownership on multiple contracts sequentially.
// Returns the receipts collected so far on first failure (so the caller can
// see how far the handover got).
func (c *Client) AcceptOwnershipBatch(ctx context.Context, contracts []common.Address, opts ...SendOption) ([]*types.Receipt, error) {
	receipts := make([]*types.Receipt, 0, len(contracts))
	for _, addr := range contracts {
		r, err := c.AcceptOwnership(ctx, addr, opts...)
		if err != nil {
			return receipts, fmt.Errorf("acceptOwnership %s: %w", addr.Hex(), err)
		}
		receipts = append(receipts, r)
	}
	return receipts, nil
}
