package afi

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func simulateSwap(ctx context.Context, client *ethclient.Client, afiABI abi.ABI, q *Quote, from common.Address) error {
	input, err := afiABI.Pack("swap", q.TokenIn, q.AmountInWei, q.TokenOut, q.MinOutWei, q.Steps)
	if err != nil {
		return fmt.Errorf("pack swap: %w", err)
	}

	_, err = client.CallContract(ctx, ethereum.CallMsg{
		From: from,
		To:   &AfiAddress,
		Data: input,
	}, nil)
	if err != nil {
		if data := extractRevertData(err); data != nil {
			if decoded := DecodeRevertReason(data); decoded != nil {
				return ErrSimulationDecoded(decoded.String(), decoded)
			}
		}
		return ErrSimulation(err.Error())
	}
	return nil
}

func submitSwap(ctx context.Context, c *Client, afiABI abi.ABI, q *Quote, gasBuffer uint, nonceOverride *uint64) (*PendingSwap, error) {
	input, err := afiABI.Pack("swap", q.TokenIn, q.AmountInWei, q.TokenOut, q.MinOutWei, q.Steps)
	if err != nil {
		return nil, fmt.Errorf("pack swap: %w", err)
	}

	txHash, err := c.sendTx(ctx, &AfiAddress, input, gasBuffer, nonceOverride)
	if err != nil {
		return nil, ErrSwapReverted(err.Error())
	}

	pending := &PendingSwap{
		TxHash: txHash,
		waitFn: func(ctx context.Context, opts WaitForTxOptions) (*SwapResult, error) {
			receipt, err := c.waitReceiptWithOpts(ctx, txHash, opts)
			if err != nil {
				return nil, err
			}
			if receipt.Status == types.ReceiptStatusFailed {
				return nil, ErrSwapReverted("transaction failed on-chain")
			}
			amountIn, amountOut, err := parseSwapExecuted(afiABI, receipt)
			if err != nil {
				return nil, err
			}
			eff, feeWei, feeEth := feeFromReceipt(receipt)
			return &SwapResult{
				TxHash:            receipt.TxHash,
				BlockNumber:       receipt.BlockNumber.Uint64(),
				AmountIn:          amountIn,
				AmountOut:         amountOut,
				TokenIn:           q.TokenIn,
				TokenOut:          q.TokenOut,
				GasUsed:           receipt.GasUsed,
				EffectiveGasPrice: eff,
				FeeWei:            feeWei,
				FeeETH:            feeEth,
			}, nil
		},
	}
	return pending, nil
}

// feeFromReceipt extracts the effective gas price and total fee from a tx receipt.
func feeFromReceipt(receipt *types.Receipt) (effectiveGasPrice, feeWei *big.Int, feeEth string) {
	if receipt.EffectiveGasPrice != nil {
		effectiveGasPrice = new(big.Int).Set(receipt.EffectiveGasPrice)
	} else {
		effectiveGasPrice = new(big.Int)
	}
	feeWei = new(big.Int).Mul(effectiveGasPrice, new(big.Int).SetUint64(receipt.GasUsed))
	feeEth = formatAmount(feeWei, 18)
	return
}

func parseSwapExecuted(afiABI abi.ABI, receipt *types.Receipt) (*big.Int, *big.Int, error) {
	event, ok := afiABI.Events["SwapExecuted"]
	if !ok {
		return nil, nil, fmt.Errorf("SwapExecuted not in ABI")
	}

	for _, log := range receipt.Logs {
		if !strings.EqualFold(log.Address.Hex(), AfiAddress.Hex()) {
			continue
		}
		if len(log.Topics) == 0 || log.Topics[0] != event.ID {
			continue
		}

		// Non-indexed fields (amountIn, amountOut) are in log.Data
		decoded := make(map[string]any)
		if err := afiABI.UnpackIntoMap(decoded, "SwapExecuted", log.Data); err != nil {
			return nil, nil, fmt.Errorf("unpack SwapExecuted: %w", err)
		}

		amountIn, _ := decoded["amountIn"].(*big.Int)
		amountOut, _ := decoded["amountOut"].(*big.Int)
		return amountIn, amountOut, nil
	}

	return nil, nil, fmt.Errorf("SwapExecuted event not found in receipt")
}

// afiABICached is parsed once at package init for use by standalone helpers like ParseSwapResult.
var afiABICached = mustParseABI(afiABIJSON)

func mustParseABI(s string) abi.ABI {
	a, err := abi.JSON(strings.NewReader(s))
	if err != nil {
		panic(fmt.Errorf("parse abi: %w", err))
	}
	return a
}

// ParseSwapResult inspects a transaction receipt for the AFI SwapExecuted event
// and reconstructs the SwapResult. Returns (nil, nil) when no SwapExecuted log
// is present (e.g. the tx wasn't an AFI swap).
//
// Use this for hashes obtained outside the SDK (indexers, replay tools, queued jobs).
func ParseSwapResult(receipt *types.Receipt) (*SwapResult, error) {
	event, ok := afiABICached.Events["SwapExecuted"]
	if !ok {
		return nil, fmt.Errorf("SwapExecuted not in ABI")
	}
	for _, lg := range receipt.Logs {
		if !strings.EqualFold(lg.Address.Hex(), AfiAddress.Hex()) {
			continue
		}
		if len(lg.Topics) < 4 || lg.Topics[0] != event.ID {
			continue
		}
		decoded := make(map[string]any)
		if err := afiABICached.UnpackIntoMap(decoded, "SwapExecuted", lg.Data); err != nil {
			return nil, fmt.Errorf("unpack SwapExecuted: %w", err)
		}
		amountIn, _ := decoded["amountIn"].(*big.Int)
		amountOut, _ := decoded["amountOut"].(*big.Int)
		eff, feeWei, feeEth := feeFromReceipt(receipt)
		return &SwapResult{
			TxHash:      receipt.TxHash,
			BlockNumber: receipt.BlockNumber.Uint64(),
			AmountIn:    amountIn,
			AmountOut:   amountOut,
			// Topics[1] = from (indexed), Topics[2] = assetIn, Topics[3] = assetOut
			TokenIn:           common.HexToAddress(lg.Topics[2].Hex()),
			TokenOut:          common.HexToAddress(lg.Topics[3].Hex()),
			GasUsed:           receipt.GasUsed,
			EffectiveGasPrice: eff,
			FeeWei:            feeWei,
			FeeETH:            feeEth,
		}, nil
	}
	return nil, nil
}

// applyGasBuffer returns gas * (1 + bufferPercent/100). bufferPercent of 0 returns gas unchanged.
func applyGasBuffer(gas uint64, bufferPercent uint) uint64 {
	if bufferPercent == 0 {
		return gas
	}
	return gas * (100 + uint64(bufferPercent)) / 100
}

// sendTx builds, signs, and sends a transaction. Returns the tx hash.
// gasBuffer is the percentage added on top of the estimated gas (0 = no buffer).
// nonceOverride takes priority over both the managed counter and the RPC query.
// Uses the RPC's chain ID (cached on the Client) so multi-chain execution works.
func (c *Client) sendTx(ctx context.Context, to *common.Address, data []byte, gasBuffer uint, nonceOverride *uint64) (string, error) {
	from := crypto.PubkeyToAddress(c.key.PublicKey)

	chainID, err := c.ChainID(ctx)
	if err != nil {
		return "", fmt.Errorf("chain id: %w", err)
	}

	nonce, err := c.allocateNonce(ctx, nonceOverride, from)
	if err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}

	gasLimit, err := c.eth.EstimateGas(ctx, ethereum.CallMsg{
		From: from,
		To:   to,
		Data: data,
	})
	if err != nil {
		// estimateGas reverts hide the actual reason — replay via eth_call to surface it.
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

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		To:        to,
		Gas:       gasLimit,
		GasTipCap: tip,
		GasFeeCap: maxFee,
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

// callRevertReason re-executes the call via eth_call to extract the revert message
// that estimateGas swallowed. Returns "" if we can't get a useful message.
// When the revert data is decodable as a known custom error, returns its formatted name.
func callRevertReason(ctx context.Context, client *ethclient.Client, from common.Address, to *common.Address, data []byte) string {
	_, err := client.CallContract(ctx, ethereum.CallMsg{From: from, To: to, Data: data}, nil)
	if err == nil {
		return ""
	}
	if rd := extractRevertData(err); rd != nil {
		if decoded := DecodeRevertReason(rd); decoded != nil {
			return decoded.String()
		}
	}
	return err.Error()
}

func (c *Client) waitReceipt(ctx context.Context, txHash string) (*types.Receipt, error) {
	return c.waitReceiptWithOpts(ctx, txHash, WaitForTxOptions{})
}

func (c *Client) waitReceiptWithOpts(ctx context.Context, txHash string, opts WaitForTxOptions) (*types.Receipt, error) {
	confs := opts.Confirmations
	if confs == 0 {
		confs = 1
	}
	poll := time.Duration(opts.PollIntervalMs) * time.Millisecond
	if poll <= 0 {
		poll = 500 * time.Millisecond
	}
	if opts.TimeoutMs > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(opts.TimeoutMs)*time.Millisecond)
		defer cancel()
	}

	hash := common.HexToHash(txHash)
	for {
		receipt, err := c.eth.TransactionReceipt(ctx, hash)
		if err == nil {
			if confs <= 1 {
				return receipt, nil
			}
			head, err := c.eth.HeaderByNumber(ctx, nil)
			if err != nil {
				return nil, fmt.Errorf("get head: %w", err)
			}
			got := head.Number.Uint64() - receipt.BlockNumber.Uint64() + 1
			if got >= confs {
				return receipt, nil
			}
		} else if !errors.Is(err, ethereum.NotFound) {
			return nil, fmt.Errorf("get receipt: %w", err)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(poll):
		}
	}
}

// privateKeyFromHex parses a private key from a hex string (with or without 0x).
func privateKeyFromHex(hexKey string) (*ecdsa.PrivateKey, error) {
	hexKey = strings.TrimPrefix(hexKey, "0x")
	return crypto.HexToECDSA(hexKey)
}
