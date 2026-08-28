# Iteration loop first: `make dev` must stay under a second, because build
# speed sets the ceiling on how fast this project can be developed.

GO      ?= go
BIN     := bin
DEVROOT := .devstate

.PHONY: build
build:
	@$(GO) build -o $(BIN)/lab  ./cmd/lab
	@$(GO) build -o $(BIN)/labd ./cmd/labd

.PHONY: test
test:
	@$(GO) test ./...

.PHONY: vet
vet:
	@$(GO) vet ./...

.PHONY: check
check: vet test

# dev runs against a scratch socket, journal, and the in-tree topologies, so
# it never touches real state. Point your shell at it with:
#   eval $(make devenv)
.PHONY: dev
dev: build
	@mkdir -p $(DEVROOT)
	@LAB_SOCKET=$(CURDIR)/$(DEVROOT)/lab.sock \
	 LAB_STATE_DIR=$(CURDIR)/$(DEVROOT) \
	 LAB_TOPO_DIR=$(CURDIR)/topologies \
	 LAB_DRIVER=mock \
	 $(BIN)/labd -idle 10m

.PHONY: devenv
devenv:
	@echo 'export LAB_SOCKET=$(CURDIR)/$(DEVROOT)/lab.sock'
	@echo 'export LAB_STATE_DIR=$(CURDIR)/$(DEVROOT)'
	@echo 'export LAB_TOPO_DIR=$(CURDIR)/topologies'
	@echo 'export LAB_DRIVER=mock'
	@echo 'export PATH=$(CURDIR)/$(BIN):$$PATH'

.PHONY: clean
clean:
	@rm -rf $(BIN) $(DEVROOT)
