package afi

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// EncodedTx is a pre-encoded transaction payload that can be passed to any
// signer that accepts (to, data, value) — useful when the private key is
// held externally (Frame, Ledger, hardware wallets, Safe SDK, etc.).
type EncodedTx struct {
	To   common.Address
	Data []byte
	// Value is always 0 for AFI swaps and ERC-20 approvals.
	Value *big.Int
}

// EncodeSwap builds the AFI router calldata for a quote. Sign and submit with
// any signer — useful when the SDK can't hold the private key directly.
func EncodeSwap(q *Quote) (*EncodedTx, error) {
	a, err := afiABICached.Pack("swap", q.TokenIn, q.AmountInWei, q.TokenOut, q.MinOutWei, q.Steps)
	if err != nil {
		return nil, fmt.Errorf("pack swap: %w", err)
	}
	return &EncodedTx{To: AfiAddress, Data: a, Value: new(big.Int)}, nil
}

// EncodeApprove builds ERC-20 `approve(AFI, amountWei)` calldata.
func EncodeApprove(token common.Address, amountWei *big.Int) (*EncodedTx, error) {
	a, err := erc20ABICached.Pack("approve", AfiAddress, amountWei)
	if err != nil {
		return nil, fmt.Errorf("pack approve: %w", err)
	}
	return &EncodedTx{To: token, Data: a, Value: new(big.Int)}, nil
}

// EncodeRevoke builds the calldata that zeroes the AFI router's allowance on `token`.
func EncodeRevoke(token common.Address) (*EncodedTx, error) {
	return EncodeApprove(token, new(big.Int))
}

// erc20ABICached is parsed once at package init for use by the standalone helpers.
var erc20ABICached = mustParseABI(erc20ABIJSON)
