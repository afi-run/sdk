import { describe, expect, it } from "vitest"
import {
  encodeAbiParameters,
  encodeEventTopics,
  pad,
  type Log,
} from "viem"
import { AFI_ABI, NMR_ABI } from "../constants.js"
import {
  parseFeeBpsUpdated,
  parseFeeCollected,
  parseFlashLoanExecuted,
  parseFlashLoanFailed,
  parseFlashLoanFailedWithData,
  parseFlashLoanRequested,
  parseNmrSwapExecuted,
  parseProfitShareUpdated,
  parseProfitSwept,
  parseSwapExecuted,
  parseTreasuryUpdated,
  parseUserFeeBpsCleared,
  parseUserFeeBpsSet,
} from "../events.js"

const ADDR_A = "0x1111111111111111111111111111111111111111" as const
const ADDR_B = "0x2222222222222222222222222222222222222222" as const
const ADDR_C = "0x3333333333333333333333333333333333333333" as const

function emptyLog(): Log {
  return {
    address: ADDR_A,
    blockHash: pad("0x01", { size: 32 }),
    blockNumber: 1n,
    data: "0x",
    logIndex: 0,
    removed: false,
    topics: [],
    transactionHash: pad("0x02", { size: 32 }),
    transactionIndex: 0,
  }
}

function makeLog(
  abi: typeof AFI_ABI | typeof NMR_ABI,
  eventName: string,
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  indexedArgs: any,
  dataTypes: readonly { type: string }[],
  dataValues: readonly unknown[],
): Log {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const topics = encodeEventTopics<any, any>({
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    abi: abi as any,
    eventName,
    args: indexedArgs,
  })
  const data = dataTypes.length > 0
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    ? encodeAbiParameters(dataTypes as any, dataValues as any)
    : "0x"
  return {
    ...emptyLog(),
    topics: topics as never,
    data,
  }
}

describe("parseSwapExecuted", () => {
  it("decodes SwapExecuted logs", () => {
    const log = makeLog(
      AFI_ABI,
      "SwapExecuted",
      { from: ADDR_A, assetIn: ADDR_B, assetOut: ADDR_C },
      [{ type: "uint256" }, { type: "uint256" }],
      [100n, 200n],
    )
    const parsed = parseSwapExecuted([log])
    expect(parsed).toHaveLength(1)
    expect(parsed[0].from.toLowerCase()).toBe(ADDR_A)
    expect(parsed[0].assetIn.toLowerCase()).toBe(ADDR_B)
    expect(parsed[0].assetOut.toLowerCase()).toBe(ADDR_C)
    expect(parsed[0].amountIn).toBe(100n)
    expect(parsed[0].amountOut).toBe(200n)
  })

  it("skips non-matching logs silently", () => {
    expect(parseSwapExecuted([emptyLog()])).toEqual([])
  })
})

describe("parseFeeCollected", () => {
  it("decodes FeeCollected logs", () => {
    const log = makeLog(
      AFI_ABI,
      "FeeCollected",
      { token: ADDR_A },
      [{ type: "uint256" }],
      [42n],
    )
    const parsed = parseFeeCollected([log])
    expect(parsed).toHaveLength(1)
    expect(parsed[0].token.toLowerCase()).toBe(ADDR_A)
    expect(parsed[0].amount).toBe(42n)
  })
})

describe("parseTreasuryUpdated", () => {
  it("decodes Afi TreasuryUpdated logs", () => {
    const log = makeLog(
      AFI_ABI,
      "TreasuryUpdated",
      { treasury: ADDR_B },
      [],
      [],
    )
    const parsed = parseTreasuryUpdated([log])
    expect(parsed).toHaveLength(1)
    expect(parsed[0].treasury.toLowerCase()).toBe(ADDR_B)
  })
})

describe("parseFeeBpsUpdated", () => {
  it("decodes FeeBpsUpdated logs", () => {
    const log = makeLog(
      AFI_ABI,
      "FeeBpsUpdated",
      {},
      [{ type: "uint16" }],
      [25],
    )
    const parsed = parseFeeBpsUpdated([log])
    expect(parsed).toHaveLength(1)
    expect(parsed[0].feeBps).toBe(25)
  })
})

describe("parseUserFeeBpsSet", () => {
  it("decodes UserFeeBpsSet logs", () => {
    const log = makeLog(
      AFI_ABI,
      "UserFeeBpsSet",
      { user: ADDR_C },
      [{ type: "uint16" }],
      [99],
    )
    const parsed = parseUserFeeBpsSet([log])
    expect(parsed).toHaveLength(1)
    expect(parsed[0].user.toLowerCase()).toBe(ADDR_C)
    expect(parsed[0].feeBps).toBe(99)
  })
})

describe("parseUserFeeBpsCleared", () => {
  it("decodes UserFeeBpsCleared logs", () => {
    const log = makeLog(
      AFI_ABI,
      "UserFeeBpsCleared",
      { user: ADDR_A },
      [],
      [],
    )
    const parsed = parseUserFeeBpsCleared([log])
    expect(parsed).toHaveLength(1)
    expect(parsed[0].user.toLowerCase()).toBe(ADDR_A)
  })
})

describe("parseFlashLoanRequested", () => {
  it("decodes FlashLoanRequested logs", () => {
    const log = makeLog(
      NMR_ABI,
      "FlashLoanRequested",
      { asset: ADDR_A },
      [{ type: "uint256" }],
      [1_000n],
    )
    const parsed = parseFlashLoanRequested([log])
    expect(parsed).toHaveLength(1)
    expect(parsed[0].asset.toLowerCase()).toBe(ADDR_A)
    expect(parsed[0].amount).toBe(1_000n)
  })
})

describe("parseFlashLoanExecuted", () => {
  it("decodes FlashLoanExecuted logs", () => {
    const log = makeLog(
      NMR_ABI,
      "FlashLoanExecuted",
      { asset: ADDR_B },
      [{ type: "uint256" }, { type: "uint256" }, { type: "uint256" }],
      [1_000n, 5n, 10n],
    )
    const parsed = parseFlashLoanExecuted([log])
    expect(parsed).toHaveLength(1)
    expect(parsed[0].asset.toLowerCase()).toBe(ADDR_B)
    expect(parsed[0].amount).toBe(1_000n)
    expect(parsed[0].premium).toBe(5n)
    expect(parsed[0].profit).toBe(10n)
  })
})

describe("parseFlashLoanFailed", () => {
  it("decodes FlashLoanFailed logs", () => {
    const log = makeLog(
      NMR_ABI,
      "FlashLoanFailed",
      { asset: ADDR_C },
      [{ type: "uint256" }, { type: "string" }],
      [1_000n, "oops"],
    )
    const parsed = parseFlashLoanFailed([log])
    expect(parsed).toHaveLength(1)
    expect(parsed[0].asset.toLowerCase()).toBe(ADDR_C)
    expect(parsed[0].amount).toBe(1_000n)
    expect(parsed[0].reason).toBe("oops")
  })
})

describe("parseFlashLoanFailedWithData", () => {
  it("decodes FlashLoanFailedWithData logs", () => {
    const log = makeLog(
      NMR_ABI,
      "FlashLoanFailedWithData",
      { asset: ADDR_C },
      [{ type: "uint256" }, { type: "bytes" }],
      [1_000n, "0x08c379a0"],
    )
    const parsed = parseFlashLoanFailedWithData([log])
    expect(parsed).toHaveLength(1)
    expect(parsed[0].asset.toLowerCase()).toBe(ADDR_C)
    expect(parsed[0].amount).toBe(1_000n)
    expect(parsed[0].data).toBe("0x08c379a0")
  })
})

describe("parseNmrSwapExecuted", () => {
  it("decodes NMR SwapExecuted logs (no `from`, distinct from Afi)", () => {
    const log = makeLog(
      NMR_ABI,
      "SwapExecuted",
      { assetIn: ADDR_B, assetOut: ADDR_C },
      [{ type: "uint256" }, { type: "uint256" }],
      [500n, 525n],
    )
    const parsed = parseNmrSwapExecuted([log])
    expect(parsed).toHaveLength(1)
    expect(parsed[0].assetIn.toLowerCase()).toBe(ADDR_B)
    expect(parsed[0].assetOut.toLowerCase()).toBe(ADDR_C)
    expect(parsed[0].amountIn).toBe(500n)
    expect(parsed[0].amountOut).toBe(525n)
  })

  it("does not match Afi SwapExecuted (different topic0)", () => {
    const afiLog = makeLog(
      AFI_ABI,
      "SwapExecuted",
      { from: ADDR_A, assetIn: ADDR_B, assetOut: ADDR_C },
      [{ type: "uint256" }, { type: "uint256" }],
      [100n, 200n],
    )
    expect(parseNmrSwapExecuted([afiLog])).toHaveLength(0)
  })
})

describe("parseProfitSwept", () => {
  it("decodes ProfitSwept logs", () => {
    const log = makeLog(
      NMR_ABI,
      "ProfitSwept",
      { asset: ADDR_A, to: ADDR_B },
      [{ type: "uint256" }],
      [777n],
    )
    const parsed = parseProfitSwept([log])
    expect(parsed).toHaveLength(1)
    expect(parsed[0].asset.toLowerCase()).toBe(ADDR_A)
    expect(parsed[0].to.toLowerCase()).toBe(ADDR_B)
    expect(parsed[0].amount).toBe(777n)
  })
})

describe("parseProfitShareUpdated", () => {
  it("decodes ProfitShareUpdated logs", () => {
    const log = makeLog(
      NMR_ABI,
      "ProfitShareUpdated",
      {},
      [{ type: "uint8" }],
      [50],
    )
    const parsed = parseProfitShareUpdated([log])
    expect(parsed).toHaveLength(1)
    expect(parsed[0].profitShare).toBe(50)
  })
})

describe("parsers ignore unrelated logs", () => {
  it("returns [] when no logs match", () => {
    const noise = emptyLog()
    expect(parseSwapExecuted([noise])).toEqual([])
    expect(parseFlashLoanExecuted([noise])).toEqual([])
    expect(parseProfitSwept([noise])).toEqual([])
  })
})
