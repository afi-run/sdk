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

// ReferralRouterAddresses maps Network → deployed AfiReferralRouter contract.
// The router is a thin wrapper in front of Afi that charges an optional
// referral fee (<= 0.10%) on the output token. See go/referral.go for helpers.
// Deployed 2026-06-03.
var ReferralRouterAddresses = map[Network]common.Address{
	NetworkEthereum: common.HexToAddress("0x47E7cE4237130F02202e081Efa1Fd338F23Ead77"),
	NetworkBSC:      common.HexToAddress("0x7356960324a627994bb5959CF615DC5f2B38B738"),
	NetworkUnichain: common.HexToAddress("0xcdC506dEA82FE7d034C0281564d0dbe49171D242"),
	NetworkBase:     common.HexToAddress("0x2dC7a3990618baa91c450521004F14A334BF47c6"),
	NetworkArbitrum: common.HexToAddress("0x9DaD9322e196F734Fa25eC3b0db90387945B397C"),
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

// AFIABIJSON, ERC20ABIJSON, Multicall3ABIJSON are the raw ABI
// definitions used by the SDK. Exported so callers can do custom contract
// reads/writes with the same shapes.
var (
	AFIABIJSON        = afiABIJSON
	ERC20ABIJSON      = erc20ABIJSON
	Multicall3ABIJSON = multicall3ABIJSON
)

// AfiABI is the parsed/exposed JSON string for the Afi contract.
const AfiABI = afiABIJSON

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

// AfiReferralRouterABI is the full ABI of the AfiReferralRouter contract.
// See ReferralRouterAddresses for deployments and referral.go for helpers.
const AfiReferralRouterABI = `[
  {
    "type": "constructor",
    "inputs": [
      {
        "name": "_initialOwner",
        "type": "address"
      },
      {
        "name": "_afi",
        "type": "address"
      }
    ],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "HARD_CAP_BPS",
    "inputs": [],
    "outputs": [
      {
        "name": "",
        "type": "uint16"
      }
    ],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "acceptOwnership",
    "inputs": [],
    "outputs": [],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "afi",
    "inputs": [],
    "outputs": [
      {
        "name": "",
        "type": "address"
      }
    ],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "claim",
    "inputs": [
      {
        "name": "token",
        "type": "address"
      },
      {
        "name": "to",
        "type": "address"
      }
    ],
    "outputs": [],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "claimMany",
    "inputs": [
      {
        "name": "tokens",
        "type": "address[]"
      },
      {
        "name": "to",
        "type": "address"
      }
    ],
    "outputs": [],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "credits",
    "inputs": [
      {
        "name": "referrer",
        "type": "address"
      },
      {
        "name": "token",
        "type": "address"
      }
    ],
    "outputs": [
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "delegateAllowance",
    "inputs": [
      {
        "name": "owner",
        "type": "address"
      },
      {
        "name": "token",
        "type": "address"
      },
      {
        "name": "delegate",
        "type": "address"
      }
    ],
    "outputs": [
      {
        "name": "amount",
        "type": "uint208"
      },
      {
        "name": "deadline",
        "type": "uint48"
      }
    ],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "maxReferralBps",
    "inputs": [],
    "outputs": [
      {
        "name": "",
        "type": "uint16"
      }
    ],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "owner",
    "inputs": [],
    "outputs": [
      {
        "name": "",
        "type": "address"
      }
    ],
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
    "name": "paused",
    "inputs": [],
    "outputs": [
      {
        "name": "",
        "type": "bool"
      }
    ],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "pendingOwner",
    "inputs": [],
    "outputs": [
      {
        "name": "",
        "type": "address"
      }
    ],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "renounceOwnership",
    "inputs": [],
    "outputs": [],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "rescueTokens",
    "inputs": [
      {
        "name": "token",
        "type": "address"
      },
      {
        "name": "to",
        "type": "address"
      }
    ],
    "outputs": [],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "revokeDelegate",
    "inputs": [
      {
        "name": "token",
        "type": "address"
      },
      {
        "name": "delegate",
        "type": "address"
      }
    ],
    "outputs": [],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "setDelegateAllowance",
    "inputs": [
      {
        "name": "token",
        "type": "address"
      },
      {
        "name": "delegate",
        "type": "address"
      },
      {
        "name": "amount",
        "type": "uint208"
      },
      {
        "name": "deadline",
        "type": "uint48"
      }
    ],
    "outputs": [],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "setMaxReferralBps",
    "inputs": [
      {
        "name": "bps",
        "type": "uint16"
      }
    ],
    "outputs": [],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "swapWithReferral",
    "inputs": [
      {
        "name": "tokenIn",
        "type": "address"
      },
      {
        "name": "amountIn",
        "type": "uint256"
      },
      {
        "name": "tokenOut",
        "type": "address"
      },
      {
        "name": "minOut",
        "type": "uint256"
      },
      {
        "name": "params",
        "type": "bytes"
      },
      {
        "name": "referrer",
        "type": "address"
      },
      {
        "name": "referralBps",
        "type": "uint16"
      }
    ],
    "outputs": [
      {
        "name": "userAmount",
        "type": "uint256"
      }
    ],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "swapWithReferralFor",
    "inputs": [
      {
        "name": "user",
        "type": "address"
      },
      {
        "name": "tokenIn",
        "type": "address"
      },
      {
        "name": "amountIn",
        "type": "uint256"
      },
      {
        "name": "tokenOut",
        "type": "address"
      },
      {
        "name": "minOut",
        "type": "uint256"
      },
      {
        "name": "params",
        "type": "bytes"
      },
      {
        "name": "referrer",
        "type": "address"
      },
      {
        "name": "referralBps",
        "type": "uint16"
      }
    ],
    "outputs": [
      {
        "name": "userAmount",
        "type": "uint256"
      }
    ],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "totalCredits",
    "inputs": [
      {
        "name": "token",
        "type": "address"
      }
    ],
    "outputs": [
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "transferOwnership",
    "inputs": [
      {
        "name": "newOwner",
        "type": "address"
      }
    ],
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
    "type": "event",
    "name": "DelegateAllowanceSet",
    "inputs": [
      {
        "name": "owner",
        "type": "address",
        "indexed": true
      },
      {
        "name": "token",
        "type": "address",
        "indexed": true
      },
      {
        "name": "delegate",
        "type": "address",
        "indexed": true
      },
      {
        "name": "amount",
        "type": "uint208",
        "indexed": false
      },
      {
        "name": "deadline",
        "type": "uint48",
        "indexed": false
      }
    ],
    "anonymous": false
  },
  {
    "type": "event",
    "name": "DelegateRevoked",
    "inputs": [
      {
        "name": "owner",
        "type": "address",
        "indexed": true
      },
      {
        "name": "token",
        "type": "address",
        "indexed": true
      },
      {
        "name": "delegate",
        "type": "address",
        "indexed": true
      }
    ],
    "anonymous": false
  },
  {
    "type": "event",
    "name": "MaxReferralBpsUpdated",
    "inputs": [
      {
        "name": "bps",
        "type": "uint16",
        "indexed": false
      }
    ],
    "anonymous": false
  },
  {
    "type": "event",
    "name": "OwnershipTransferStarted",
    "inputs": [
      {
        "name": "previousOwner",
        "type": "address",
        "indexed": true
      },
      {
        "name": "newOwner",
        "type": "address",
        "indexed": true
      }
    ],
    "anonymous": false
  },
  {
    "type": "event",
    "name": "OwnershipTransferred",
    "inputs": [
      {
        "name": "previousOwner",
        "type": "address",
        "indexed": true
      },
      {
        "name": "newOwner",
        "type": "address",
        "indexed": true
      }
    ],
    "anonymous": false
  },
  {
    "type": "event",
    "name": "Paused",
    "inputs": [
      {
        "name": "account",
        "type": "address",
        "indexed": false
      }
    ],
    "anonymous": false
  },
  {
    "type": "event",
    "name": "ReferralAccrued",
    "inputs": [
      {
        "name": "referrer",
        "type": "address",
        "indexed": true
      },
      {
        "name": "token",
        "type": "address",
        "indexed": true
      },
      {
        "name": "amount",
        "type": "uint256",
        "indexed": false
      }
    ],
    "anonymous": false
  },
  {
    "type": "event",
    "name": "ReferralClaimed",
    "inputs": [
      {
        "name": "referrer",
        "type": "address",
        "indexed": true
      },
      {
        "name": "token",
        "type": "address",
        "indexed": true
      },
      {
        "name": "amount",
        "type": "uint256",
        "indexed": false
      },
      {
        "name": "to",
        "type": "address",
        "indexed": true
      }
    ],
    "anonymous": false
  },
  {
    "type": "event",
    "name": "ReferralSwap",
    "inputs": [
      {
        "name": "user",
        "type": "address",
        "indexed": true
      },
      {
        "name": "tokenIn",
        "type": "address",
        "indexed": true
      },
      {
        "name": "amountIn",
        "type": "uint256",
        "indexed": false
      },
      {
        "name": "tokenOut",
        "type": "address",
        "indexed": true
      },
      {
        "name": "userAmount",
        "type": "uint256",
        "indexed": false
      },
      {
        "name": "referrer",
        "type": "address",
        "indexed": false
      },
      {
        "name": "fee",
        "type": "uint256",
        "indexed": false
      },
      {
        "name": "executor",
        "type": "address",
        "indexed": false
      }
    ],
    "anonymous": false
  },
  {
    "type": "event",
    "name": "Rescued",
    "inputs": [
      {
        "name": "token",
        "type": "address",
        "indexed": true
      },
      {
        "name": "amount",
        "type": "uint256",
        "indexed": false
      },
      {
        "name": "to",
        "type": "address",
        "indexed": true
      }
    ],
    "anonymous": false
  },
  {
    "type": "event",
    "name": "Unpaused",
    "inputs": [
      {
        "name": "account",
        "type": "address",
        "indexed": false
      }
    ],
    "anonymous": false
  },
  {
    "type": "error",
    "name": "DelegationExpired",
    "inputs": []
  },
  {
    "type": "error",
    "name": "EnforcedPause",
    "inputs": []
  },
  {
    "type": "error",
    "name": "ExpectedPause",
    "inputs": []
  },
  {
    "type": "error",
    "name": "FeeTooHigh",
    "inputs": [
      {
        "name": "bps",
        "type": "uint16"
      }
    ]
  },
  {
    "type": "error",
    "name": "InsufficientDelegateAllowance",
    "inputs": [
      {
        "name": "requested",
        "type": "uint256"
      },
      {
        "name": "available",
        "type": "uint256"
      }
    ]
  },
  {
    "type": "error",
    "name": "InsufficientOutput",
    "inputs": [
      {
        "name": "received",
        "type": "uint256"
      }
    ]
  },
  {
    "type": "error",
    "name": "NothingToClaim",
    "inputs": []
  },
  {
    "type": "error",
    "name": "NothingToRescue",
    "inputs": []
  },
  {
    "type": "error",
    "name": "OwnableInvalidOwner",
    "inputs": [
      {
        "name": "owner",
        "type": "address"
      }
    ]
  },
  {
    "type": "error",
    "name": "OwnableUnauthorizedAccount",
    "inputs": [
      {
        "name": "account",
        "type": "address"
      }
    ]
  },
  {
    "type": "error",
    "name": "Reentrancy",
    "inputs": []
  },
  {
    "type": "error",
    "name": "SafeERC20FailedOperation",
    "inputs": [
      {
        "name": "token",
        "type": "address"
      }
    ]
  },
  {
    "type": "error",
    "name": "ZeroAddress",
    "inputs": []
  },
  {
    "type": "error",
    "name": "ZeroAmount",
    "inputs": []
  }
]
`
