package afi

import "github.com/ethereum/go-ethereum/common"

// BalanceSlot is the storage slot index at which a given ERC-20 token stores
// its `balances` mapping. For state_override-based simulations the actual slot
// for an address is keccak256(addr . slot).
//
// Slots vary by token:
//   - Most OZ-based ERC20s: slot 0 (after _name, _symbol — those are slot 5+ in some versions)
//   - WETH (Base, Mainnet): slot 3
//   - USDC proxies use slot 9 in their implementation
//
// This table is an optional fast-path: SimulateRoute auto-detects and caches
// the slot for any token not listed here via DetectBalanceSlot. Adding a popular
// token avoids that one-time on-chain probe, but the table never needs to be
// kept in sync by hand for correctness.
var balanceSlots = map[int64]map[common.Address]uint64{
	8453: { // Base
		common.HexToAddress("0x0b3e328455c4059EEb9e3f84b5543F74E24e7E1b"): 5,  // VIRTUAL
		common.HexToAddress("0x236aa50979D5f3De3Bd1Eeb40E81137F22ab794b"): 51, // tBTC
		common.HexToAddress("0x417Ac0e078398C154EdFadD9Ef675d30Be60Af93"): 0,  // crvUSD
		common.HexToAddress("0x5d3a1Ff2b6BAb83b63cd9AD0787074081a52ef34"): 5,  // USDe
		common.HexToAddress("0x820C137fa70C8691f0e44Dc420a5e53c168921Dc"): 2,  // USDS
		common.HexToAddress("0xAb6363dA0C80cEF3Ae105Bd6241E30872355d021"): 0,  // ROLL
		common.HexToAddress("0x4200000000000000000000000000000000000006"): 3,  // WETH
		common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"): 9,  // USDC
		common.HexToAddress("0xd9aAEc86B65D86f6A7B5B1b0c42FFA531710b6CA"): 9,  // USDbC
		common.HexToAddress("0x50c5725949A6F0c72E6C4a641F24049A917DB0Cb"): 0,  // DAI
		common.HexToAddress("0xfde4C96c8593536E31F229EA8f37b2ADa2699bb2"): 0,  // USDT
		common.HexToAddress("0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf"): 0,  // cbBTC
		common.HexToAddress("0x2Ae3F1Ec7F1F5012CFEab0185bfc7aa3cf0DEc22"): 0,  // cbETH
		common.HexToAddress("0x940181a94A35A4569E4529A3CDfB74e38FD98631"): 0,  // AERO
		common.HexToAddress("0x1cea84203673764244e05693e42e6Ace62bE9bA5"): 0,  // WBTC
		common.HexToAddress("0x63706e401c06ac8513145b7687A14804d17f814b"): 0,  // AAVE
		common.HexToAddress("0x04C0599Ae5A44757c0af6F9eC3b93da8976c150A"): 0,  // weETH
		common.HexToAddress("0x60a3E35Cc302bFA44Cb288Bc5a4F316Fdb1adb42"): 0,  // EURC
	},
	42161: { // Arbitrum
		common.HexToAddress("0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf"): 9,  // cbBTC
		common.HexToAddress("0xFa7F8980b0f1E64A2062791cc3b0871572f1F7f0"): 51, // UNI
		common.HexToAddress("0xf97f4df75117a78c1A5a0DBb814Af92458539FB4"): 51, // LINK
		common.HexToAddress("0x5d3a1Ff2b6BAb83b63cd9AD0787074081a52ef34"): 5,  // USDe
		common.HexToAddress("0x6491c05A82219b8D1479057361ff1654749b876b"): 2,  // USDS
		common.HexToAddress("0x82aF49447D8a07e3bd95BD0d56f35241523fBab1"): 51, // WETH
		common.HexToAddress("0xaf88d065e77c8cC2239327C5EDb3A432268e5831"): 9,  // USDC (native)
		common.HexToAddress("0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8"): 51, // USDC.e
		common.HexToAddress("0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9"): 51, // USDT
		common.HexToAddress("0xDA10009cBd5D07dd0CeCc66161FC93D7c9000da1"): 2,  // DAI
		common.HexToAddress("0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f"): 0,  // WBTC
		common.HexToAddress("0x912CE59144191C1204E64559FE8253a0e49E6548"): 51, // ARB
		common.HexToAddress("0xfc5A1A6EB076a2C7aD06eD22C90d7E710E35ad0a"): 0,  // GMX
		common.HexToAddress("0x539bdE0d7Dbd336b79148AA742883198BBF60342"): 0,  // MAGIC
		common.HexToAddress("0x0c880f6761F1af8d9Aa9C466984b80DAb9a8c9e8"): 0,  // PENDLE
		common.HexToAddress("0x35751007a407ca6FEFfE80b3cB397736D2cf4dbe"): 0,  // weETH
	},
	1: { // Mainnet
		common.HexToAddress("0x4c9EDD5852cd905f086C759E8383e09bff1E68B3"): 2,  // USDe
		common.HexToAddress("0xB8c77482e45F1F44dE1745F52C74426C631bDD52"): 5,  // BNB
		common.HexToAddress("0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf"): 9,  // cbBTC
		common.HexToAddress("0x514910771AF9Ca656af840dff83E8264EcF986CA"): 1,  // LINK
		common.HexToAddress("0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984"): 4,  // UNI
		common.HexToAddress("0x7Fc66500c84A76Ad7e9c93437bFc5Ac33E2DDaE9"): 0,  // AAVE
		common.HexToAddress("0x9f8F72aA9304c8B593d555F12eF6589cC3A579A2"): 1,  // MKR
		common.HexToAddress("0x57e114B691Db790C35207b2e685D4A43181e6061"): 2,  // ENA
		common.HexToAddress("0x68749665FF8D2d112Fa859AA293F07A622782F38"): 51, // XAUT
		common.HexToAddress("0xa3931d71877C0E7a3148CB7Eb4463524FEc27fbD"): 2,  // sUSDS
		common.HexToAddress("0xdC035D45d973E3EC169d2276DDab16f1e407384F"): 2,  // USDS
		common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"): 3,  // WETH
		common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"): 9,  // USDC
		common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"): 2,  // USDT
		common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F"): 2,  // DAI
		common.HexToAddress("0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599"): 0,  // WBTC
		common.HexToAddress("0xae7ab96520DE3A18E5e111B5EaAb095312D7fE84"): 0,  // stETH
		common.HexToAddress("0x6c3ea9036406852006290770BEdFcAbA0e23A0e8"): 0,  // PYUSD
		common.HexToAddress("0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0"): 0,  // wstETH
		common.HexToAddress("0xae78736Cd615f374D3085123A210448E74Fc6393"): 1,  // rETH
		common.HexToAddress("0xD533a949740bb3306d119CC777fa900bA034cd52"): 3,  // CRV
	},
	56: { // BSC
		common.HexToAddress("0x5d3a1Ff2b6BAb83b63cd9AD0787074081a52ef34"): 5, // USDe
		common.HexToAddress("0xfb6115445Bff7b52FeB98650C87f44907E58f802"): 1, // AAVE
		common.HexToAddress("0x4338665CBB7B2485A8855A139b75D5e34AB0DB94"): 1, // LTC
		common.HexToAddress("0x7083609fCE4d1d8Dc0C979AAb8c869Ea2C873402"): 1, // DOT
		common.HexToAddress("0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c"): 1, // WBNB
		common.HexToAddress("0x55d398326f99059fF775485246999027B3197955"): 1, // USDT
		common.HexToAddress("0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d"): 1, // USDC
		common.HexToAddress("0xe9e7CEA3DedcA5984780Bafc599bD69ADd087D56"): 1, // BUSD
		common.HexToAddress("0x1AF3F329e8BE154074D8769D1FFa4eE058B1DBc3"): 1, // DAI
		common.HexToAddress("0x2170Ed0880ac9A755fd29B2688956BD959F933F8"): 1, // ETH
		common.HexToAddress("0x7130d2A12B9BCbFAe4f2634d864A1Ee1Ce3Ead9c"): 1, // BTCB
		common.HexToAddress("0x0E09FaBB73Bd3Ade0a17ECC321fD13a19e81cE82"): 0, // CAKE
	},
	130: { // Unichain
		common.HexToAddress("0x078D782b760474a361dDA0AF3839290b0EF57AD6"): 9,  // USDC
		common.HexToAddress("0x9151434b16b9763660705744891fA906F660EcC5"): 51, // USDT
		common.HexToAddress("0x0555E30da8f98308EdB960aa94C0Db47230d2B9c"): 5,  // WBTC
		common.HexToAddress("0x4200000000000000000000000000000000000006"): 0,  // WETH
		common.HexToAddress("0x8f187aA05619a017077f5308904739877ce9eA21"): 0,  // UNI
		common.HexToAddress("0x15D0e0c55a3E7eE67152aD7E89acf164253Ff68d"): 0,  // HYPE
		common.HexToAddress("0x20CAb320A855b39F724131C69424240519573f81"): 0,  // DAI
		common.HexToAddress("0x588CE4F028D8e7B53B687865d6A67b3A54C75518"): 0,  // USDTb
		common.HexToAddress("0x7E10036Acc4B56d4dFCa3b77810356CE52313F9C"): 2,  // USDS
		common.HexToAddress("0xA06b10Db9F390990364A3984C04FaDf1c13691b5"): 2,  // sUSDS
	},
}

// LookupBalanceSlot returns the storage slot index where `token` stores its
// `balances` mapping head. ok=false if no static entry exists.
func LookupBalanceSlot(chainID int64, token common.Address) (uint64, bool) {
	if m, exists := balanceSlots[chainID]; exists {
		slot, ok := m[token]
		return slot, ok
	}
	return 0, false
}

// RegisterBalanceSlot lets callers add a slot mapping at runtime — useful when
// auto-detected slots want to be cached, or for tokens not yet in the static map.
func RegisterBalanceSlot(chainID int64, token common.Address, slot uint64) {
	if balanceSlots[chainID] == nil {
		balanceSlots[chainID] = make(map[common.Address]uint64)
	}
	balanceSlots[chainID][token] = slot
}
