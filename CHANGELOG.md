# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/) and the project
adheres to Semantic Versioning.

## [Unreleased]

### Added

#### Deployed contract addresses (2026-05-30)
- Afi, RouteQuoter, NMR addresses populated for 5 chains: Ethereum, BSC,
  Unichain, Base, Arbitrum. NMR present only on Aave V3 chains (Eth/Base/Arb).

#### High-level workflows (operator + admin)
- `client.swapFor`, `client.batchSwapFor` — operator swaps on behalf of users
- `client.executeNMRArbitrage`, `client.nmrCycleSwap`, `client.nmrLoanArbitrage`
  — NMR flash-loan arbitrage paths
- `client.sweepNMRProfit` — operator withdraw of accumulated NMR profit
- 12 admin methods: `adminPause`, `adminUnpause`, `adminSetTreasury`,
  `adminSetFeeBps`, `adminSetUserFeeBps`, `adminSetUserFeeBpsBatch`,
  `adminClearUserFeeBps`, `adminResetAnyUserOverride`, `adminAddRule`,
  `adminClearRules`, `adminSetOperator`, `adminRescueTokens`
- `client.acceptOwnership(addr)` / `acceptOwnershipBatch(addrs)` — Ownable2Step
  handover completion

#### Read methods
- `isPaused`, `getFeeBpsOf`, `hasRules`, `getTreasuryAddress`,
  `getRegistryAddress`, `getPrimaryOperator`, `isAfiOperator`, `getOwner`,
  `getPendingOwner`, `getNMRTreasury`, `getNMRProfitShare`, `isNMROperator`,
  `getRoute`, `listRoutes`, `getTreasuryBalance`, `verifyDeployment`

#### Per-DEX step builders
- `buildUniV3Step`, `buildCakeV3Step`, `buildUniV4Step`, `buildAerodromeStep`,
  `buildBalancerV3Step`, `buildFluidStep`, `buildCurve128Step`, `buildCurve256Step`,
  `buildAaveLiquidatorStep`

#### HTTP quoter endpoints
- `findArbitrage`, `findPath`, `getRoutes`, `getLiquidationCandidates`,
  `liquidate`, `priceQuote`, `quoteDex`

#### Event parsers
- `parseSwapExecuted`, `parseFeeCollected`, `parseTreasuryUpdated`,
  `parseFeeBpsUpdated`, `parseUserFeeBpsSet`, `parseUserFeeBpsCleared`,
  `parseFlashLoanRequested`, `parseFlashLoanExecuted`, `parseFlashLoanFailed`,
  `parseProfitSwept`, `parseProfitShareUpdated`

#### Tight-format encoder
- `encodeSteps` — local pack of `uint8 numSteps + [uint16 id|uint16 dataLen|bytes data]×N`

#### Calldata encoders (low-level)
- `encodeNMRRequestOperation`, `encodeNMRSwap`, `encodeNMRLoan`,
  `encodeNMRSweepProfit`, `encodeNMRSetTreasury`
- 12 `encodeAfi*` admin encoders

#### Constants
- `AFI_ADDRESSES`, `ROUTE_QUOTER_ADDRESSES`, `NMR_ADDRESSES` (chain-id keyed records)
- `AFI_ABI`, `NMR_ABI`, `ROUTE_REGISTRY_ABI`
- `MAX_FEE_BPS = 50`, `MAX_PROFIT_SHARE = 50`

### Fixed
- Critical ABI mismatch: `NMR.loan` parameter order corrected (was `(asset, tokenOut, ...)` should be `(user, asset, amount, minOut, params)`).
- ABI mismatches for `ProfitSwept` event (renamed fields `token, treasury` -> `asset, to`) and `ProfitShareUpdated` (renamed `share` -> `profitShare`).
- Nonce race condition in `allocateNonce` under concurrent async sends.
- Gas estimation failures now surface the underlying contract revert reason.

### Documentation
- 10 new example files (5 TypeScript + 5 Go) covering operator batch, NMR
  arbitrage (cycle + loan), admin governance, and event indexing.
- 3 READMEs (EN, PT-BR, ES) expanded with: decision table at top, Operator
  workflows section, Admin/governance section, Event indexing section,
  Per-DEX step builders section, HTTP quoter endpoints section, Migration
  guide.

## [0.1.0] — Initial release
- User-facing swap (`client.swap`, `client.quote`), token info, fee read,
  approval flow, simulation via RouteQuoter, multicall reads.
