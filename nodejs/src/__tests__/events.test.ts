import { describe, expect, it } from "vitest"
import {
  encodeAbiParameters,
  encodeEventTopics,
  pad,
  type Log,
} from "viem"
import { AFI_ABI } from "../constants.js"
import {
  parseFeeBpsUpdated,
  parseFeeCollected,
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
  abi: typeof AFI_ABI,
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

describe("parsers ignore unrelated logs", () => {
  it("returns [] when no logs match", () => {
    const noise = emptyLog()
    expect(parseSwapExecuted([noise])).toEqual([])
    expect(parseFeeCollected([noise])).toEqual([])
  })

  it("skips a log whose topic0 doesn't decode (catch path)", () => {
    // Non-empty topics that don't correspond to any AFI event → decodeEventLog throws.
    const garbage: Log = {
      ...emptyLog(),
      topics: [pad("0xdeadbeef", { size: 32 }) as `0x${string}`],
      data: "0x",
    }
    expect(parseSwapExecuted([garbage])).toEqual([])
  })

  it("skips a valid log of a different event (eventName mismatch)", () => {
    // A real FeeCollected log handed to the SwapExecuted parser.
    const feeLog = makeLog(
      AFI_ABI,
      "FeeCollected",
      { token: ADDR_A },
      [{ type: "uint256" }],
      [1000n],
    )
    expect(parseSwapExecuted([feeLog])).toEqual([])
    // ...but the matching parser still decodes it.
    expect(parseFeeCollected([feeLog])).toHaveLength(1)
  })
})
