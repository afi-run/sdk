package afi

import "github.com/ethereum/go-ethereum/common"

var (
	// AfiAddress is the deprecated single-chain default.
	// Use AfiAddresses[network] for multi-chain deployments.
	// Filled in post-deployment — empty until deploy completes.
	AfiAddress = common.Address{}

	// WETH on Base (kept for legacy single-chain callers). New code should
	// look up the WETH address per network from constants.
	WETH = common.HexToAddress("0x4200000000000000000000000000000000000006")

	// Multicall3 is the same address on every EVM chain that has it deployed.
	Multicall3Address = common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")
)

// AfiAddresses maps Network → deployed Afi contract on that chain.
// Deployed 2026-05-30. Re-populate after redeploy.
var AfiAddresses = map[Network]common.Address{
	NetworkEthereum: common.HexToAddress("0xc578a4e89795803F396160610F4990c44abA8dAb"),
	NetworkBSC:      common.HexToAddress("0xFd4F8822f13D01aB142Bc985Ce587E35d7673C6e"),
	NetworkUnichain: common.HexToAddress("0xFd4F8822f13D01aB142Bc985Ce587E35d7673C6e"),
	NetworkBase:     common.HexToAddress("0xFd4F8822f13D01aB142Bc985Ce587E35d7673C6e"),
	NetworkArbitrum: common.HexToAddress("0xd74F60BD38243d089e286E3B6b9348f43a2314dF"),
}

// RouteQuoterAddresses maps Network → deployed RouteQuoter contract.
// Used by the simulation flow (simulate.go).
// Deployed 2026-05-30. Re-populate after redeploy.
var RouteQuoterAddresses = map[Network]common.Address{
	NetworkEthereum: common.HexToAddress("0x5e41b417E9742DB9c5402F8B1969a33891628Bed"),
	NetworkBSC:      common.HexToAddress("0xcA37E05a20E93fD88E5367F9d7d1422937c57A38"),
	NetworkUnichain: common.HexToAddress("0x2Cc852Cd57CC1b57CA09dbA7f69F0e225008cEBE"),
	NetworkBase:     common.HexToAddress("0xB5637138Cee6e757B679FFF8aDEA8DBa3E7544bB"),
	NetworkArbitrum: common.HexToAddress("0xBdD42B4fF06aCa8908D5E5d4826fFf5cdaC43895"),
}

// NMRAddresses maps Network → deployed NathanMayerRothschild (NMR) contract.
// Only deployed on Aave V3 chains.
// Deployed 2026-05-30. Re-populate after redeploy.
var NMRAddresses = map[Network]common.Address{
	NetworkEthereum: common.HexToAddress("0x29EfbFC1534A9B7af02142A5D97454E24Dc51b3a"),
	NetworkBase:     common.HexToAddress("0xefA12ba0196FD5ec44AF2ecAddc17333dF5FA779"),
	NetworkArbitrum: common.HexToAddress("0x6b533D53ec93eC30963b38576Ed8330Ff346a723"),
}

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

// AFIABIJSON, ERC20ABIJSON, Multicall3ABIJSON, NMRABIJSON are the raw ABI
// definitions used by the SDK. Exported so callers can do custom contract
// reads/writes with the same shapes.
var (
	AFIABIJSON        = afiABIJSON
	ERC20ABIJSON      = erc20ABIJSON
	Multicall3ABIJSON = multicall3ABIJSON
	NMRABIJSON        = nmrABIJSON
)

// AfiABI is the parsed/exposed JSON string for the Afi contract.
const AfiABI = afiABIJSON

// NMRABI is the parsed/exposed JSON string for the NathanMayerRothschild contract.
const NMRABI = nmrABIJSON

// RouteRegistryABI is a minimal ABI exposing the routing read methods used by
// the SDK (getRoute(uint16) -> address).
const RouteRegistryABI = `[
  {
    "type": "function",
    "name": "getRoute",
    "inputs": [{"name": "id", "type": "uint16"}],
    "outputs": [{"name": "", "type": "address"}],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "getRoutes",
    "inputs": [{"name": "ids", "type": "uint16[]"}],
    "outputs": [{"name": "", "type": "address[]"}],
    "stateMutability": "view"
  }
]`

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
    "name": "swapFor",
    "inputs": [
      {"name": "user",     "type": "address"},
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
    "name": "batchSwapFor",
    "inputs": [
      {
        "name": "swaps",
        "type": "tuple[]",
        "components": [
          {"name": "user",     "type": "address"},
          {"name": "tokenIn",  "type": "address"},
          {"name": "amountIn", "type": "uint256"},
          {"name": "tokenOut", "type": "address"},
          {"name": "minOut",   "type": "uint256"},
          {"name": "params",   "type": "bytes"}
        ]
      }
    ],
    "outputs": [],
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
    "type": "function",
    "name": "feeBpsOf",
    "inputs": [{"name": "user", "type": "address"}],
    "outputs": [{"name": "", "type": "uint16"}],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "paused",
    "inputs": [],
    "outputs": [{"name": "", "type": "bool"}],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "hasRules",
    "inputs": [],
    "outputs": [{"name": "", "type": "bool"}],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "treasury",
    "inputs": [],
    "outputs": [{"name": "", "type": "address"}],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "registry",
    "inputs": [],
    "outputs": [{"name": "", "type": "address"}],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "owner",
    "inputs": [],
    "outputs": [{"name": "", "type": "address"}],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "pendingOwner",
    "inputs": [],
    "outputs": [{"name": "", "type": "address"}],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "primaryOperator",
    "inputs": [],
    "outputs": [{"name": "", "type": "address"}],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "isOperator",
    "inputs": [{"name": "account", "type": "address"}],
    "outputs": [{"name": "", "type": "bool"}],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "pause",
    "inputs": [],
    "outputs": [],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "unpause",
    "inputs": [],
    "outputs": [],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "setTreasury",
    "inputs": [{"name": "_treasury", "type": "address"}],
    "outputs": [],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "setOperator",
    "inputs": [
      {"name": "_operator", "type": "address"},
      {"name": "_value",    "type": "bool"}
    ],
    "outputs": [],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "addRule",
    "inputs": [{"name": "_rule", "type": "address"}],
    "outputs": [],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "clearRules",
    "inputs": [],
    "outputs": [],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "setFeeBps",
    "inputs": [{"name": "_feeBps", "type": "uint16"}],
    "outputs": [],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "setUserFeeBps",
    "inputs": [
      {"name": "user",    "type": "address"},
      {"name": "_feeBps", "type": "uint16"}
    ],
    "outputs": [],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "clearUserFeeBps",
    "inputs": [{"name": "user", "type": "address"}],
    "outputs": [],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "setUserFeeBpsBatch",
    "inputs": [
      {"name": "users",   "type": "address[]"},
      {"name": "feesBps", "type": "uint16[]"}
    ],
    "outputs": [],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "resetAnyUserOverride",
    "inputs": [],
    "outputs": [],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "rescueTokens",
    "inputs": [
      {"name": "token", "type": "address"},
      {"name": "value", "type": "uint256"},
      {"name": "to",    "type": "address"}
    ],
    "outputs": [],
    "stateMutability": "nonpayable"
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
  {
    "type": "event",
    "name": "FeeCollected",
    "inputs": [
      {"name": "token",  "type": "address", "indexed": true},
      {"name": "amount", "type": "uint256", "indexed": false}
    ]
  },
  {
    "type": "event",
    "name": "TreasuryUpdated",
    "inputs": [
      {"name": "treasury", "type": "address", "indexed": true}
    ]
  },
  {
    "type": "event",
    "name": "FeeBpsUpdated",
    "inputs": [
      {"name": "feeBps", "type": "uint16", "indexed": false}
    ]
  },
  {
    "type": "event",
    "name": "UserFeeBpsSet",
    "inputs": [
      {"name": "user",   "type": "address", "indexed": true},
      {"name": "feeBps", "type": "uint16",  "indexed": false}
    ]
  },
  {
    "type": "event",
    "name": "UserFeeBpsCleared",
    "inputs": [
      {"name": "user", "type": "address", "indexed": true}
    ]
  },
  {
    "type": "event",
    "name": "AnyUserOverrideReset",
    "inputs": []
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

const nmrABIJSON = `[
  {
    "type": "function",
    "name": "requestOperation",
    "inputs": [
      {"name": "asset",  "type": "address"},
      {"name": "amount", "type": "uint256"},
      {"name": "params", "type": "bytes"}
    ],
    "outputs": [],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "swap",
    "inputs": [
      {"name": "asset",  "type": "address"},
      {"name": "amount", "type": "uint256"},
      {"name": "minOut", "type": "uint256"},
      {"name": "params", "type": "bytes"}
    ],
    "outputs": [{"name": "", "type": "uint256"}],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "loan",
    "inputs": [
      {"name": "user",   "type": "address"},
      {"name": "asset",  "type": "address"},
      {"name": "amount", "type": "uint256"},
      {"name": "minOut", "type": "uint256"},
      {"name": "params", "type": "bytes"}
    ],
    "outputs": [{"name": "", "type": "uint256"}],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "sweepProfit",
    "inputs": [
      {"name": "asset",  "type": "address"},
      {"name": "amount", "type": "uint256"}
    ],
    "outputs": [],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "setTreasury",
    "inputs": [{"name": "_treasury", "type": "address"}],
    "outputs": [],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "treasury",
    "inputs": [],
    "outputs": [{"name": "", "type": "address"}],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "PROFIT_SHARE",
    "inputs": [],
    "outputs": [{"name": "", "type": "uint8"}],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "isOperator",
    "inputs": [{"name": "account", "type": "address"}],
    "outputs": [{"name": "", "type": "bool"}],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "owner",
    "inputs": [],
    "outputs": [{"name": "", "type": "address"}],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "pendingOwner",
    "inputs": [],
    "outputs": [{"name": "", "type": "address"}],
    "stateMutability": "view"
  },
  {
    "type": "event",
    "name": "FlashLoanRequested",
    "inputs": [
      {"name": "asset",  "type": "address", "indexed": true},
      {"name": "amount", "type": "uint256", "indexed": false}
    ]
  },
  {
    "type": "event",
    "name": "FlashLoanExecuted",
    "inputs": [
      {"name": "asset",   "type": "address", "indexed": true},
      {"name": "amount",  "type": "uint256", "indexed": false},
      {"name": "premium", "type": "uint256", "indexed": false},
      {"name": "profit",  "type": "uint256", "indexed": false}
    ]
  },
  {
    "type": "event",
    "name": "FlashLoanFailed",
    "inputs": [
      {"name": "asset",  "type": "address", "indexed": true},
      {"name": "amount", "type": "uint256", "indexed": false},
      {"name": "reason", "type": "string",  "indexed": false}
    ]
  },
  {
    "type": "event",
    "name": "FlashLoanFailedWithData",
    "inputs": [
      {"name": "asset",  "type": "address", "indexed": true},
      {"name": "amount", "type": "uint256", "indexed": false},
      {"name": "data",   "type": "bytes",   "indexed": false}
    ]
  },
  {
    "type": "event",
    "name": "SwapExecuted",
    "inputs": [
      {"name": "assetIn",   "type": "address", "indexed": true},
      {"name": "amountIn",  "type": "uint256", "indexed": false},
      {"name": "assetOut",  "type": "address", "indexed": true},
      {"name": "amountOut", "type": "uint256", "indexed": false}
    ]
  },
  {
    "type": "event",
    "name": "ProfitSwept",
    "inputs": [
      {"name": "asset",  "type": "address", "indexed": true},
      {"name": "amount", "type": "uint256", "indexed": false},
      {"name": "to",     "type": "address", "indexed": true}
    ]
  },
  {
    "type": "event",
    "name": "TreasuryUpdated",
    "inputs": [
      {"name": "treasury", "type": "address", "indexed": true}
    ]
  },
  {
    "type": "event",
    "name": "ProfitShareUpdated",
    "inputs": [
      {"name": "profitShare", "type": "uint8", "indexed": false}
    ]
  }
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
