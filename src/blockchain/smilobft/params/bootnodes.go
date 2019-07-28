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
	//"enode://b1ff9da9bd6f135a852625793235a24333ece79047a952ebcea8cf464768b506a4a4e0897af13e0641bfd5e2a120bce89d200e8e20c81d170533210e91907387@52.214.8.213:21000",
	//"enode://c2b45040e3a23ca91a1adb3eeacbd3dbe4d64d0b11abbc2e87686786797f77dedc79f2d43211b8321c0f2a9f145dc9de387cf155020ba02c645311350a9ade4c@52.214.105.103:21000",
	//"enode://1deebc4d4034a6e2e156fe4f81a5691416a9caca25a73a00b8cca1569d3283baa33cf3fcb159df001916c86f1f78d67814137bf3b9f0fa1293fefb219ae3cf35@34.247.187.251:21000",
	//"enode://239c00712351fa83612ec52970432bbff81c119c7d64b2a35dfa21f63cb38f2d36ffadfe7331cbfb32275b27456c41818a67979b379b8003685175f9bb05b1fa@34.250.30.134:21000",
	//"enode://afbb873d8453ee11b7815453b964a16a39fcfa9a43f1932f06d34c3c170ef39b008883634c2d33dd2790fd412acd538664ab2f6f44316ed39a887bc62696cd29@34.242.23.51:21000",
	//"enode://d190d339aac119eb4a361b3da09c1ce38eb7f89f4c2bccc8582a1710d19b237a72a046443137014d3b81a92ac776a363a6ccf210adb160a0011cfd0371e470d5@54.194.85.174:21000",
	//"enode://8f525069ddf324739614b9b374d0282572c8db04b23b09f654a8484e0d58a3cf63e40c7dc0f34bb5cf45de7a132589fbc1d712cfa455da49992c319f8936b66f@34.244.157.19:21000",
	//"enode://06ee5987c1a65f6dcaf389ae60e26451c17bf9da2bb92d0d5c2016167439ecadad851b089ceae32609a31c20233369d9140e7e3560a145e85cc3df226cc2a5a0@34.249.150.234:21000",
	//"enode://2cfc0edf2b4e8fb99f5fa847c00c7bee60530232ec17a36005cfbe006b2965f0842cae1e8abc58acb834ee83570f8c07cd12525ca6f7cd128b1ed4397f6df256@34.255.253.175:21000",
	//"enode://ceed166da2243d8c06d8efa1eabcf37ba052c67b847da353ef29eba71bb7d43b67d0238f1a108b695cc589675c6d393ca0a81b6c8c64e9e974e55c077dee110a@52.51.176.157:21000",
	//"enode://8a2df21b3f41f4d8019b156f95b3ca89a788449b56a213d215a017e219c91f7b8a4ce2075d93a775700de3220180d4eca6bd985cdad5d4e946b94d1b3b476d9f@99.80.135.21:21000",
	//"enode://ce4face35274714f517b74064f165268c9f0ccc97bd26b8591ca4808323639cd8530bc34c2a1ca716e49ec51e27604217538d5443a416339294833d493c13acb@99.80.216.120:21000",
	//"enode://dfb829bdd8301f4b52026430f39ee46500f8010a4bbd256d93ad84bd2087d4f9d23b8c58ff2d6c7ca71114e591352bcac68a3014ed0753a001e370b64763541f@99.80.8.91:21000",
	//"enode://5011281210625d1936cbbba50b35bbf218f2bfdf47bcfb7aa21195240c8186db1d2fd1d2f06cd77209f66963cb0c09d1b1559a4654e8e6b0ee3cee7ea8c761e8@99.81.26.104:21000",
}

// RinkebyBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Rinkeby test network.
//TODO: config bootnodes
var RinkebyBootnodes = []string{}

// GoerliBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Görli test network.
var GoerliBootnodes = []string{}

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
