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
	"enode://07f7317ac02bbdf04024ce8b832395bb372e7869b34648d4d2adb2f72c09b18b6e710039a6f85d12fefbb25467020cc5e0f8a6bfe1013e2c06dce169cad74c72@51.89.103.232:21000",
	"enode://2765f23ee14b18fb7af06117edc5d73e35cb937d062724e7313ca10004bb0ab2f078335933aa907fb160d81f3f85ffdeba502488a6936f6d60da0b8a1937bedc@51.89.103.233:21000",
	"enode://394f1c72f7049a419d432dab3755a804c7858cf99a5f8c63d9cb4b13c7d0c012450c6c8e3f241b2d2926179b74c24e3f67afeb6c52bbd1d586a9838a69062635@51.89.103.234:21000",
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
	"enode://179cb79e71c87b93a2bbe63b60ca9a3dc14dcbffcf50cf5a6cc32db538aa3660febea920423aa9c5041d8b040d11b6f779adfa82d07bcd5bfd78664ef000c822@94.23.218.88:21010",
	"enode://ead8adbf5665051890b6f56083ef2aab1af668125d94c6974836920c5c6a67c33ced4eb1a61b37d63328e6e7d242bac668a36eb5a779bcdcaa5a3ff455a47ce1@94.23.218.88:21011",
	"enode://c000fccd79cde43b0deb10a70af9fac7cbd6c95b5eadd77102bd1e1bac6e963df9b6df4b31d24a7e5561deb97c384e88dc656397f1f8def1c791632f38399b62@94.23.218.88:21012",
	"enode://cb5354cadffd4431cf471cac38132afdbc2c9ffee02002939563119c361ac7441f5ea66378ebb7fb53d2a5e32ae284fd510db5434fa4e7634bd20115e6450a90@94.23.218.88:21013",
	"enode://5558db3ef0061c34dfc86eefd7c6a86c0ce7a251dc1a06d51e449cfa6b9b0e2a5e651f60a09d03995f67535962eda95f8a897175ef02fd5f3d7871c0afdd42eb@94.23.218.88:21014",
}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
//TODO: config bootnodes
var DiscoveryV5Bootnodes = []string{}
