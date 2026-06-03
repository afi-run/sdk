import { describe, expect, it } from "vitest"
import { decodeFunctionData, toFunctionSelector } from "viem"
import { AFI_REFERRAL_ROUTER_ABI } from "../constants.js"
import {
  REFERRAL_HARD_CAP_BPS,
  referralRouterAddress,
  encodeSwapWithReferral,
  encodeSwapWithReferralFor,
  encodeReferralClaim,
  encodeReferralClaimMany,
  encodeSetDelegateAllowance,
  encodeRevokeDelegate,
  encodeReferralPause,
  encodeReferralUnpause,
  encodeReferralSetMaxReferralBps,
  encodeReferralRescueTokens,
} from "../referral.js"

const TOKEN_IN  = "0x1111111111111111111111111111111111111111" as const
const TOKEN_OUT = "0x2222222222222222222222222222222222222222" as const
const REFERRER  = "0x3333333333333333333333333333333333333333" as const
const USER      = "0x4444444444444444444444444444444444444444" as const
const DELEGATE  = "0x5555555555555555555555555555555555555555" as const
const TO        = "0x6666666666666666666666666666666666666666" as const
const ZERO      = "0x0000000000000000000000000000000000000000" as const

function assertCall(data: string, fnName: string, signature: string) {
  expect(data).toMatch(/^0x[0-9a-f]+$/i)
  expect(data.startsWith(toFunctionSelector(signature))).toBe(true)
  const decoded = decodeFunctionData({ abi: AFI_REFERRAL_ROUTER_ABI, data: data as `0x${string}` })
  expect(decoded.functionName).toBe(fnName)
  return decoded
}

describe("referralRouterAddress", () => {
  it("resolves deployed chains", () => {
    expect(referralRouterAddress(8453)).toBe("0x2dC7a3990618baa91c450521004F14A334BF47c6")
    expect(referralRouterAddress(1)).toBe("0x47E7cE4237130F02202e081Efa1Fd338F23Ead77")
  })
  it("throws for an undeployed chain", () => {
    expect(() => referralRouterAddress(999999)).toThrow()
  })
})

describe("encodeSwapWithReferral", () => {
  it("encodes swapWithReferral(...)", () => {
    const data = encodeSwapWithReferral({
      tokenIn: TOKEN_IN, amountIn: 1000n, tokenOut: TOKEN_OUT, minOut: 900n,
      params: "0x0102", referrer: REFERRER, referralBps: 5,
    })
    const decoded = assertCall(data, "swapWithReferral",
      "swapWithReferral(address,uint256,address,uint256,bytes,address,uint16)")
    const args = decoded.args as readonly unknown[]
    expect(args[0]).toBe(TOKEN_IN)
    expect(args[1]).toBe(1000n)
    expect(args[6]).toBe(5)
  })
  it("rejects zero amountIn", () => {
    expect(() => encodeSwapWithReferral({
      tokenIn: TOKEN_IN, amountIn: 0n, tokenOut: TOKEN_OUT, minOut: 0n,
      params: "0x", referrer: REFERRER, referralBps: 5,
    })).toThrow()
  })
  it("rejects referralBps above the hard cap", () => {
    expect(() => encodeSwapWithReferral({
      tokenIn: TOKEN_IN, amountIn: 1n, tokenOut: TOKEN_OUT, minOut: 0n,
      params: "0x", referrer: REFERRER, referralBps: REFERRAL_HARD_CAP_BPS + 1,
    })).toThrow()
  })
})

describe("encodeSwapWithReferralFor", () => {
  it("encodes swapWithReferralFor(...)", () => {
    const data = encodeSwapWithReferralFor({
      user: USER, tokenIn: TOKEN_IN, amountIn: 1000n, tokenOut: TOKEN_OUT, minOut: 900n,
      params: "0x", referrer: REFERRER, referralBps: 10,
    })
    assertCall(data, "swapWithReferralFor",
      "swapWithReferralFor(address,address,uint256,address,uint256,bytes,address,uint16)")
  })
  it("rejects zero user / zero amount / bps too high", () => {
    const base = { tokenIn: TOKEN_IN, tokenOut: TOKEN_OUT, minOut: 0n, params: "0x" as const, referrer: REFERRER }
    expect(() => encodeSwapWithReferralFor({ ...base, user: ZERO, amountIn: 1n, referralBps: 5 })).toThrow()
    expect(() => encodeSwapWithReferralFor({ ...base, user: USER, amountIn: 0n, referralBps: 5 })).toThrow()
    expect(() => encodeSwapWithReferralFor({ ...base, user: USER, amountIn: 1n, referralBps: REFERRAL_HARD_CAP_BPS + 1 })).toThrow()
  })
})

describe("encodeReferralClaim", () => {
  it("encodes claim(token,to)", () => {
    const decoded = assertCall(encodeReferralClaim(TOKEN_OUT, TO), "claim", "claim(address,address)")
    const args = decoded.args as readonly unknown[]
    expect(args[0]).toBe(TOKEN_OUT)
    expect(args[1]).toBe(TO)
  })
  it("rejects zero to", () => {
    expect(() => encodeReferralClaim(TOKEN_OUT, ZERO)).toThrow()
  })
})

describe("encodeReferralClaimMany", () => {
  it("encodes claimMany(tokens,to)", () => {
    assertCall(encodeReferralClaimMany([TOKEN_IN, TOKEN_OUT], TO), "claimMany", "claimMany(address[],address)")
  })
  it("rejects empty tokens and zero to", () => {
    expect(() => encodeReferralClaimMany([], TO)).toThrow()
    expect(() => encodeReferralClaimMany([TOKEN_IN], ZERO)).toThrow()
  })
})

describe("encodeSetDelegateAllowance", () => {
  it("encodes setDelegateAllowance(token,delegate,uint208,uint48)", () => {
    const decoded = assertCall(
      encodeSetDelegateAllowance(TOKEN_IN, DELEGATE, 500n, 1893456000),
      "setDelegateAllowance",
      "setDelegateAllowance(address,address,uint208,uint48)",
    )
    const args = decoded.args as readonly unknown[]
    expect(args[2]).toBe(500n)
    expect(args[3]).toBe(1893456000)
  })
  it("rejects zero delegate / over-uint208 amount / over-uint48 deadline", () => {
    expect(() => encodeSetDelegateAllowance(TOKEN_IN, ZERO, 1n, 1)).toThrow()
    expect(() => encodeSetDelegateAllowance(TOKEN_IN, DELEGATE, 1n << 208n, 1)).toThrow()
    expect(() => encodeSetDelegateAllowance(TOKEN_IN, DELEGATE, 1n, 2 ** 48)).toThrow()
  })
})

describe("encodeRevokeDelegate", () => {
  it("encodes revokeDelegate(token,delegate)", () => {
    assertCall(encodeRevokeDelegate(TOKEN_IN, DELEGATE), "revokeDelegate", "revokeDelegate(address,address)")
  })
  it("rejects zero delegate", () => {
    expect(() => encodeRevokeDelegate(TOKEN_IN, ZERO)).toThrow()
  })
})

describe("owner-only controls", () => {
  it("encodes pause() / unpause()", () => {
    assertCall(encodeReferralPause(), "pause", "pause()")
    assertCall(encodeReferralUnpause(), "unpause", "unpause()")
  })
  it("encodes setMaxReferralBps(uint16) and rejects above cap", () => {
    assertCall(encodeReferralSetMaxReferralBps(REFERRAL_HARD_CAP_BPS), "setMaxReferralBps", "setMaxReferralBps(uint16)")
    expect(() => encodeReferralSetMaxReferralBps(REFERRAL_HARD_CAP_BPS + 1)).toThrow()
  })
  it("encodes rescueTokens(token,to) and rejects zero to", () => {
    assertCall(encodeReferralRescueTokens(TOKEN_OUT, TO), "rescueTokens", "rescueTokens(address,address)")
    expect(() => encodeReferralRescueTokens(TOKEN_OUT, ZERO)).toThrow()
  })
})
