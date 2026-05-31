# Builds every example main package into bin/<name>.
#
# bin/ is gitignored, so compiled example binaries never clutter `git status`
# regardless of how many examples exist — no per-example .gitignore entries.
#
# Standalone files in this directory (1_list_tokens.go, swap.go, …) carry a
# `//go:build ignore` tag and are run directly with `go run <file>.go`; only the
# subdirectory main packages are compiled here.

BIN := bin
EXAMPLES := $(sort $(notdir $(patsubst %/,%,$(dir $(wildcard */main.go)))))

.PHONY: build clean run list

## build: compile every example into bin/<name>
build:
	@mkdir -p $(BIN)
	@for ex in $(EXAMPLES); do \
		echo "  building $$ex"; \
		go build -o $(BIN)/$$ex ./$$ex || exit 1; \
	done
	@echo "✓ built $(words $(EXAMPLES)) examples into $(BIN)/"

## run: run a single example — make run EX=get-quote
run:
	@test -n "$(EX)" || { echo "usage: make run EX=<example>  (one of: $(EXAMPLES))"; exit 1; }
	@go run ./$(EX)

## list: print the discovered examples
list:
	@echo $(EXAMPLES)

## clean: remove the bin/ output directory
clean:
	@rm -rf $(BIN)
