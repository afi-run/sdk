export class AfiError extends Error {
  constructor(
    message: string,
    public readonly code: string,
  ) {
    super(message)
    this.name = "AfiError"
  }
}

function formatWei(amount: bigint, decimals?: number): string {
  if (decimals === undefined) return amount.toString()
  const divisor = 10n ** BigInt(decimals)
  const whole = amount / divisor
  const frac = amount % divisor
  if (frac === 0n) return whole.toString()
  const fracStr = frac.toString().padStart(decimals, "0").replace(/0+$/, "")
  return `${whole}.${fracStr}`
}

export class InsufficientBalanceError extends AfiError {
  constructor(
    public readonly token: string,
    public readonly balance: bigint,
    public readonly required: bigint,
    public readonly owner?: string,
    public readonly symbol?: string,
    public readonly decimals?: number,
  ) {
    const tokenLabel = symbol ?? token
    const have = formatWei(balance, decimals)
    const need = formatWei(required, decimals)
    const ownerPart = owner ? ` for ${owner}` : ""
    super(
      `Insufficient ${tokenLabel}${ownerPart}: have ${have}, need ${need}`,
      "INSUFFICIENT_BALANCE",
    )
    this.name = "InsufficientBalanceError"
  }
}

export class QuoteError extends AfiError {
  constructor(reason: string) {
    super(`Quote failed: ${reason}`, "QUOTE_FAILED")
    this.name = "QuoteError"
  }
}

export class SimulationFailedError extends AfiError {
  constructor(
    public readonly reason: string,
    public readonly revertData?: string,
    /**
     * Structured decoded revert (when the data matched a known custom error).
     * Built-in AFI errors and standard Error/Panic are decoded automatically;
     * register your own with `registerCustomErrors(abi)`.
     */
    public readonly decoded?: { name: string; signature: string; args: readonly unknown[] },
  ) {
    super(`Swap simulation reverted: ${reason}`, "SIMULATION_FAILED")
    this.name = "SimulationFailedError"
  }
}

export class ApprovalError extends AfiError {
  constructor(reason: string) {
    super(`Token approval failed: ${reason}`, "APPROVAL_FAILED")
    this.name = "ApprovalError"
  }
}

export class SwapRevertedError extends AfiError {
  constructor(
    public readonly reason: string,
    public readonly decoded?: { name: string; signature: string; args: readonly unknown[] },
  ) {
    super(`Swap reverted: ${reason}`, "SWAP_REVERTED")
    this.name = "SwapRevertedError"
  }
}

export class NoSignerError extends AfiError {
  constructor() {
    super("Private key required — create the client with privateKey or call connect()", "NO_SIGNER")
    this.name = "NoSignerError"
  }
}

// ─── Type guards ───────────────────────────────────────────────────────────────
// Prefer these over `instanceof` when transpilers (esbuild, swc with class shims,
// older targets) may break the prototype chain.

export function isAfiError(e: unknown): e is AfiError {
  return e instanceof Error && typeof (e as AfiError).code === "string"
}

const guard = <T extends AfiError>(code: string) =>
  (e: unknown): e is T => isAfiError(e) && e.code === code

export const isInsufficientBalanceError = guard<InsufficientBalanceError>("INSUFFICIENT_BALANCE")
export const isQuoteError              = guard<QuoteError>("QUOTE_FAILED")
export const isSimulationFailedError   = guard<SimulationFailedError>("SIMULATION_FAILED")
export const isApprovalError           = guard<ApprovalError>("APPROVAL_FAILED")
export const isSwapRevertedError       = guard<SwapRevertedError>("SWAP_REVERTED")
export const isNoSignerError           = guard<NoSignerError>("NO_SIGNER")
