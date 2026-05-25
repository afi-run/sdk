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
        statements: 95,
        branches: 95,
        functions: 95,
        lines: 95,
      },
    },
  },
})
