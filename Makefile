#!/usr/bin/env bash

.PHONY: testnet1 testnet2 testnet3 testnet4

COMPANY=Smilo
AUTHOR=go-smilo

DIR = $(shell pwd)
PACKAGES = $(shell find ./src -type d -not -path '\./src')
PACKAGES_ETH = $(shell find src/blockchain/smilobft -type d -not -path '\src/blockchain/smilobft')

GOBIN = $(shell pwd)/build/bin
GO ?= latest

build: clean
	go build -o go-smilo main.go
	docker build --no-cache -t $(FULLDOCKERNAME) .

test: clean ## Run tests
	go test ./src/blockchain/... -timeout=10m

test-c: clean ## Run tests with coverage
	go test ./src/... -timeout=15m -cover

test-all: clean
	$(foreach pkg,$(PACKAGES),\
		go test $(pkg) -timeout=5m;)

test-race: clean ## Run tests with -race. Note: expected to fail, but look for "DATA RACE" failures specifically
	go test ./src/... -timeout=5m -race

lint: clean ## Run linters. Use make install-linters first.
	vendorcheck ./src/...
	gometalinter --deadline=3m -j 2 --disable-all --tests --vendor \
		-E deadcode \
		-E errcheck \
		-E gas \
		-E goconst \
		-E gofmt \
		-E goimports \
		-E golint \
		-E ineffassign \
		-E interfacer \
		-E maligned \
		-E megacheck \
		-E misspell \
		-E nakedret \
		-E structcheck \
		-E unconvert \
		-E unparam \
		-E varcheck \
		-E vet \
		./src/...


cover: ## Runs tests on ./src/ with HTML code coverage
	@echo "mode: count" > coverage-all.out
	$(foreach pkg,$(PACKAGES),\
		go test -coverprofile=coverage.out $(pkg);\
		tail -n +2 coverage.out >> coverage-all.out;)
	go tool cover -html=coverage-all.out

doc:
	godoc2md go-smilo/src/model > ./docs/model.md
	godoc2md go-smilo/src/server > ./docs/server.md
	$(foreach pkg,$(PACKAGES_ETH),\
	    rm -rf $(PWD)/docs/$(pkg); mkdir -p $(PWD)/docs/$(pkg); \
		godoc2md  go-smilo/$(pkg) > $(PWD)/docs/$(pkg).md;)

install-linters: ## Install linters
	go get -u github.com/FiloSottile/vendorcheck
	go get -u github.com/alecthomas/gometalinter
	go get -u github.com/davecheney/godoc2md
	gometalinter --vendored-linters --install


format:  # Formats the code. Must have goimports installed (use make install-linters).
	# This sorts imports by [stdlib, 3rdpart]
	$(foreach pkg,$(PACKAGES),\
		goimports -w -local go-smilo $(pkg);\
		gofmt -s -w $(pkg);)
	goimports -w -local go-smilo main.go
	gofmt -s -w main.go



# ********* BEGIN GETH BUILD TASKS *********

eth: clean
	src/blockchain/smilobft/build/env.sh go run ./src/blockchain/smilobft/build/ci.go install

test-eth: eth
	src/blockchain/smilobft/build/env.sh go run src/blockchain/smilobft/build/ci.go test

unlink:
	sudo unlink /usr/local/bin/geth || true

geth-link: unlink eth
	sudo ln -s  /opt/gocode/src/go-smilo/build/bin/geth /usr/local/bin/geth  || true

geth: eth
	@echo "Done building."
	@echo "Run \"$(GOBIN)/geth\" to launch geth."

swarm: clean
	src/blockchain/smilobft/build/env.sh go run src/blockchain/smilobft/build/ci.go install ./src/blockchain/smilobft/cmd/swarm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swarm\" to launch swarm."

android: clean
	src/blockchain/smilobft/build/env.sh go run src/blockchain/smilobft/build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/geth.aar\" to use the library."

ios: clean
	src/blockchain/smilobft/build/env.sh go run src/blockchain/smilobft/build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Geth.framework\" to use the library."

clean:
	rm -fr build/_workspace
	rm -fr build/bin

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/jteeuwen/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go install ./src/blockchain/smilobft/cmd/abigen
	env GOBIN= go get -u github.com/golang/protobuf/protoc-gen-go

generate:
	src/blockchain/smilobft/build/env.sh go generate go-smilo/src/blockchain/smilobft
	src/blockchain/smilobft/build/env.sh go generate go-smilo/src/blockchain/smilobft/internal/jsre/deps
	src/blockchain/smilobft/build/env.sh go generate ./src/blockchain/smilobft/...


# Cross Compilation Targets (xgo)

geth-cross: geth-linux geth-darwin geth-windows geth-android geth-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/geth-*

geth-linux: geth-linux-386 geth-linux-amd64 geth-linux-arm geth-linux-mips64 geth-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/geth-linux-*

geth-linux-386:
	src/blockchain/smilobft/build/env.sh go run src/blockchain/smilobft/build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/geth
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/geth-linux-* | grep 386

geth-linux-amd64:
	src/blockchain/smilobft/build/env.sh go run src/blockchain/smilobft/build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/geth
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/geth-linux-* | grep amd64

geth-linux-arm: geth-linux-arm-5 geth-linux-arm-6 geth-linux-arm-7 geth-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/geth-linux-* | grep arm

geth-linux-arm-5:
	src/blockchain/smilobft/build/env.sh go run src/blockchain/smilobft/build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/geth
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/geth-linux-* | grep arm-5

geth-linux-arm-6:
	src/blockchain/smilobft/build/env.sh go run src/blockchain/smilobft/build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/geth
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/geth-linux-* | grep arm-6

geth-linux-arm-7:
	src/blockchain/smilobft/build/env.sh go run src/blockchain/smilobft/build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/geth
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/geth-linux-* | grep arm-7

geth-linux-arm64:
	src/blockchain/smilobft/build/env.sh go run src/blockchain/smilobft/build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/geth
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/geth-linux-* | grep arm64

geth-linux-mips:
	src/blockchain/smilobft/build/env.sh go run src/blockchain/smilobft/build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/geth
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/geth-linux-* | grep mips

geth-linux-mipsle:
	src/blockchain/smilobft/build/env.sh go run src/blockchain/smilobft/build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/geth
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/geth-linux-* | grep mipsle

geth-linux-mips64:
	src/blockchain/smilobft/build/env.sh go run src/blockchain/smilobft/build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/geth
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/geth-linux-* | grep mips64

geth-linux-mips64le:
	src/blockchain/smilobft/build/env.sh go run src/blockchain/smilobft/build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/geth
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/geth-linux-* | grep mips64le

geth-darwin: geth-darwin-386 geth-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/geth-darwin-*

geth-darwin-386:
	src/blockchain/smilobft/build/env.sh go run src/blockchain/smilobft/build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/geth
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/geth-darwin-* | grep 386

geth-darwin-amd64:
	src/blockchain/smilobft/build/env.sh go run src/blockchain/smilobft/build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/geth
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/geth-darwin-* | grep amd64

geth-windows: geth-windows-386 geth-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/geth-windows-*

geth-windows-386:
	src/blockchain/smilobft/build/env.sh go run src/blockchain/smilobft/build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/geth
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/geth-windows-* | grep 386

geth-windows-amd64:
	src/blockchain/smilobft/build/env.sh go run src/blockchain/smilobft/build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/geth
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/geth-windows-* | grep amd64



# ********* END GETH BUILD TASKS *********




