// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package params

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main Ethereum network.
//TODO: config bootnodes
// Ethereum Foundation Go Bootnodes
var MainnetBootnodes = []string{}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Ropsten test network.
var TestnetBootnodes = []string{
	//TODO: config bootnodes
	"enode://b1ff9da9bd6f135a852625793235a24333ece79047a952ebcea8cf464768b506a4a4e0897af13e0641bfd5e2a120bce89d200e8e20c81d170533210e91907387@34.244.254.171:21000",
	"enode://c2b45040e3a23ca91a1adb3eeacbd3dbe4d64d0b11abbc2e87686786797f77dedc79f2d43211b8321c0f2a9f145dc9de387cf155020ba02c645311350a9ade4c@52.214.105.103:21000",
	"enode://1deebc4d4034a6e2e156fe4f81a5691416a9caca25a73a00b8cca1569d3283baa33cf3fcb159df001916c86f1f78d67814137bf3b9f0fa1293fefb219ae3cf35@34.247.187.251:21000",
	"enode://239c00712351fa83612ec52970432bbff81c119c7d64b2a35dfa21f63cb38f2d36ffadfe7331cbfb32275b27456c41818a67979b379b8003685175f9bb05b1fa@34.250.30.134:21000",
}

// RinkebyBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Rinkeby test network.
//TODO: config bootnodes
var RinkebyBootnodes = []string{}

// RinkebyV5Bootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Rinkeby test network for the experimental RLPx v5 topic-discovery network.
//TODO: config bootnodes
var RinkebyV5Bootnodes = []string{}

// Sport are the enode URLs of the P2P bootstrap nodes running on the SPORT consensus mainnet.
var SportBootnodes = []string{
	// Smilo Foundation Go Bootnodes
	"enode://35beee3c86cb3e4d25009779b25ed2f964f31b0f160766a5a53c2ca8c0d705d827a35dde71d479ac0c7a954f7cec8f1a501062bf132ae735aa9569ec98180cef@52.214.227.187:30301",
	"enode://aa255d87e4f7586332b8a2cb2b39a4572029c34b109874cc4819ba19a7b7a3b50ad5ebb0e9af05f26594cd89c66e11f67e49e280ae2318125c9b298ff4d36f24@52.50.18.20:30301",
	"enode://db485ce2629952c2d213930bacfdb8ab0f51b55a103dd9e6350d079ea26cf03b452613c74ac264c4652aa2df1c721f4dad0e9da0e556e416c022afd7c8526520@34.252.54.93:30301",
	"enode://dcfb91c1d54eacee2e605f0deb6296d97faf2a5a4284f4e476e9c5dfd9c28db698eaeeddab07247a24f2483e9687865d6fae21b6c8127b5639b42f4ba36c4c93@34.252.54.93:30301",
	"enode://06c06c0d7273e0886fe56f98e70686a8490e636190d2655fdc3da9838eb2eca3f7d760f2c19bb1c85c97727cf514a5b5ff0c7aa4f13529e2d5f7b69c726cb73d@52.212.79.188:30301",
}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
//TODO: config bootnodes
var DiscoveryV5Bootnodes = []string{}
