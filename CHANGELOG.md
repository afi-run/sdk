# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/) and the project
adheres to Semantic Versioning.

## [Unreleased]

### Added

#### Deployed contract addresses (2026-05-30)
- Afi, RouteQuoter addresses populated for 5 chains: Ethereum, BSC,
  Unichain, Base, Arbitrum.

#### High-level workflows (operator + admin)
- `client.swapFor`, `client.batchSwapFor` — operator swaps on behalf of users
- 12 admin methods: `adminPause`, `adminUnpause`, `adminSetTreasury`,
  `adminSetFeeBps`, `adminSetUserFeeBps`, `adminSetUserFeeBpsBatch`,
  `adminClearUserFeeBps`, `adminResetAnyUserOverride`, `adminAddRule`,
  `adminClearRules`, `adminSetOperator`, `adminRescueTokens`
- `client.acceptOwnership(addr)` / `acceptOwnershipBatch(addrs)` — Ownable2Step
  handover completion

#### Read methods
- `isPaused`, `getFeeBpsOf`, `hasRules`, `getTreasuryAddress`,
  `getRegistryAddress`, `getPrimaryOperator`, `isAfiOperator`, `getOwner`,
  `getPendingOwner`, `getRoute`, `listRoutes`, `getTreasuryBalance`,
  `verifyDeployment`

#### Off-chain route simulation
- `simulateRoute` (TS) / `SimulateRoute` (Go) — dry-run a route chain via
  `eth_call` + `state_override` against the RouteQuoter.
- `detectBalanceSlot`, `lookupBalanceSlot`, `registerBalanceSlot` — balance
  storage-slot helpers. `simulateRoute` auto-detects and caches the slot for any
  token not in the static fast-path table, so it works for the full backend
  token list without hand-maintained slot data.

#### Per-DEX step builders
- `buildUniV3Step`, `buildCakeV3Step`, `buildUniV4Step`, `buildAerodromeStep`,
  `buildBalancerV3Step`, `buildFluidStep`, `buildCurve128Step`, `buildCurve256Step`,
  `buildAaveLiquidatorStep`

#### HTTP quoter endpoints
- `findArbitrage`, `findPath`, `getRoutes`, `getLiquidationCandidates`,
  `liquidate`, `priceQuote`, `quoteDex`

#### Event parsers
- `parseSwapExecuted`, `parseFeeCollected`, `parseTreasuryUpdated`,
  `parseFeeBpsUpdated`, `parseUserFeeBpsSet`, `parseUserFeeBpsCleared`

#### Tight-format encoder
- `encodeSteps` — local pack of `uint8 numSteps + [uint16 id|uint16 dataLen|bytes data]×N`

#### Calldata encoders (low-level)
- 12 `encodeAfi*` admin encoders

#### Constants
- `AFI_ADDRESSES`, `ROUTE_QUOTER_ADDRESSES` (chain-id keyed records)
- `AFI_ABI`, `ROUTE_REGISTRY_ABI`
- `MAX_FEE_BPS = 50`

### Fixed
- Nonce race condition in `allocateNonce` under concurrent async sends.
- Gas estimation failures now surface the underlying contract revert reason.

### Documentation
- New example files (TypeScript + Go) covering operator batch, admin
  governance, and event indexing.
- 3 READMEs (EN, PT-BR, ES) expanded with: decision table at top, Operator
  workflows section, Admin/governance section, Event indexing section,
  Per-DEX step builders section, HTTP quoter endpoints section, Migration
  guide.

## [0.1.0] — Initial release
- User-facing swap (`client.swap`, `client.quote`), token info, fee read,
  approval flow, simulation via RouteQuoter, multicall reads.
