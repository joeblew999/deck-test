# Makefile to install deck tooling and fetch sample data
GO ?= go
GIT ?= git

BINS := decksh dshfmt dshlint ebdeck pdfdeck pngdeck svgdeck giftsh gift gcdeck
# decksh:   https://github.com/ajstarks/decksh/tree/master/cmd/decksh
# dshfmt:   https://github.com/ajstarks/decksh/tree/master/cmd/dshfmt
# dshlint:  https://github.com/ajstarks/decksh/tree/master/cmd/dshlint
# ebdeck:   https://github.com/ajstarks/ebcanvas/tree/main/ebdeck
# pdfdeck:  https://github.com/ajstarks/deck/tree/master/cmd/pdfdeck
# pngdeck:  https://github.com/ajstarks/deck/tree/master/cmd/pngdeck
# svgdeck:  https://github.com/ajstarks/deck/tree/master/cmd/svgdeck
# giftsh:   https://github.com/ajstarks/giftsh
# gift:     https://github.com/ajstarks/gift
# gcdeck:   https://github.com/ajstarks/giocanvas/tree/master/gcdeck

decksh_pkg := github.com/ajstarks/decksh/cmd/decksh
dshfmt_pkg := github.com/ajstarks/decksh/cmd/dshfmt
dshlint_pkg := github.com/ajstarks/decksh/cmd/dshlint
ebdeck_pkg := github.com/ajstarks/ebcanvas/ebdeck
pdfdeck_pkg := github.com/ajstarks/deck/cmd/pdfdeck
pngdeck_pkg := github.com/ajstarks/deck/cmd/pngdeck
svgdeck_pkg := github.com/ajstarks/deck/cmd/svgdeck
giftsh_pkg := github.com/ajstarks/giftsh
gift_pkg := github.com/ajstarks/gift
gcdeck_pkg := github.com/ajstarks/giocanvas/gcdeck

DECKVIZ_REPO := https://github.com/ajstarks/deckviz.git
DECKVIZ_DIR ?= deckviz
DECKVIZ_BRANCH ?= master
DECKVIZ_DEPTH ?= 1

DECKFONTS_REPO := https://github.com/ajstarks/deckfonts.git
DECKFONTS_DIR ?= deckfonts
DECKFONTS_BRANCH ?= master
DECKFONTS_DEPTH ?= 1

DUBOIS_REPO := https://github.com/ajstarks/dubois-data-portraits.git
DUBOIS_DIR ?= dubois-data-portraits
DUBOIS_BRANCH ?= master
DUBOIS_DEPTH ?= 1
DUBOIS_FILTER ?= --filter=blob:none
DUBOIS_SPARSE ?=

DECKFONTS_DEFAULT := $(abspath $(DECKFONTS_DIR))
DECKFONTS ?= $(DECKFONTS_DEFAULT)
export DECKFONTS

GOBIN_DIR := $(shell $(GO) env GOBIN)
ifeq ($(strip $(GOBIN_DIR)),)
GOBIN_DIR := $(shell $(GO) env GOPATH)/bin
endif

DECKSH_BIN := $(GOBIN_DIR)/decksh
DSHFMT_BIN := $(GOBIN_DIR)/dshfmt
DSHLINT_BIN := $(GOBIN_DIR)/dshlint
EBDECK_BIN := $(GOBIN_DIR)/ebdeck
PDFDECK_BIN := $(GOBIN_DIR)/pdfdeck
PNGDECK_BIN := $(GOBIN_DIR)/pngdeck
SVGDECK_BIN := $(GOBIN_DIR)/svgdeck
GIFTSH_BIN := $(GOBIN_DIR)/giftsh
GIFT_BIN := $(GOBIN_DIR)/gift
GCDECK_BIN := $(GOBIN_DIR)/gcdeck

EXAMPLE ?= fire
EXAMPLES ?= $(EXAMPLE)

.DEFAULT_GOAL := help

.PHONY: help all install ensure-bins $(BINS) deckviz deckfonts dubois examples list-examples run-example run-examples view-example ebdeck decktool decktool-install decktool-completions

help:
	@echo "Deck tooling helper targets:"
	@echo "  make install          Install decksh, dshfmt, dshlint, ebdeck binaries"
	@echo "                        (also pdfdeck, pngdeck, svgdeck, giftsh, gift)"
	@echo "  make deckviz          Clone or update deckviz sample data (shallow fetch)"
	@echo "  make deckfonts        Clone or update deckfonts (shallow fetch)"
	@echo "  make dubois           Clone or update Du Bois data portraits (shallow fetch)"
	@echo "  make examples         List available deckviz examples"
	@echo "  make list-examples    Same as above (alias)"
	@echo "  make run-example      Lint and render one example (EXAMPLE=fire)"
	@echo "  make run-examples     Lint and render many examples (EXAMPLES=\"fire flag\")"
	@echo "  make view-example     Render and display an example via ebdeck"
	@echo "  make ebdeck           Show raw ebdeck usage examples"
	@echo "  make decktool         Build ./decktool binary for faster CLI use"
	@echo "  make decktool-install Install CLI into GOBIN for global usage"
	@echo "  make decktool-completions SHELL=zsh > file  # emit completion script"
	@echo ""
	@echo "Environment:"
	@echo "  Override DECKFONTS to point at your font checkout (default $(DECKFONTS_DIR))"
	@echo "  Set DUBOIS_FILTER=\"--filter=blob:none\" (default) or blank to control clone depth"
	@echo "  Set DUBOIS_SPARSE=\"plate-01 plate-02\" to sparse-checkout specific portraits"
	@echo ""
	@echo "Usage examples:"
	@echo "  make run-example EXAMPLE=fire"
	@echo "  make run-examples EXAMPLES=\"fire flag\""
	@echo "  make run-example EXAMPLE=dubois/baldwin"
	@echo "  make view-example EXAMPLE=dubois/baldwin"

all: install deckviz deckfonts dubois

install: ensure-bins

ensure-bins: $(BINS)

decksh: $(DECKSH_BIN)
dshfmt: $(DSHFMT_BIN)
dshlint: $(DSHLINT_BIN)
ebdeck: $(EBDECK_BIN)
pdfdeck: $(PDFDECK_BIN)
pngdeck: $(PNGDECK_BIN)
svgdeck: $(SVGDECK_BIN)
giftsh: $(GIFTSH_BIN)
gift: $(GIFT_BIN)
gcdeck: $(GCDECK_BIN)

$(DECKSH_BIN):
	$(GO) install $(decksh_pkg)@latest

$(DSHFMT_BIN):
	$(GO) install $(dshfmt_pkg)@latest

$(DSHLINT_BIN):
	$(GO) install $(dshlint_pkg)@latest

$(EBDECK_BIN):
	$(GO) install $(ebdeck_pkg)@latest

$(PDFDECK_BIN):
	$(GO) install $(pdfdeck_pkg)@latest

$(PNGDECK_BIN):
	$(GO) install $(pngdeck_pkg)@latest

$(SVGDECK_BIN):
	$(GO) install $(svgdeck_pkg)@latest

$(GIFTSH_BIN):
	$(GO) install $(giftsh_pkg)@latest

$(GIFT_BIN):
	$(GO) install $(gift_pkg)@latest

$(GCDECK_BIN):
	$(GO) install $(gcdeck_pkg)@latest

deckviz:
	@if [ -d "$(DECKVIZ_DIR)/.git" ]; then \
		$(GIT) -C "$(DECKVIZ_DIR)" fetch --depth=$(DECKVIZ_DEPTH) origin $(DECKVIZ_BRANCH); \
		$(GIT) -C "$(DECKVIZ_DIR)" checkout $(DECKVIZ_BRANCH); \
		$(GIT) -C "$(DECKVIZ_DIR)" reset --hard origin/$(DECKVIZ_BRANCH); \
	else \
		$(GIT) clone --depth=$(DECKVIZ_DEPTH) --branch $(DECKVIZ_BRANCH) $(DECKVIZ_REPO) "$(DECKVIZ_DIR)"; \
	fi

deckfonts:
	@if [ -d "$(DECKFONTS_DIR)/.git" ]; then \
		$(GIT) -C "$(DECKFONTS_DIR)" fetch --depth=$(DECKFONTS_DEPTH) origin $(DECKFONTS_BRANCH); \
		$(GIT) -C "$(DECKFONTS_DIR)" checkout $(DECKFONTS_BRANCH); \
		$(GIT) -C "$(DECKFONTS_DIR)" reset --hard origin/$(DECKFONTS_BRANCH); \
	else \
		$(GIT) clone --depth=$(DECKFONTS_DEPTH) --branch $(DECKFONTS_BRANCH) $(DECKFONTS_REPO) "$(DECKFONTS_DIR)"; \
	fi
	@echo "export DECKFONTS=$(DECKFONTS)"
	@echo "Add the line above to your shell profile if you want it permanently."

dubois:
	@if [ -d "$(DUBOIS_DIR)/.git" ]; then \
		$(GIT) -C "$(DUBOIS_DIR)" fetch --depth=$(DUBOIS_DEPTH) $(DUBOIS_FILTER) origin $(DUBOIS_BRANCH); \
		$(GIT) -C "$(DUBOIS_DIR)" checkout $(DUBOIS_BRANCH); \
		$(GIT) -C "$(DUBOIS_DIR)" reset --hard origin/$(DUBOIS_BRANCH); \
		if [ -n "$(strip $(DUBOIS_SPARSE))" ]; then \
			$(GIT) -C "$(DUBOIS_DIR)" sparse-checkout set $(DUBOIS_SPARSE); \
		fi; \
	else \
		$(GIT) clone --depth=$(DUBOIS_DEPTH) $(DUBOIS_FILTER) --branch $(DUBOIS_BRANCH) $(DUBOIS_REPO) "$(DUBOIS_DIR)"; \
		if [ -n "$(strip $(DUBOIS_SPARSE))" ]; then \
			$(GIT) -C "$(DUBOIS_DIR)" sparse-checkout init --cone; \
			$(GIT) -C "$(DUBOIS_DIR)" sparse-checkout set $(DUBOIS_SPARSE); \
		fi; \
	fi

examples: deckviz dubois
	@{ \
		find "$(DECKVIZ_DIR)" -mindepth 1 -maxdepth 1 -type d -exec sh -c 'printf "deckviz/%s\n" "$(basename "$1")"' _ {} \; ; \
		find "$(DUBOIS_DIR)" -mindepth 1 -maxdepth 1 -type d -exec sh -c 'printf "dubois/%s\n" "$(basename "$1")"' _ {} \; ; \
	} | sort

list-examples: examples

run-example: EXAMPLES := $(EXAMPLE)
run-example: run-examples

run-examples:
	@set -eu; \
	if [ "$(strip $(NO_SYNC))" != "1" ]; then \
		$(MAKE) install deckviz deckfonts dubois; \
	fi; \
	if [ -z "$(strip $(EXAMPLES))" ]; then \
		echo "No examples specified. Set EXAMPLES=\"fire flag\" or EXAMPLE=fire."; \
		exit 1; \
	fi; \
	for ex in $(EXAMPLES); do \
		src="deckviz"; \
		name="$$ex"; \
		case "$$ex" in \
			*/*) \
				src="$${ex%%/*}"; \
				name="$${ex#*/}"; \
			;; \
		esac; \
		case "$$src" in \
			deckviz) base="$(DECKVIZ_DIR)";; \
			dubois) base="$(DUBOIS_DIR)";; \
			*) echo "Skipping $$ex: unknown source '$$src'"; continue;; \
		esac; \
		dir="$$base/$$name"; \
		dsh="$$dir/$$name.dsh"; \
		xml="$$dir/$$name.xml"; \
		if [ ! -f "$$dsh" ]; then \
			echo "Skipping $$ex: expected $$dsh"; \
			continue; \
		fi; \
		echo "Linting $$dsh"; \
		( cd "$$dir" && DECKFONTS="$(DECKFONTS)" "$(DSHLINT_BIN)" "$$name.dsh" ); \
		echo "Rendering $$xml"; \
		( cd "$$dir" && DECKFONTS="$(DECKFONTS)" "$(DECKSH_BIN)" "$$name.dsh" > "$$name.xml" ); \
		echo "Wrote $$xml"; \
	done

view-example: run-example
	@set -eu; \
	ex="$(EXAMPLE)"; \
	src="deckviz"; \
	name="$$ex"; \
	case "$$ex" in \
		*/*) \
			src="$${ex%%/*}"; \
			name="$${ex#*/}"; \
		;; \
	esac; \
	case "$$src" in \
		deckviz) xml="$(DECKVIZ_DIR)/$$name/$$name.xml";; \
		dubois) xml="$(DUBOIS_DIR)/$$name/$$name.xml";; \
		*) echo "Unknown example source '$$src'"; exit 1;; \
	esac; \
	if [ ! -f "$$xml" ]; then \
		echo "Rendered deck not found at $$xml"; \
		exit 1; \
	fi; \
	"$(EBDECK_BIN)" "$$xml"

decktool: main.go go.mod go.sum
	$(GO) build -o decktool .

decktool-install:
	$(GO) install .

decktool-completions:
	@if [ -z "$(SHELL)" ]; then \
		echo "Set SHELL=bash|zsh|fish|powershell to choose a completion script." >&2; \
		exit 2; \
	fi; \
	case "$(SHELL)" in \
		bash|zsh|fish|powershell) ;; \
		*) echo "Unsupported shell: $(SHELL)" >&2; exit 2 ;; \
	esac; \
	$(GO) run . completion $(SHELL)
