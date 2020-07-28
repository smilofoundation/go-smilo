#!/usr/bin/env bash

.PHONY: geth android ios geth-cross swarm evm all test clean
.PHONY: geth-linux geth-linux-386 geth-linux-amd64 geth-linux-mips64 geth-linux-mips64le
.PHONY: geth-linux-arm geth-linux-arm-5 geth-linux-arm-6 geth-linux-arm-7 geth-linux-arm64
.PHONY: geth-darwin geth-darwin-386 geth-darwin-amd64
.PHONY: geth-windows geth-windows-386 geth-windows-amd64


COMPANY=Smilo
AUTHOR=go-smilo

DIR = $(shell pwd)
PACKAGES = $(shell find ./src -type d -not -path '\./src')
PACKAGES_ETH = $(shell find src/blockchain/smilobft -type d -not -path '\src/blockchain/smilobft')

SRC_DIR = "src/blockchain/smilobft"


GOBIN = $(shell pwd)/build/bin
GO ?= 1.12

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

lint: lint-eth ## Run linters

lint-eth: clean
	src/blockchain/smilobft/build/env.sh go run ./src/blockchain/smilobft/build/ci.go lint

imports:
	./src/blockchain/smilobft/build/goimports.sh

cover: ## Runs tests on ./src/ with HTML code coverage
	@echo "mode: count" > coverage-all.out
	$(foreach pkg,$(PACKAGES),\
		go test -coverprofile=coverage.out $(pkg);\
		tail -n +2 coverage.out >> coverage-all.out;)

coveralls: ## Runs tests on ./src/ with HTML code coverage
	go get github.com/mattn/goveralls
	@echo "mode: count" > coverage-all.out
	$(foreach pkg,$(PACKAGES),\
		go test -covermode=count -coverprofile=coverage.out $(pkg);\
		tail -n +2 coverage.out >> coverage-all.out;)
	$(GOPATH)/bin/goveralls -coverprofile=coverage-all.out -service=travis-ci -repotoken $(COVERALLS_TOKEN)

doc:
	godoc2md go-smilo/src/model > ./docs/model.md
	godoc2md go-smilo/src/server > ./docs/server.md
	$(foreach pkg,$(PACKAGES_ETH),\
	    rm -rf $(PWD)/docs/$(pkg); mkdir -p $(PWD)/docs/$(pkg); \
		godoc2md  go-smilo/$(pkg) > $(PWD)/docs/$(pkg).md;)

install-linters: ## Install linters
	go get -u github.com/FiloSottile/vendorcheck
	go get -u gopkg.in/alecthomas/gometalinter.v2
	go get -u golang.org/x/tools/cmd/goimports
	go get -u github.com/mattn/goveralls


format:  # Formats the code. Must have goimports installed (use make install-linters).
	# This sorts imports by [stdlib, 3rdpart]
	$(foreach pkg,$(PACKAGES),\
		goimports -w -local go-smilo $(pkg);\
		gofmt -s -w $(pkg);)
	gofmt -s -w main.go
	goimports -w -local go-smilo main.go



# ********* BEGIN GETH BUILD TASKS *********

all:
	src/blockchain/smilobft/build/env.sh go run ./src/blockchain/smilobft/build/ci.go install

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

android:
#	export ANDROID_NDK_HOME=~/Downloads/android-studio/plugins/android-ndk/ # or replace it with your NDK_HOME, see: https://developer.android.com/ndk/guides/index.html
	src/blockchain/smilobft/build/env.sh go run src/blockchain/smilobft/build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/geth.aar\" to use the library."

ios:
	src/blockchain/smilobft/build/env.sh go run src/blockchain/smilobft/build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Geth.framework\" to use the library."

clean:
	rm -fr build/_workspace
	rm -fr build/bin

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u github.com/golang/dep/cmd/dep
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/kevinburke/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go get -u github.com/golang/protobuf/protoc-gen-go
	env GOBIN= go install ./src/blockchain/smilobft/cmd/abigen
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm'
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

generate:
	src/blockchain/smilobft/build/env.sh go generate go-smilo/src/blockchain/smilobft
	src/blockchain/smilobft/build/env.sh go generate go-smilo/src/blockchain/smilobft/internal/jsre/deps
	src/blockchain/smilobft/build/env.sh go generate ./src/blockchain/smilobft/eth/config.go
	src/blockchain/smilobft/build/env.sh go generate ./src/blockchain/smilobft/eth/tracers/tracers.go
	src/blockchain/smilobft/build/env.sh go generate ./src/blockchain/smilobft/permission/contract/gen/gen.go
	src/blockchain/smilobft/build/env.sh go generate ./src/blockchain/smilobft/...


# Cross Compilation Targets (xgo)
geth-cross: geth-linux geth-darwin android ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/geth-*

geth-linux: geth-linux-386 geth-linux-amd64 geth-linux-arm geth-linux-mips64 geth-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/geth-linux-*

geth-linux-386:
	$(SRC_DIR)/build/env.sh go run $(SRC_DIR)/build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./$(SRC_DIR)/cmd/geth
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/geth-linux-* | grep 386

geth-linux-amd64:
	$(SRC_DIR)/build/env.sh go run $(SRC_DIR)/build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./$(SRC_DIR)/cmd/geth
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/geth-linux-* | grep amd64

geth-linux-arm: geth-linux-arm-5 geth-linux-arm-6 geth-linux-arm-7 geth-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/geth-linux-* | grep arm

geth-linux-arm-5:
	$(SRC_DIR)/build/env.sh go run $(SRC_DIR)/build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./$(SRC_DIR)/cmd/geth
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/geth-linux-* | grep arm-5

geth-linux-arm-6:
	$(SRC_DIR)/build/env.sh go run $(SRC_DIR)/build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./$(SRC_DIR)/cmd/geth
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/geth-linux-* | grep arm-6

geth-linux-arm-7:
	$(SRC_DIR)/build/env.sh go run $(SRC_DIR)/build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./$(SRC_DIR)/cmd/geth
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/geth-linux-* | grep arm-7

geth-linux-arm64:
	$(SRC_DIR)/build/env.sh go run $(SRC_DIR)/build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./$(SRC_DIR)/cmd/geth
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/geth-linux-* | grep arm64

geth-linux-mips:
	$(SRC_DIR)/build/env.sh go run $(SRC_DIR)/build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./$(SRC_DIR)/cmd/geth
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/geth-linux-* | grep mips

geth-linux-mipsle:
	$(SRC_DIR)/build/env.sh go run $(SRC_DIR)/build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./$(SRC_DIR)/cmd/geth
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/geth-linux-* | grep mipsle

geth-linux-mips64:
	$(SRC_DIR)/build/env.sh go run $(SRC_DIR)/build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./$(SRC_DIR)/cmd/geth
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/geth-linux-* | grep mips64

geth-linux-mips64le:
	$(SRC_DIR)/build/env.sh go run $(SRC_DIR)/build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./$(SRC_DIR)/cmd/geth
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/geth-linux-* | grep mips64le

geth-darwin: geth-darwin-386 geth-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/geth-darwin-*

geth-darwin-386:
	$(SRC_DIR)/build/env.sh go run $(SRC_DIR)/build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./$(SRC_DIR)/cmd/geth
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/geth-darwin-* | grep 386

geth-darwin-amd64:
	$(SRC_DIR)/build/env.sh go run $(SRC_DIR)/build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./$(SRC_DIR)/cmd/geth
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/geth-darwin-* | grep amd64

geth-windows: geth-windows-386 geth-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/geth-windows-*

geth-windows-386:
	$(SRC_DIR)/build/env.sh go run $(SRC_DIR)/build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./$(SRC_DIR)/cmd/geth
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/geth-windows-* | grep 386

geth-windows-amd64:
	$(SRC_DIR)/build/env.sh go run $(SRC_DIR)/build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./$(SRC_DIR)/cmd/geth
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/geth-windows-* | grep amd64




# ********* END GETH BUILD TASKS *********






mockgen:
	mockgen -source=src/blockchain/smilobft/consensus/tendermint/core/core_backend.go -destination=src/blockchain/smilobft/consensus/tendermint/core/backend_mock.go
	mockgen -source=src/blockchain/smilobft/consensus/tendermint/validator/validator_interface.go -destination=src/blockchain/smilobft/consensus/tendermint/validator/validator_mock.go
