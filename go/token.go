package afi

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func callERC20(ctx context.Context, client *ethclient.Client, erc20ABI abi.ABI, token common.Address, method string, args ...any) ([]any, error) {
	input, err := erc20ABI.Pack(method, args...)
	if err != nil {
		return nil, fmt.Errorf("pack %s: %w", method, err)
	}

	result, err := client.CallContract(ctx, ethereum.CallMsg{To: &token, Data: input}, nil)
	if err != nil {
		return nil, fmt.Errorf("call %s: %w", method, err)
	}

	return erc20ABI.Unpack(method, result)
}

func getDecimals(ctx context.Context, client *ethclient.Client, erc20ABI abi.ABI, token common.Address) (uint8, error) {
	out, err := callERC20(ctx, client, erc20ABI, token, "decimals")
	if err != nil {
		return 0, err
	}
	return out[0].(uint8), nil
}

func getBalance(ctx context.Context, client *ethclient.Client, erc20ABI abi.ABI, token, owner common.Address) (*big.Int, error) {
	out, err := callERC20(ctx, client, erc20ABI, token, "balanceOf", owner)
	if err != nil {
		return nil, err
	}
	return out[0].(*big.Int), nil
}

func getAllowance(ctx context.Context, client *ethclient.Client, erc20ABI abi.ABI, token, owner common.Address) (*big.Int, error) {
	out, err := callERC20(ctx, client, erc20ABI, token, "allowance", owner, AfiAddress)
	if err != nil {
		return nil, err
	}
	return out[0].(*big.Int), nil
}

func assertSufficientBalance(ctx context.Context, client *ethclient.Client, erc20ABI abi.ABI, token, owner common.Address, required *big.Int) error {
	balance, err := getBalance(ctx, client, erc20ABI, token, owner)
	if err != nil {
		return err
	}
	if balance.Cmp(required) < 0 {
		return ErrInsufficientBalance(token.Hex(), balance, required)
	}
	return nil
}

// ensureExactApproval approves exactly `amount` for the AFI contract.
// Returns the approval tx hash, or empty string if allowance was already sufficient.
func ensureExactApproval(
	ctx context.Context,
	client *ethclient.Client,
	erc20ABI abi.ABI,
	c *Client,
	token, owner common.Address,
	amount *big.Int,
) (string, error) {
	current, err := getAllowance(ctx, client, erc20ABI, token, owner)
	if err != nil {
		return "", err
	}

	if current.Cmp(amount) >= 0 {
		return "", nil
	}

	// Some tokens (USDT-style) reject non-zero → non-zero allowance changes.
	// Reset to 0 first if needed.
	if current.Sign() > 0 {
		if _, err := sendApprove(ctx, c, erc20ABI, token, big.NewInt(0)); err != nil {
			// Token doesn't require reset, continue
			_ = err
		}
	}

	hash, err := sendApprove(ctx, c, erc20ABI, token, amount)
	if err != nil {
		return "", ErrApproval(err.Error())
	}

	// Verify allowance on-chain before proceeding
	confirmed, err := getAllowance(ctx, client, erc20ABI, token, owner)
	if err != nil {
		return "", err
	}
	if confirmed.Cmp(amount) < 0 {
		return "", ErrApproval("allowance not reflected on-chain after confirmation")
	}

	return hash, nil
}

func sendApproveTx(ctx context.Context, c *Client, erc20ABI abi.ABI, token common.Address, amount *big.Int) (string, error) {
	input, err := erc20ABI.Pack("approve", AfiAddress, amount)
	if err != nil {
		return "", err
	}
	return c.sendTx(ctx, &token, input)
}

func sendApprove(ctx context.Context, c *Client, erc20ABI abi.ABI, token common.Address, amount *big.Int) (string, error) {
	hash, err := sendApproveTx(ctx, c, erc20ABI, token, amount)
	if err != nil {
		return "", err
	}
	_, err = c.waitReceipt(ctx, hash)
	if err != nil {
		return "", err
	}
	return strings.ToLower(hash), nil
}

func submitApprove(
	ctx context.Context,
	client *ethclient.Client,
	erc20ABI abi.ABI,
	c *Client,
	token, owner common.Address,
	amount *big.Int,
) (*PendingTx, error) {
	current, err := getAllowance(ctx, client, erc20ABI, token, owner)
	if err != nil {
		return nil, err
	}
	if current.Cmp(amount) >= 0 {
		return nil, nil
	}

	if current.Sign() > 0 {
		if _, err := sendApprove(ctx, c, erc20ABI, token, big.NewInt(0)); err != nil {
			_ = err // token doesn't require reset
		}
	}

	hash, err := sendApproveTx(ctx, c, erc20ABI, token, amount)
	if err != nil {
		return nil, ErrApproval(err.Error())
	}

	amountCopy := new(big.Int).Set(amount)
	pending := &PendingTx{
		TxHash: hash,
		waitFn: func(ctx context.Context) (*TxReceipt, error) {
			receipt, err := c.waitReceipt(ctx, hash)
			if err != nil {
				return nil, err
			}
			confirmed, err := getAllowance(ctx, client, erc20ABI, token, owner)
			if err != nil {
				return nil, err
			}
			if confirmed.Cmp(amountCopy) < 0 {
				return nil, ErrApproval("allowance not reflected on-chain after confirmation")
			}
			return &TxReceipt{BlockNumber: receipt.BlockNumber.Uint64(), GasUsed: receipt.GasUsed}, nil
		},
	}
	return pending, nil
}
