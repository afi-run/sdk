.PHONY: test test-coverage

COVERAGE_THRESHOLD := 95

test:
	go test ./...

# Enforces ≥$(COVERAGE_THRESHOLD)% coverage on unit-testable functions.
# Excludes: client.go, swap.go, token.go, multicall.go (require live RPC — covered by integration tests)
#           fetchTokens, fetchQuote                    (thin wrappers to production URLs)
#           extractRevertData                          (only exercised when an RPC error arrives)
test-coverage:
	@go test -coverprofile=coverage.out ./...
	@echo ""
	@echo "Coverage breakdown (all functions):"
	@go tool cover -func=coverage.out
	@echo ""
	@LC_ALL=C go tool cover -func=coverage.out \
		| grep -v -E "/(client|swap|token|multicall)\.go:|fetchTokens[[:space:]]|fetchQuote[[:space:]]|^total:" \
		| LC_ALL=C awk -v t=$(COVERAGE_THRESHOLD) '{gsub(/%/,"",$$3);s+=$$3;n++} END{avg=s/n; printf "Testable coverage: %.1f%% (threshold: %d%%)\n",avg,t; if(avg<t){printf "FAIL: %.1f%% < %d%%\n",avg,t;exit 1}else print "PASS"}'
	@rm -f coverage.out
