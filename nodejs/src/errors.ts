export class AfiError extends Error {
  constructor(
    message: string,
    public readonly code: string,
  ) {
    super(message)
    this.name = "AfiError"
  }
}

export class InsufficientBalanceError extends AfiError {
  constructor(
    public readonly token: string,
    public readonly balance: bigint,
    public readonly required: bigint,
  ) {
    super(
      `Insufficient balance for token ${token}: have ${balance}, need ${required}`,
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
  constructor(public readonly reason: string) {
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
