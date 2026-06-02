package afi

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

// SimulationResult captures the output of a RouteQuoter eth_call.
type SimulationResult struct {
	// OutputAsset is the asset at the end of the route chain. For arbitrage
	// cycles this should equal the input asset.
	OutputAsset common.Address

	// AmountOut is the actual amount produced after executing the full chain.
	AmountOut *big.Int

	// Reverted is true if any route along the chain reverted.
	Reverted bool

	// RevertData carries the raw revert payload when Reverted is true. Useful
	// for surfacing custom error reasons (InvalidRouteID, InsufficientFunds, etc).
	RevertData []byte
}

// SimulateOpts groups the inputs for SimulateRoute.
type SimulateOpts struct {
	// RPCClient is the JSON-RPC client. Must support state_override (Geth,
	// Erigon, Anvil, Alchemy, Infura all support it).
	RPCClient *rpc.Client

	// ChainID is used to look up the token balance storage slot.
	ChainID int64

	// QuoterAddr is the deployed RouteQuoter contract for this chain.
	QuoterAddr common.Address

	// Asset is the input token entering the route chain.
	Asset common.Address

	// Amount is how much of Asset enters the chain (raw units).
	Amount *big.Int

	// StepsEncoded is the tight-format step list, as produced by EncodeSteps.
	StepsEncoded []byte

	// Block is the block number to simulate against. nil = "latest".
	Block *big.Int
}

const routeQuoterABIJSON = `[
  {
    "type": "function",
    "name": "quote",
    "inputs": [
      {"name": "asset",  "type": "address"},
      {"name": "amount", "type": "uint256"},
      {"name": "params", "type": "bytes"}
    ],
    "outputs": [
      {"name": "outputAsset", "type": "address"},
      {"name": "amountOut",   "type": "uint256"}
    ],
    "stateMutability": "nonpayable"
  }
]`

// RouteQuoterABIJSON is exported so callers can do custom encoding/decoding.
var RouteQuoterABIJSON = routeQuoterABIJSON

var routeQuoterABI = mustParseABI(routeQuoterABIJSON)

// autoDetectMaxSlot bounds the brute-force balance-slot search triggered when a
// token is missing from the static table (the highest curated slot is 51).
const autoDetectMaxSlot = 128

// SimulateRoute runs `RouteQuoter.quote(asset, amount, params)` via eth_call
// with a state_override that grants the quoter contract a virtual balance of
// `amount` of `asset`.
//
// On success returns the real outputAsset and amountOut. On any internal route
// revert, returns Reverted=true and the raw revert payload for decoding.
//
// If `asset`'s balance slot is not in the static table, it is auto-detected
// on-chain via DetectBalanceSlot and cached (one extra round of eth_calls the
// first time a new token is seen; subsequent calls hit the cache). This keeps
// simulation working for any token the backend lists without hand-maintaining
// the slot table.
func SimulateRoute(ctx context.Context, opts SimulateOpts) (*SimulationResult, error) {
	if opts.RPCClient == nil {
		return nil, fmt.Errorf("afi.SimulateRoute: RPCClient is required")
	}
	if opts.Amount == nil || opts.Amount.Sign() <= 0 {
		return nil, fmt.Errorf("afi.SimulateRoute: Amount must be > 0")
	}

	callData, err := routeQuoterABI.Pack("quote", opts.Asset, opts.Amount, opts.StepsEncoded)
	if err != nil {
		return nil, fmt.Errorf("afi.SimulateRoute: pack quote: %w", err)
	}

	// Self-heal: an unknown token's slot is detected on-chain and registered,
	// so BuildBalanceOverride below resolves it.
	if _, ok := LookupBalanceSlot(opts.ChainID, opts.Asset); !ok {
		if _, derr := DetectBalanceSlot(ctx, opts.RPCClient, opts.ChainID, opts.Asset, autoDetectMaxSlot); derr != nil {
			return nil, fmt.Errorf("afi.SimulateRoute: detect balance slot: %w", derr)
		}
	}

	override, err := BuildBalanceOverride(opts.ChainID, opts.Asset, opts.QuoterAddr, opts.Amount)
	if err != nil {
		return nil, fmt.Errorf("afi.SimulateRoute: build state override: %w", err)
	}

	blockTag := "latest"
	if opts.Block != nil {
		blockTag = "0x" + opts.Block.Text(16)
	}

	var raw string
	rpcErr := opts.RPCClient.CallContext(ctx, &raw,
		"eth_call",
		map[string]interface{}{
			"to":   opts.QuoterAddr.Hex(),
			"data": "0x" + hex.EncodeToString(callData),
		},
		blockTag,
		override,
	)

	if rpcErr != nil {
		// Revert path — extract the raw error payload for caller-side decoding.
		return &SimulationResult{
			Reverted:   true,
			RevertData: extractRevertData(rpcErr),
		}, nil
	}

	// Success — decode (address, uint256)
	outBytes := common.FromHex(raw)
	out, err := routeQuoterABI.Unpack("quote", outBytes)
	if err != nil {
		return nil, fmt.Errorf("afi.SimulateRoute: decode result: %w", err)
	}
	return &SimulationResult{
		OutputAsset: out[0].(common.Address),
		AmountOut:   out[1].(*big.Int),
		Reverted:    false,
	}, nil
}
