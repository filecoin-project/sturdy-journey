SHELL=/usr/bin/env bash

unexport GOFLAGS

GOCC?=go

ldflags=-X=github.com/filecoin-project/sturdy-journey/build.CurrentCommit=+git.$(subst -,.,$(shell git describe --always --match=NeVeRmAtCh --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null))

ifneq ($(strip $(LDFLAGS)),)
	ldflags+=-extldflags=$(LDFLAGS)
endif

GOFLAGS+=-ldflags="$(ldflags)"

BINS:=

all: sturdy-journey
.PHONY: all

clean:
	rm -rf $(BINS)
.PHONY: clean

sturdy-journey:
	$(GOCC) build $(GOFLAGS) -o sturdy-journey ./cmd/sturdy-journey/
.PHONY: sturdy-journey
BINS+=sturdy-journey
