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
		return ErrSimulation(err.Error())
	}
	return nil
}

func submitSwap(ctx context.Context, c *Client, afiABI abi.ABI, q *Quote) (*PendingSwap, error) {
	input, err := afiABI.Pack("swap", q.TokenIn, q.AmountInWei, q.TokenOut, q.MinOutWei, q.Steps)
	if err != nil {
		return nil, fmt.Errorf("pack swap: %w", err)
	}

	txHash, err := c.sendTx(ctx, &AfiAddress, input)
	if err != nil {
		return nil, ErrSwapReverted(err.Error())
	}

	pending := &PendingSwap{
		TxHash: txHash,
		waitFn: func(ctx context.Context) (*SwapResult, error) {
			receipt, err := c.waitReceipt(ctx, txHash)
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
			return &SwapResult{
				TxHash:      receipt.TxHash,
				BlockNumber: receipt.BlockNumber.Uint64(),
				AmountIn:    amountIn,
				AmountOut:   amountOut,
				TokenIn:     q.TokenIn,
				TokenOut:    q.TokenOut,
				GasUsed:     receipt.GasUsed,
			}, nil
		},
	}
	return pending, nil
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

// sendTx builds, signs, and sends a transaction. Returns the tx hash.
func (c *Client) sendTx(ctx context.Context, to *common.Address, data []byte) (string, error) {
	from := crypto.PubkeyToAddress(c.key.PublicKey)

	nonce, err := c.eth.PendingNonceAt(ctx, from)
	if err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}

	gasLimit, err := c.eth.EstimateGas(ctx, ethereum.CallMsg{
		From: from,
		To:   to,
		Data: data,
	})
	if err != nil {
		return "", fmt.Errorf("estimate gas: %w", err)
	}
	gasLimit = gasLimit * 12 / 10 // 1.2x buffer

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
		ChainID:   big.NewInt(BaseChainID),
		Nonce:     nonce,
		To:        to,
		Gas:       gasLimit,
		GasTipCap: tip,
		GasFeeCap: maxFee,
		Data:      data,
	})

	signed, err := types.SignTx(tx, types.NewLondonSigner(big.NewInt(BaseChainID)), c.key)
	if err != nil {
		return "", fmt.Errorf("sign: %w", err)
	}

	if err := c.eth.SendTransaction(ctx, signed); err != nil {
		return "", fmt.Errorf("send: %w", err)
	}

	return signed.Hash().Hex(), nil
}

func (c *Client) waitReceipt(ctx context.Context, txHash string) (*types.Receipt, error) {
	hash := common.HexToHash(txHash)
	for {
		receipt, err := c.eth.TransactionReceipt(ctx, hash)
		if err == nil {
			return receipt, nil
		}
		if !errors.Is(err, ethereum.NotFound) {
			return nil, fmt.Errorf("get receipt: %w", err)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// privateKeyFromHex parses a private key from a hex string (with or without 0x).
func privateKeyFromHex(hexKey string) (*ecdsa.PrivateKey, error) {
	hexKey = strings.TrimPrefix(hexKey, "0x")
	return crypto.HexToECDSA(hexKey)
}
