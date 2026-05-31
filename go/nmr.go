package afi

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// nmrParsedABI is the lazily-parsed NMR contract ABI — it relies on the
// NMRABI constant defined in constants.go.
var nmrParsedABI = func() abi.ABI {
	a, err := abi.JSON(strings.NewReader(NMRABI))
	if err != nil {
		panic("invalid NMRABI: " + err.Error())
	}
	return a
}()

// EncodeNMRRequestOperation builds calldata for NMR.requestOperation(asset, amount, params).
func EncodeNMRRequestOperation(asset common.Address, amount *big.Int, params []byte) ([]byte, error) {
	return nmrParsedABI.Pack("requestOperation", asset, amount, params)
}

// EncodeNMRSwap builds calldata for NMR.swap (arbitrage cycle, input==output).
func EncodeNMRSwap(asset common.Address, amount, minOut *big.Int, params []byte) ([]byte, error) {
	return nmrParsedABI.Pack("swap", asset, amount, minOut, params)
}

// EncodeNMRLoan builds calldata for NMR.loan (user-funded arbitrage).
func EncodeNMRLoan(user, asset common.Address, amount, minOut *big.Int, params []byte) ([]byte, error) {
	return nmrParsedABI.Pack("loan", user, asset, amount, minOut, params)
}

// EncodeNMRSweepProfit builds calldata for NMR.sweepProfit(asset, amount).
// Destination is fixed to the contract's treasury — operator-callable.
func EncodeNMRSweepProfit(asset common.Address, amount *big.Int) ([]byte, error) {
	return nmrParsedABI.Pack("sweepProfit", asset, amount)
}

// EncodeNMRSetTreasury builds calldata for NMR.setTreasury (owner-only).
func EncodeNMRSetTreasury(treasury common.Address) ([]byte, error) {
	return nmrParsedABI.Pack("setTreasury", treasury)
}
