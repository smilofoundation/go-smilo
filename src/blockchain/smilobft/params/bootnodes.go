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
var MainnetBootnodes = []string{
	//TODO: config bootnodes
	// Ethereum Foundation Go Bootnodes
	//"enode://a979fb575495b8d6db44f750317d0f4622bf4c2aa3365d6af7c284339968eef29b69ad0dce72a4d8db5ebb4968de0e3bec910127f134779fbcb0cb6d3331163c@52.16.188.185:30303", // IE
	//"enode://3f1d12044546b76342d59d4a05532c14b85aa669704bfe1f864fe079415aa2c02d743e03218e57a33fb94523adb54032871a6c51b2cc5514cb7c7e35b3ed0a99@13.93.211.84:30303",  // US-WEST
	//"enode://78de8a0916848093c73790ead81d1928bec737d565119932b98c6b100d944b7a95e94f847f689fc723399d2e31129d182f7ef3863f2b4c820abbf3ab2722344d@191.235.84.50:30303", // BR
	//"enode://158f8aab45f6d19c6cbf4a089c2670541a8da11978a2f90dbf6a502a4a3bab80d288afdbeb7ec0ef6d92de563767f3b1ea9e8e334ca711e9f8e2df5a0385e8e6@13.75.154.138:30303", // AU
	//"enode://1118980bf48b0a3640bdba04e0fe78b1add18e1cd99bf22d53daac1fd9972ad650df52176e7c7d89d1114cfef2bc23a2959aa54998a46afcf7d91809f0855082@52.74.57.123:30303",  // SG

	// Ethereum Foundation C++ Bootnodes
	//"enode://979b7fa28feeb35a4741660a16076f1943202cb72b6af70d327f053e248bab9ba81760f39d0701ef1d8f89cc1fbd2cacba0710a12cd5314d5e0c9021aa3637f9@5.1.83.226:30303", // DE
}

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
var RinkebyBootnodes = []string{
	//TODO: config bootnodes
	//"enode://a24ac7c5484ef4ed0c5eb2d36620ba4e4aa13b8c84684e1b4aab0cebea2ae45cb4d375b77eab56516d34bfbd3c1a833fc51296ff084b770b94fb9028c4d25ccf@52.169.42.101:30303", // IE
	//"enode://343149e4feefa15d882d9fe4ac7d88f885bd05ebb735e547f12e12080a9fa07c8014ca6fd7f373123488102fe5e34111f8509cf0b7de3f5b44339c9f25e87cb8@52.3.158.184:30303",  // INFURA
	//"enode://b6b28890b006743680c52e64e0d16db57f28124885595fa03a562be1d2bf0f3a1da297d56b13da25fb992888fd556d4c1a27b1f39d531bde7de1921c90061cc6@159.89.28.211:30303", // AKASHA
}

// RinkebyV5Bootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Rinkeby test network for the experimental RLPx v5 topic-discovery network.
var RinkebyV5Bootnodes = []string{
	//TODO: config bootnodes
	//"enode://a24ac7c5484ef4ed0c5eb2d36620ba4e4aa13b8c84684e1b4aab0cebea2ae45cb4d375b77eab56516d34bfbd3c1a833fc51296ff084b770b94fb9028c4d25ccf@52.169.42.101:30303?discport=30304", // IE
	//"enode://343149e4feefa15d882d9fe4ac7d88f885bd05ebb735e547f12e12080a9fa07c8014ca6fd7f373123488102fe5e34111f8509cf0b7de3f5b44339c9f25e87cb8@52.3.158.184:30303?discport=30304",  // INFURA
}

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
var DiscoveryV5Bootnodes = []string{
	//TODO: config bootnodes
	//"enode://06051a5573c81934c9554ef2898eb13b33a34b94cf36b202b69fde139ca17a85051979867720d4bdae4323d4943ddf9aeeb6643633aa656e0be843659795007a@35.177.226.168:30303",
	//"enode://0cc5f5ffb5d9098c8b8c62325f3797f56509bff942704687b6530992ac706e2cb946b90a34f1f19548cd3c7baccbcaea354531e5983c7d1bc0dee16ce4b6440b@40.118.3.223:30304",
	//"enode://1c7a64d76c0334b0418c004af2f67c50e36a3be60b5e4790bdac0439d21603469a85fad36f2473c9a80eb043ae60936df905fa28f1ff614c3e5dc34f15dcd2dc@40.118.3.223:30306",
	//"enode://85c85d7143ae8bb96924f2b54f1b3e70d8c4d367af305325d30a61385a432f247d2c75c45c6b4a60335060d072d7f5b35dd1d4c45f76941f62a4f83b6e75daaf@40.118.3.223:30307",
}
