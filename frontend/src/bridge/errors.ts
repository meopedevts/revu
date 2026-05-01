export class BridgeUnavailableError extends Error {
  constructor() {
    super("Wails bridge not available (window.go.app.App missing)")
    this.name = "BridgeUnavailableError"
  }
}

export class BridgeMethodMissingError extends Error {
  constructor(public readonly method: string) {
    super(`Bridge method ${method} not wired in current build`)
    this.name = "BridgeMethodMissingError"
  }
}
