package afi

import "github.com/ethereum/go-ethereum/common"

var (
	AfiAddress = common.HexToAddress("0xB8cC65321d169D55b93b4402D795701c6B308ce4")
	WETH       = common.HexToAddress("0x4200000000000000000000000000000000000006")
)

const (
	BaseChainID = int64(8453)
	APIBaseURL  = "https://rpc.afi.run"
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
