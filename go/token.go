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
	return getAllowanceFor(ctx, client, erc20ABI, token, owner, AfiAddress)
}

// getAllowanceFor reads ERC20(token).allowance(owner, spender). Generic version
// of getAllowance — required by workflow prechecks that target the NMR contract
// rather than the AFI router.
func getAllowanceFor(ctx context.Context, client *ethclient.Client, erc20ABI abi.ABI, token, owner, spender common.Address) (*big.Int, error) {
	out, err := callERC20(ctx, client, erc20ABI, token, "allowance", owner, spender)
	if err != nil {
		return nil, err
	}
	return out[0].(*big.Int), nil
}

func assertSufficientBalance(ctx context.Context, client *ethclient.Client, erc20ABI, mcABI abi.ABI, token, owner common.Address, required *big.Int) error {
	balance, err := getBalance(ctx, client, erc20ABI, token, owner)
	if err != nil {
		return err
	}
	if balance.Cmp(required) >= 0 {
		return nil
	}
	// Enrich the error with symbol/decimals via multicall — one extra RPC on the failure path
	// gives a vastly better message than raw addresses.
	symbol, decimals := "", uint8(0)
	if info, err := fetchTokenInfo(ctx, client, erc20ABI, mcABI, token, common.Address{}, nil); err == nil {
		symbol = info.Symbol
		decimals = info.Decimals
	}
	return ErrInsufficientBalanceDetailed(token.Hex(), owner.Hex(), symbol, decimals, balance, required)
}

// submitRevoke sends approve(AFI, 0) for `token`. Returns nil when allowance is already zero.
func submitRevoke(
	ctx context.Context,
	client *ethclient.Client,
	erc20ABI abi.ABI,
	c *Client,
	token, owner common.Address,
) (*PendingTx, error) {
	current, err := getAllowance(ctx, client, erc20ABI, token, owner)
	if err != nil {
		return nil, err
	}
	if current.Sign() == 0 {
		return nil, nil
	}

	hash, err := sendApproveTx(ctx, c, erc20ABI, token, big.NewInt(0))
	if err != nil {
		return nil, ErrApproval(err.Error())
	}

	pending := &PendingTx{
		TxHash: hash,
		waitFn: func(ctx context.Context, opts WaitForTxOptions) (*TxReceipt, error) {
			receipt, err := c.waitReceiptWithOpts(ctx, hash, opts)
			if err != nil {
				return nil, err
			}
			confirmed, err := getAllowance(ctx, client, erc20ABI, token, owner)
			if err != nil {
				return nil, err
			}
			if confirmed.Sign() != 0 {
				return nil, ErrApproval("allowance not zeroed after confirmation")
			}
			return receiptToTxReceipt(receipt), nil
		},
	}
	return pending, nil
}

// fetchTokenInfo runs symbol+name+decimals (+balance+allowance when owner != zero) in a single multicall.
// `cache` is consulted for metadata first and populated on misses.
func fetchTokenInfo(
	ctx context.Context,
	client *ethclient.Client,
	erc20ABI, mcABI abi.ABI,
	token, owner common.Address,
	cache *metadataCache,
) (*TokenInfo, error) {
	infos, err := fetchTokenInfoBatch(ctx, client, erc20ABI, mcABI, []common.Address{token}, owner, cache)
	if err != nil {
		return nil, err
	}
	return infos[0], nil
}

// fetchTokenInfoBatch is the N-token form of fetchTokenInfo — one multicall round-trip.
// Tokens whose metadata is already in `cache` only fetch balance/allowance (or nothing, when no owner).
func fetchTokenInfoBatch(
	ctx context.Context,
	client *ethclient.Client,
	erc20ABI, mcABI abi.ABI,
	tokens []common.Address,
	owner common.Address,
	cache *metadataCache,
) ([]*TokenInfo, error) {
	if len(tokens) == 0 {
		return []*TokenInfo{}, nil
	}
	withOwner := owner != (common.Address{})

	pack := func(method string, args ...any) []byte {
		data, err := erc20ABI.Pack(method, args...)
		if err != nil {
			panic(fmt.Errorf("pack %s: %w", method, err))
		}
		return data
	}

	type plan struct {
		token  common.Address
		cached *TokenMetadata
		offset int
		count  int
	}

	plans := make([]plan, 0, len(tokens))
	calls := make([]Multicall3Call, 0, len(tokens)*5)
	for _, t := range tokens {
		var cached *TokenMetadata
		if m, ok := cache.get(t); ok {
			cached = &m
		}
		p := plan{token: t, cached: cached, offset: len(calls)}
		if cached == nil {
			calls = append(calls,
				Multicall3Call{Target: t, AllowFailure: true, CallData: pack("symbol")},
				Multicall3Call{Target: t, AllowFailure: true, CallData: pack("name")},
				Multicall3Call{Target: t, AllowFailure: false, CallData: pack("decimals")},
			)
			p.count += 3
		}
		if withOwner {
			calls = append(calls,
				Multicall3Call{Target: t, AllowFailure: false, CallData: pack("balanceOf", owner)},
				Multicall3Call{Target: t, AllowFailure: false, CallData: pack("allowance", owner, AfiAddress)},
			)
			p.count += 2
		}
		plans = append(plans, p)
	}

	results, err := aggregate3(ctx, client, mcABI, calls)
	if err != nil {
		return nil, err
	}

	out := make([]*TokenInfo, len(tokens))
	for i, p := range plans {
		info := &TokenInfo{Address: p.token, Owner: owner}
		cursor := p.offset

		var meta TokenMetadata
		if p.cached != nil {
			meta = *p.cached
		} else {
			if results[cursor].Success {
				if dec, err := erc20ABI.Unpack("symbol", results[cursor].ReturnData); err == nil && len(dec) > 0 {
					if s, ok := dec[0].(string); ok {
						meta.Symbol = s
					}
				}
			}
			cursor++
			if results[cursor].Success {
				if dec, err := erc20ABI.Unpack("name", results[cursor].ReturnData); err == nil && len(dec) > 0 {
					if s, ok := dec[0].(string); ok {
						meta.Name = s
					}
				}
			}
			cursor++
			dec, err := erc20ABI.Unpack("decimals", results[cursor].ReturnData)
			if err != nil {
				return nil, fmt.Errorf("unpack decimals for %s: %w", p.token.Hex(), err)
			}
			meta.Decimals = dec[0].(uint8)
			cursor++
			cache.set(p.token, meta)
		}

		info.Symbol = meta.Symbol
		info.Name = meta.Name
		info.Decimals = meta.Decimals

		if withOwner {
			bal, err := erc20ABI.Unpack("balanceOf", results[cursor].ReturnData)
			if err != nil {
				return nil, fmt.Errorf("unpack balanceOf for %s: %w", p.token.Hex(), err)
			}
			info.Balance = bal[0].(*big.Int)
			cursor++
			al, err := erc20ABI.Unpack("allowance", results[cursor].ReturnData)
			if err != nil {
				return nil, fmt.Errorf("unpack allowance for %s: %w", p.token.Hex(), err)
			}
			info.Allowance = al[0].(*big.Int)
		}

		out[i] = info
	}

	return out, nil
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
	// Reset to 0 first if needed. We preserve the reset error so it can be surfaced
	// when the subsequent approve also fails — otherwise the dev sees a confusing
	// "allowance not reflected" without knowing the reset was the real culprit.
	var resetErr error
	if current.Sign() > 0 {
		if _, err := sendApprove(ctx, c, erc20ABI, token, big.NewInt(0)); err != nil {
			resetErr = err
		}
	}

	hash, err := sendApprove(ctx, c, erc20ABI, token, amount)
	if err != nil {
		if resetErr != nil {
			return "", ErrApproval(fmt.Sprintf("%s (allowance reset also failed: %s)", err.Error(), resetErr))
		}
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
	return sendApproveTxWithNonce(ctx, c, erc20ABI, token, amount, nil)
}

func sendApproveTxWithNonce(ctx context.Context, c *Client, erc20ABI abi.ABI, token common.Address, amount *big.Int, nonce *uint64) (string, error) {
	input, err := erc20ABI.Pack("approve", AfiAddress, amount)
	if err != nil {
		return "", err
	}
	return c.sendTx(ctx, &token, input, c.gasBufferPct, nonce)
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

	var resetErr error
	if current.Sign() > 0 {
		if _, err := sendApprove(ctx, c, erc20ABI, token, big.NewInt(0)); err != nil {
			resetErr = err
		}
	}

	hash, err := sendApproveTx(ctx, c, erc20ABI, token, amount)
	if err != nil {
		if resetErr != nil {
			return nil, ErrApproval(fmt.Sprintf("%s (allowance reset also failed: %s)", err.Error(), resetErr))
		}
		return nil, ErrApproval(err.Error())
	}

	amountCopy := new(big.Int).Set(amount)
	pending := &PendingTx{
		TxHash: hash,
		waitFn: func(ctx context.Context, opts WaitForTxOptions) (*TxReceipt, error) {
			receipt, err := c.waitReceiptWithOpts(ctx, hash, opts)
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
			return receiptToTxReceipt(receipt), nil
		},
	}
	return pending, nil
}
