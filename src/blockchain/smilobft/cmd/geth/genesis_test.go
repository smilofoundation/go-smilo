// Copyright 2016 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

//1000000000000000000000000000 == 0x446c3b15f9926687d2c40534fdb564000000000000

var customGenesisTests = []struct {
	genesis string
	query   string
	result  string
}{
	// Plain genesis file without anything extra
	{
		genesis: `{
			"alloc"      : {},
			"coinbase"   : "0x0000000000000000000000000000000000000000",
			"difficulty" : "0x20000",
			"extraData"  : "",
			"gasLimit"   : "0x2fefd8",
			"nonce"      : "0x0000000000000042",
			"mixhash"    : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"timestamp"  : "0x00",
			"config"     : {"isSmilo":false}
		}`,
		query:  "eth.getBlock(0).nonce",
		result: "0x0000000000000042",
	},
	// Genesis file with an empty chain configuration (ensure missing fields work)
	{
		genesis: `{
			"alloc"      : {},
			"coinbase"   : "0x0000000000000000000000000000000000000000",
			"difficulty" : "0x20000",
			"extraData"  : "",
			"gasLimit"   : "0x2fefd8",
			"nonce"      : "0x0000000000000042",
			"mixhash"    : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"timestamp"  : "0x00",
			"config"     : {"isSmilo":false }
		}`,
		query:  "eth.getBlock(0).nonce",
		result: "0x0000000000000042",
	},
	// Genesis file with specific chain configurations
	{
		genesis: `{
			"alloc"      : {},
			"coinbase"   : "0x0000000000000000000000000000000000000000",
			"difficulty" : "0x20000",
			"extraData"  : "",
			"gasLimit"   : "0x2fefd8",
			"nonce"      : "0x0000000000000042",
			"mixhash"    : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"timestamp"  : "0x00",
			"config"     : {
				"homesteadBlock" : 314,
				"daoForkBlock"   : 141,
				"daoForkSupport" : true,
				"isSmilo" : false
	
			}
		}`,
		query:  "eth.getBlock(0).nonce",
		result: "0x0000000000000042",
	},
	{
		genesis: `{
  "alloc": {
    "ecf7e57d01d3d155e5fc33dbc7a58355685ba39c": {
      "balance": "0x446c3b15f9926687d2c40534fdb564000000000000"
    },
    "c0ce2fd65f71c6ce82d22db11fcf7ca43357f172": {
      "balance": "0x446c3b15f9926687d2c40534fdb564000000000000"
    },
    "7cb791430d2461268691bfba6e35d8a8c7ea2e63": {
      "balance": "0x446c3b15f9926687d2c40534fdb564000000000000"
    },
    "d54924701cd0d94d677d0a66dee75c978e175c74": {
      "balance": "0x446c3b15f9926687d2c40534fdb564000000000000"
    },
    "2f65a895741143953aabed3680177594818a5f9a": {
      "balance": "0x446c3b15f9926687d2c40534fdb564000000000000"
    },
    "497c8fe926bc88b61e736afe7aae2ea21414671f": {
      "balance": "0x446c3b15f9926687d2c40534fdb564000000000000"
    },
    "0fbc07ebdce2bfead66f1686d67f9ea5c759e433": {
      "balance": "0x446c3b15f9926687d2c40534fdb564000000000000" 
    }
  },
  "coinbase": "0x0000000000000000000000000000000000000000",
  "config": {
    "byzantiumBlock": 1,
    "eip150Block": 2,
    "eip150Hash": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "eip155Block": 0,
    "eip158Block": 3,
    "sport": {
      "epoch": 30000,
      "policy": 0
    },
    "isSmilo": true,
    "chainId": 10
  },
  "extraData": "0x0000000000000000000000000000000000000000000000000000000000000000f8d9f89394ecf7e57d01d3d155e5fc33dbc7a58355685ba39c94c0ce2fd65f71c6ce82d22db11fcf7ca43357f172947cb791430d2461268691bfba6e35d8a8c7ea2e6394d54924701cd0d94d677d0a66dee75c978e175c74942f65a895741143953aabed3680177594818a5f9a94497c8fe926bc88b61e736afe7aae2ea21414671f940fbc07ebdce2bfead66f1686d67f9ea5c759e433b8410000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0",
  "gasLimit": "0xE0000000",
  "difficulty": "0x1",
  "mixHash": "0x63746963616c2062797a616e74696e65206661756c7420746f6c6572616e6365",
  "nonce": "0x0000000000000042",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "timestamp": "0x00"
}`,
		query:  "eth.getBlock(0).nonce",
		result: "0x0000000000000042",
	},
}

// Tests that initializing Geth with a custom genesis block and chain definitions
// work properly.
func TestCustomGenesis(t *testing.T) {
	for i, tt := range customGenesisTests {
		// Create a temporary data directory to use and inspect later
		datadir := tmpdir(t)
		defer os.RemoveAll(datadir)

		// Initialize the data directory with the custom genesis block
		json := filepath.Join(datadir, "genesis.json")
		if err := ioutil.WriteFile(json, []byte(tt.genesis), 0600); err != nil {
			t.Fatalf("test %d: failed to write genesis file: %v", i, err)
		}
		runGeth(t, "--datadir", datadir, "init", json).WaitExit()

		// Query the custom genesis block
		geth := runGeth(t,
			"--datadir", datadir, "--maxpeers", "0", "--port", "0",
			"--nodiscover", "--nat", "none", "--ipcdisable",
			"--exec", tt.query, "console")
		geth.ExpectRegexp(tt.result)
		geth.ExpectExit()
	}
}
