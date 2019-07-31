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
	"enode://e8f9d548513084c85333132244fb13e5bd1c14ee70fe5ef0a67f92be047d3044b0d4794d58fba1feda3335230a173ecbb2f6508536e21fa4a91e42fdd46e3fb4@51.89.103.235:21000",
	"enode://4269fd3500b8117993097391b88cc54d9456c2a9604f48629db8f7a3754ea51934d18c73d25b2841009597c82978f960cb2f21f20ddc0e89b7412e9c2f17f011@51.89.103.236:21000",
	"enode://d1998775a52286bfc919417294809c2abce817a2c9d6d056348cec09c525886beb73f9b1bed8acd281044c28e572195528fd2a6c34e368656d14eb8ccc9602c8@51.89.103.237:21000",
	"enode://ec970fee4629a28b0d4ee4b3df04db4ef6955170cfce974030112959b3da0902103fd0836839329bb804de397b39692bb5911c1c4d4335cce5595b199a25b9d3@51.89.103.238:21000",
}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
//TODO: config bootnodes
var DiscoveryV5Bootnodes = []string{}
