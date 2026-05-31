package afi

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// sendOptions is the resolved bag of optional knobs for SendContractTx.
type sendOptions struct {
	value              *big.Int
	confirmations      uint64
	timeoutMs          int64
	nonce              *uint64
	gasBuffer          *uint // pointer so 0 is meaningful vs. unset
	skipAllowanceCheck bool
}

// SendOption configures SendContractTx.
type SendOption func(*sendOptions)

// WithValue attaches a payable ether value to the transaction.
func WithValue(v *big.Int) SendOption {
	return func(o *sendOptions) { o.value = v }
}

// WithConfirmations overrides the number of confirmations to wait for (default 1).
func WithConfirmations(n uint64) SendOption {
	return func(o *sendOptions) { o.confirmations = n }
}

// WithTimeoutMs aborts the receipt wait after the given milliseconds.
func WithTimeoutMs(ms int64) SendOption {
	return func(o *sendOptions) { o.timeoutMs = ms }
}

// WithNonce pins the tx to a specific nonce, bypassing the managed counter.
func WithNonce(n uint64) SendOption {
	return func(o *sendOptions) { o.nonce = &n }
}

// WithGasBuffer overrides the per-call gas buffer (percent). Pass 0 to disable.
func WithGasBuffer(pct uint) SendOption {
	return func(o *sendOptions) { o.gasBuffer = &pct }
}

// WithoutAllowancePrecheck disables the off-chain allowance precheck performed
// by SwapFor / NMRLoanArbitrage. Use when you know the on-chain state changed
// inside the same block (e.g. you just sent the approve in this very batch).
func WithoutAllowancePrecheck() SendOption {
	return func(o *sendOptions) { o.skipAllowanceCheck = true }
}

// precheckEnabled resolves the SendOptions to decide whether to run the
// off-chain allowance precheck. Defaults to true.
func precheckEnabled(opts []SendOption) bool {
	o := sendOptions{}
	for _, opt := range opts {
		opt(&o)
	}
	return !o.skipAllowanceCheck
}

// SendContractTx encodes fn(args…) against parsedABI, signs it, sends it, waits
// for the receipt, and returns it. Reuses the same gas-buffer + managed-nonce
// machinery as ExecuteSwap.
//
// Returns an error before sending when:
//   - no signer is configured;
//   - encoding fn(args) against parsedABI fails;
//   - eth_estimateGas reverts (revert reason is surfaced when decodable).
func (c *Client) SendContractTx(
	ctx context.Context,
	to common.Address,
	parsedABI abi.ABI,
	fn string,
	args []interface{},
	opts ...SendOption,
) (*types.Receipt, error) {
	if err := c.requireSigner(); err != nil {
		return nil, err
	}

	o := sendOptions{confirmations: 1}
	for _, opt := range opts {
		opt(&o)
	}

	data, err := parsedABI.Pack(fn, args...)
	if err != nil {
		return nil, fmt.Errorf("SendContractTx: pack %s: %w", fn, err)
	}

	gasBuffer := c.gasBufferPct
	if o.gasBuffer != nil {
		gasBuffer = *o.gasBuffer
	}

	txHash, err := c.sendTxWithValue(ctx, &to, data, gasBuffer, o.nonce, o.value)
	if err != nil {
		return nil, err
	}

	receipt, err := c.waitReceiptWithOpts(ctx, txHash, WaitForTxOptions{
		Confirmations: o.confirmations,
		TimeoutMs:     o.timeoutMs,
	})
	if err != nil {
		return nil, err
	}
	if receipt.Status == types.ReceiptStatusFailed {
		return receipt, fmt.Errorf("SendContractTx: %s reverted on-chain (tx %s)", fn, txHash)
	}
	return receipt, nil
}

// sendTxWithValue is sendTx + an explicit msg.value.
// Mirrors sendTx in swap.go; kept distinct to avoid touching the swap-critical path.
func (c *Client) sendTxWithValue(
	ctx context.Context,
	to *common.Address,
	data []byte,
	gasBuffer uint,
	nonceOverride *uint64,
	value *big.Int,
) (string, error) {
	from := crypto.PubkeyToAddress(c.key.PublicKey)

	chainID, err := c.ChainID(ctx)
	if err != nil {
		return "", fmt.Errorf("chain id: %w", err)
	}

	nonce, err := c.allocateNonce(ctx, nonceOverride, from)
	if err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}

	callMsg := ethereum.CallMsg{From: from, To: to, Data: data}
	if value != nil && value.Sign() > 0 {
		callMsg.Value = new(big.Int).Set(value)
	}

	gasLimit, err := c.eth.EstimateGas(ctx, callMsg)
	if err != nil {
		if reason := callRevertReason(ctx, c.eth, from, to, data); reason != "" {
			return "", fmt.Errorf("estimate gas: %s", reason)
		}
		return "", fmt.Errorf("estimate gas: %w", err)
	}
	gasLimit = applyGasBuffer(gasLimit, gasBuffer)

	tip, err := c.eth.SuggestGasTipCap(ctx)
	if err != nil {
		return "", fmt.Errorf("gas tip: %w", err)
	}

	head, err := c.eth.HeaderByNumber(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("header: %w", err)
	}

	maxFee := new(big.Int).Add(new(big.Int).Mul(head.BaseFee, big.NewInt(2)), tip)

	txValue := new(big.Int)
	if value != nil {
		txValue.Set(value)
	}

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		To:        to,
		Gas:       gasLimit,
		GasTipCap: tip,
		GasFeeCap: maxFee,
		Value:     txValue,
		Data:      data,
	})

	signed, err := types.SignTx(tx, types.NewLondonSigner(chainID), c.key)
	if err != nil {
		return "", fmt.Errorf("sign: %w", err)
	}
	if err := c.eth.SendTransaction(ctx, signed); err != nil {
		return "", fmt.Errorf("send: %w", err)
	}
	return signed.Hash().Hex(), nil
}
