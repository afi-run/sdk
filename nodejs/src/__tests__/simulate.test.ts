import { describe, expect, it } from "vitest"
import { keccak256, encodeAbiParameters, getAddress, pad, type Hex } from "viem"
import { mappingSlot, simulateRoute, detectBalanceSlot } from "../simulate.js"
import { lookupBalanceSlot, registerBalanceSlot } from "../token-slots.js"
import type { Address } from "../types.js"

// A publicClient stub whose balanceOf reflects the overridden slot value
// (faithful EVM behavior), so detectBalanceSlot matches the first probed slot.
// quote() calls return an encoded (outputAsset, amountOut) tuple.
function fakeClient(asset: Address, amountOut: bigint) {
  return {
    call: async ({ data, stateOverride }: any) => {
      if (typeof data === "string" && data.startsWith("0x70a08231")) {
        return { data: stateOverride[0].stateDiff[0].value as Hex }
      }
      return {
        data: encodeAbiParameters(
          [{ type: "address" }, { type: "uint256" }],
          [asset, amountOut],
        ),
      }
    },
  } as never
}

describe("lookupBalanceSlot", () => {
  it("returns slot 3 for Base WETH", () => {
    const slot = lookupBalanceSlot(8453, "0x4200000000000000000000000000000000000006")
    expect(slot).toBe(3)
  })

  it("is case-insensitive", () => {
    const slot = lookupBalanceSlot(8453, "0x4200000000000000000000000000000000000006".toUpperCase() as Address)
    expect(slot).toBe(3)
  })

  it("returns undefined for unknown token", () => {
    const slot = lookupBalanceSlot(8453, "0x000000000000000000000000000000000000dEaD")
    expect(slot).toBeUndefined()
  })
})

describe("registerBalanceSlot", () => {
  it("adds runtime entries readable by lookup", () => {
    const addr = "0x000000000000000000000000000000000000bEEF" as Address
    registerBalanceSlot(8453, addr, 42)
    expect(lookupBalanceSlot(8453, addr)).toBe(42)
  })
})

describe("mappingSlot", () => {
  it("matches keccak256(holder . slot) by hand", () => {
    const holder = getAddress("0x1234567890abcdef1234567890abcdef12345678")
    const slot = 9
    const expected = keccak256(
      encodeAbiParameters(
        [{ type: "address" }, { type: "uint256" }],
        [holder, BigInt(slot)],
      ),
    )
    expect(mappingSlot(holder, slot)).toBe(expected)
  })

  it("accepts a bigint slot", () => {
    const holder = getAddress("0x1234567890abcdef1234567890abcdef12345678")
    expect(mappingSlot(holder, 9n)).toBe(mappingSlot(holder, 9))
  })

  it("is deterministic", () => {
    const h = getAddress("0x1234567890123456789012345678901234567890")
    expect(mappingSlot(h, 5)).toBe(mappingSlot(h, 5))
  })

  it("varies with slot", () => {
    const h = getAddress("0x1234567890123456789012345678901234567890")
    expect(mappingSlot(h, 1)).not.toBe(mappingSlot(h, 2))
  })
})

describe("simulateRoute input validation", () => {
  it("rejects zero amount", async () => {
    await expect(
      simulateRoute({
        publicClient: {} as never,
        chainId: 8453,
        quoterAddress: "0x0000000000000000000000000000000000000001",
        asset: "0x4200000000000000000000000000000000000006",
        amount: 0n,
        stepsEncoded: "0x00",
      }),
    ).rejects.toThrow(/amount must be > 0/)
  })

  it("auto-detects the slot for an unknown token and caches it", async () => {
    const asset = getAddress("0x000000000000000000000000000000000000cafe")
    expect(lookupBalanceSlot(8453, asset)).toBeUndefined()

    const res = await simulateRoute({
      publicClient: fakeClient(asset, 999n),
      chainId: 8453,
      quoterAddress: "0x0000000000000000000000000000000000000001",
      asset,
      amount: 1n,
      stepsEncoded: "0x00",
    })

    expect(res.reverted).toBe(false)
    expect(res.amountOut).toBe(999n)
    // slot detected on the first probe (0) and cached
    expect(lookupBalanceSlot(8453, asset)).toBe(0)
  })
})

describe("detectBalanceSlot", () => {
  it("detects and registers the backing slot", async () => {
    const token = "0x000000000000000000000000000000000000F00d" as Address
    expect(lookupBalanceSlot(1, token)).toBeUndefined()

    const slot = await detectBalanceSlot(fakeClient(token, 0n), 1, token)
    expect(slot).toBe(0)
    expect(lookupBalanceSlot(1, token)).toBe(0)
  })

  it("throws when no slot matches within maxSlot", async () => {
    // client that never echoes the sentinel → no slot ever matches
    const client = {
      call: async () => ({ data: pad("0x00" as Hex, { size: 32 }) }),
    } as never
    await expect(
      detectBalanceSlot(client, 1, "0x000000000000000000000000000000000000dEaD", 3),
    ).rejects.toThrow(/not detected/)
  })

  it("skips a slot whose eth_call throws and matches a later one", async () => {
    const sentinel = pad(`0x${(0xdeadbeefdeadbeefn).toString(16)}` as Hex, { size: 32 })
    let n = 0
    const client = {
      call: async () => {
        n++
        if (n === 1) throw new Error("rpc hiccup on slot 0")
        return { data: sentinel } // slot 1 echoes the sentinel
      },
    } as never
    const token = "0x000000000000000000000000000000000000Ab1e" as Address
    expect(await detectBalanceSlot(client, 1, token)).toBe(1)
  })
})

describe("simulateRoute revert handling", () => {
  const asset = getAddress("0x00000000000000000000000000000000000bEEf1")

  function throwingQuoteClient(err: unknown) {
    registerBalanceSlot(8453, asset, 0) // pre-register so detection is skipped
    return { call: async () => { throw err } } as never
  }

  async function run(err: unknown) {
    return simulateRoute({
      publicClient: throwingQuoteClient(err),
      chainId: 8453,
      quoterAddress: "0x0000000000000000000000000000000000000001",
      asset,
      amount: 1n,
      stepsEncoded: "0x00",
    })
  }

  it("extracts revert data from err.cause.data", async () => {
    const res = await run({ cause: { data: "0xdeadbeef" } })
    expect(res.reverted).toBe(true)
    expect(res.revertData).toBe("0xdeadbeef")
  })

  it("falls back to err.data", async () => {
    const res = await run({ data: "0xfeed" })
    expect(res.reverted).toBe(true)
    expect(res.revertData).toBe("0xfeed")
  })

  it("leaves revertData undefined when the error carries none", async () => {
    const res = await run(new Error("plain revert"))
    expect(res.reverted).toBe(true)
    expect(res.revertData).toBeUndefined()
  })

  it("treats an empty quote result as reverted", async () => {
    const empty = getAddress("0x00000000000000000000000000000000000bEEf2")
    registerBalanceSlot(8453, empty, 0) // skip detection
    const client = { call: async () => ({ data: undefined }) } as never
    const res = await simulateRoute({
      publicClient: client,
      chainId: 8453,
      quoterAddress: "0x0000000000000000000000000000000000000001",
      asset: empty,
      amount: 1n,
      stepsEncoded: "0x00",
    })
    expect(res.reverted).toBe(true)
  })
})
