package afi

import "github.com/ethereum/go-ethereum/common"

var (
	AfiAddress       = common.HexToAddress("0xB8cC65321d169D55b93b4402D795701c6B308ce4")
	WETH             = common.HexToAddress("0x4200000000000000000000000000000000000006")
	Multicall3Address = common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")
)

const (
	BaseChainID = int64(8453)
	APIBaseURL  = "https://rpc.afi.run"
	// DefaultGasBufferPercent is added on top of estimated gas for approve/swap txs when no override is set.
	DefaultGasBufferPercent = uint(15)
)

// NetworkExplorers maps each Network to its default block explorer base URL.
// Override entries at runtime or pass an explicit explorer to TxURL/AddressURL.
var NetworkExplorers = map[Network]string{
	NetworkBase:     "https://basescan.org",
	NetworkBSC:      "https://bscscan.com",
	NetworkArbitrum: "https://arbiscan.io",
	NetworkEthereum: "https://etherscan.io",
	NetworkUnichain: "https://uniscan.xyz",
}

// NetworkChainIDs maps each Network to its chain ID.
var NetworkChainIDs = map[Network]int64{
	NetworkBase:     8453,
	NetworkBSC:      56,
	NetworkArbitrum: 42161,
	NetworkEthereum: 1,
	NetworkUnichain: 130,
}

// AFIABIJSON, ERC20ABIJSON, Multicall3ABIJSON are the raw ABI definitions used by the SDK.
// Exported so callers can do custom contract reads/writes with the same shapes.
var (
	AFIABIJSON       = afiABIJSON
	ERC20ABIJSON     = erc20ABIJSON
	Multicall3ABIJSON = multicall3ABIJSON
)

const afiABIJSON = `[
  {
    "type": "function",
    "name": "swap",
    "inputs": [
      {"name": "tokenIn",  "type": "address"},
      {"name": "amountIn", "type": "uint256"},
      {"name": "tokenOut", "type": "address"},
      {"name": "minOut",   "type": "uint256"},
      {"name": "params",   "type": "bytes"}
    ],
    "outputs": [{"name": "out", "type": "uint256"}],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "feeBps",
    "inputs": [],
    "outputs": [{"name": "", "type": "uint16"}],
    "stateMutability": "view"
  },
  {
    "type": "event",
    "name": "SwapExecuted",
    "inputs": [
      {"name": "from",      "type": "address", "indexed": true},
      {"name": "assetIn",   "type": "address", "indexed": true},
      {"name": "amountIn",  "type": "uint256", "indexed": false},
      {"name": "assetOut",  "type": "address", "indexed": true},
      {"name": "amountOut", "type": "uint256", "indexed": false}
    ]
  },
  {"type": "error", "name": "DifferentAssets", "inputs": [
    {"name": "expected", "type": "address"},
    {"name": "actual",   "type": "address"}
  ]},
  {"type": "error", "name": "FeeTooHigh",       "inputs": [{"name": "feeBps",    "type": "uint16"}]},
  {"type": "error", "name": "InsufficientFunds","inputs": [{"name": "available", "type": "uint256"}]},
  {"type": "error", "name": "InvalidRouteID",   "inputs": [{"name": "routeID",   "type": "uint16"}]},
  {"type": "error", "name": "NotOperator",      "inputs": []},
  {"type": "error", "name": "OwnableInvalidOwner",        "inputs": [{"name": "owner",   "type": "address"}]},
  {"type": "error", "name": "OwnableUnauthorizedAccount", "inputs": [{"name": "account", "type": "address"}]},
  {"type": "error", "name": "ReentrancyGuardReentrantCall", "inputs": []},
  {"type": "error", "name": "ZeroAddress",      "inputs": []}
]`

const erc20ABIJSON = `[
  {
    "type": "function",
    "name": "decimals",
    "inputs": [],
    "outputs": [{"name": "", "type": "uint8"}],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "symbol",
    "inputs": [],
    "outputs": [{"name": "", "type": "string"}],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "name",
    "inputs": [],
    "outputs": [{"name": "", "type": "string"}],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "balanceOf",
    "inputs": [{"name": "account", "type": "address"}],
    "outputs": [{"name": "", "type": "uint256"}],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "allowance",
    "inputs": [
      {"name": "owner",   "type": "address"},
      {"name": "spender", "type": "address"}
    ],
    "outputs": [{"name": "", "type": "uint256"}],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "approve",
    "inputs": [
      {"name": "spender", "type": "address"},
      {"name": "value",   "type": "uint256"}
    ],
    "outputs": [{"name": "", "type": "bool"}],
    "stateMutability": "nonpayable"
  }
]`

const multicall3ABIJSON = `[
  {
    "type": "function",
    "name": "aggregate3",
    "inputs": [
      {
        "name": "calls",
        "type": "tuple[]",
        "components": [
          {"name": "target",       "type": "address"},
          {"name": "allowFailure", "type": "bool"},
          {"name": "callData",     "type": "bytes"}
        ]
      }
    ],
    "outputs": [
      {
        "name": "returnData",
        "type": "tuple[]",
        "components": [
          {"name": "success",    "type": "bool"},
          {"name": "returnData", "type": "bytes"}
        ]
      }
    ],
    "stateMutability": "payable"
  }
]`
