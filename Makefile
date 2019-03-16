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
GO ?= 1.11

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

lint-eth: clean
	src/blockchain/smilobft/build/env.sh go run ./src/blockchain/smilobft/build/ci.go lint

imports:
	./src/blockchain/smilobft/build/goimports.sh

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
	go get -u gopkg.in/alecthomas/gometalinter.v2
	go get -u golang.org/x/tools/cmd/goimports
	go get -u golang.org/x/tools/cmd/gofmt
#	go get -u github.com/davecheney/godoc2md
	gometalinter --vendored-linters --install


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
	env GOBIN= go get -u github.com/kevinburke/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go get -u github.com/golang/protobuf/protoc-gen-go
	env GOBIN= go install ./src/blockchain/smilobft/cmd/abigen
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm'
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

swarm-devtools:
	env GOBIN= go install ./cmd/swarm/mimegen

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
#	$(SRC_DIR)/build/env.sh go run $(SRC_DIR)/build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./$(SRC_DIR)/cmd/geth
#	@echo "Darwin 386 cross compilation done:"
#	@ls -ld $(GOBIN)/geth-darwin-* | grep 386

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




geth-local1: #geth
	rm -rf /opt/gocode/src/smilo-testnet/aws5/sdata/ss10
	./build/bin/geth --datadir /opt/gocode/src/smilo-testnet/aws5/sdata/ss10 init /opt/gocode/src/smilo-testnet/aws5/smilo-genesis-mainnet.json
	./build/bin/geth --datadir /opt/gocode/src/smilo-testnet/aws5/sdata/ss10 --verbosity 5 \
	--syncmode full --networkid 20080914 \
	--rpc --rpcaddr 0.0.0.0 --rpcapi admin,db,eth,debug,miner,net,shh,txpool,personal,web3,smilo,sport \
	--rpccorsdomain \"*\" --ws --wsaddr 0.0.0.0 --wsorigins '*' --wsapi personal,admin,db,eth,net,web3,miner,shh,txpool,debug \
	--rpcport 22009 --wsport 23009 --port 21009

#	--sport \
#	./build/bin/geth --datadir /opt/gocode/src/smilo-testnet/aws5/sdata/ss9 init /opt/gocode/src/github.com/ethereum/smilo-examples/examples/7smilos/sport-genesis-v1-8.json
geth-local: #geth
#	rm -rf temp/sdata/ss9
#	./build/bin/geth --datadir /opt/gocode/src/smilo-testnet/aws5/sdata/ss9 init /opt/gocode/src/smilo-testnet/aws5/smilo-genesis-mainnet.json
	./build/bin/geth --datadir temp/sdata/ss9 --verbosity 2 \
	--syncmode full --networkid 10 \
	--rpc --rpcaddr 0.0.0.0 --rpcapi admin,db,eth,debug,miner,net,shh,txpool,personal,web3,smilo,sport \
	--rpccorsdomain \"*\" --ws --wsaddr 0.0.0.0 --wsorigins '*' --wsapi personal,admin,db,eth,net,web3,miner,shh,txpool,debug \
	--rpcport 22008 --wsport 23008 --port 21008 --testnet
#	-bootnodes enode://8f9d91c3d9ee8f0676e5da062f2bc7a6ae00f962d65897e2b9c64f3338bcb7f76f7f802414f55f20e929cacc65f9ea466ddc05856bd22cfe975dad16ba235738@127.0.0.1:21009
#	-bootnodes enode://b1ff9da9bd6f135a852625793235a24333ece79047a952ebcea8cf464768b506a4a4e0897af13e0641bfd5e2a120bce89d200e8e20c81d170533210e91907387@127.0.0.1:21000
#	-bootnodes enode://35beee3c86cb3e4d25009779b25ed2f964f31b0f160766a5a53c2ca8c0d705d827a35dde71d479ac0c7a954f7cec8f1a501062bf132ae735aa9569ec98180cef@52.214.227.187:30301,enode://aa255d87e4f7586332b8a2cb2b39a4572029c34b109874cc4819ba19a7b7a3b50ad5ebb0e9af05f26594cd89c66e11f67e49e280ae2318125c9b298ff4d36f24@52.50.18.20:30301,enode://db485ce2629952c2d213930bacfdb8ab0f51b55a103dd9e6350d079ea26cf03b452613c74ac264c4652aa2df1c721f4dad0e9da0e556e416c022afd7c8526520@34.252.54.93:30301,enode://dcfb91c1d54eacee2e605f0deb6296d97faf2a5a4284f4e476e9c5dfd9c28db698eaeeddab07247a24f2483e9687865d6fae21b6c8127b5639b42f4ba36c4c93@34.252.54.93:30301,enode://06c06c0d7273e0886fe56f98e70686a8490e636190d2655fdc3da9838eb2eca3f7d760f2c19bb1c85c97727cf514a5b5ff0c7aa4f13529e2d5f7b69c726cb73d@52.212.79.188:30301
#	-bootnodes enode://f5cce0c7413240ade5f0a37a052383427f54b285df63c41d19694a7be54b0ef5ea14dd1947b1b96c10f2324c6dd48bcfc3480c423aa3bc12e560e0005fddeb3e@18.202.153.27:21000
