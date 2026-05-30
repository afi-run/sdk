import { defineConfig } from "vitest/config"

export default defineConfig({
  test: {
    coverage: {
      provider: "v8",
      include: ["src/**/*.ts"],
      exclude: [
        "src/client.ts",       // requires live RPC — covered by integration tests
        "src/swap.ts",         // requires live RPC — covered by integration tests
        "src/index.ts",        // re-exports only
        "src/types.ts",        // type declarations only
        "src/**/__tests__/**",
      ],
      thresholds: {
        // Branches sits at 90 (statements/lines/functions stay at 95) because the codebase
        // has defensive `?? fallback` patterns whose else-branch is naturally hard to trigger
        // without exotic mocks — driving those to 95 burns hours for negligible safety gain.
        statements: 95,
        branches: 90,
        functions: 95,
        lines: 95,
      },
    },
  },
})
