// Copyright 2019 The go-smilo Authors
// Copyright 2017 The go-ethereum Authors
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

package backend

import (
	"bytes"
	"crypto/ecdsa"
	"go-smilo/src/blockchain/smilobft/params"
	"math/big"
	"reflect"
	"testing"

	"go-smilo/src/blockchain/smilobft/core/rawdb"

	"github.com/ethereum/go-ethereum/crypto"

	"go-smilo/src/blockchain/smilobft/cmn"

	"github.com/ethereum/go-ethereum/common"

	"go-smilo/src/blockchain/smilobft/consensus/sport"
	"go-smilo/src/blockchain/smilobft/consensus/sport/fullnode"
	"go-smilo/src/blockchain/smilobft/core"
	"go-smilo/src/blockchain/smilobft/core/types"
	"go-smilo/src/blockchain/smilobft/core/vm"
)

type testerVote struct {
	fullnode string
	voted    string
	auth     bool
}

// testerAccountPool is a pool to maintain currently active tester accounts,
// mapped from textual names used in the tests below to actual Ethereum private
// keys capable of signing transactions.
type testerAccountPool struct {
	accounts map[string]*ecdsa.PrivateKey
}

func newTesterAccountPool() *testerAccountPool {
	return &testerAccountPool{
		accounts: make(map[string]*ecdsa.PrivateKey),
	}
}

func (ap *testerAccountPool) sign(header *types.Header, validator string) {
	// Ensure we have a persistent key for the validator
	if ap.accounts[validator] == nil {
		ap.accounts[validator], _ = crypto.GenerateKey()
	}
	// Sign the header and embed the signature in extra data
	hashData := crypto.Keccak256([]byte(sigHash(header).Bytes()))
	sig, _ := crypto.Sign(hashData, ap.accounts[validator])

	writeSeal(header, sig)
}

func (ap *testerAccountPool) address(account string) common.Address {
	// Ensure we have a persistent key for the account
	if ap.accounts[account] == nil {
		ap.accounts[account], _ = crypto.GenerateKey()
	}
	// Resolve and return the Ethereum address
	return crypto.PubkeyToAddress(ap.accounts[account].PublicKey)
}

// Tests that voting is evaluated correctly for various simple and complex scenarios.
func TestVoting(t *testing.T) {
	// Define the various voting scenarios to test
	tests := []struct {
		name       string
		epoch      uint64
		validators []string
		votes      []testerVote
		results    []string
	}{
		{
			name:       "Single fullnode, no votes cast",
			validators: []string{"A"},
			votes:      []testerVote{{fullnode: "A"}},
			results:    []string{"A"},
		}, {
			name:       "Single fullnode, voting to add two others (only accept first, second needs 2 votes)",
			validators: []string{"A"},
			votes: []testerVote{
				{fullnode: "A", voted: "B", auth: true},
				{fullnode: "B"},
				{fullnode: "A", voted: "C", auth: true},
			},
			results: []string{"A", "B"},
		}, {
			name:       "Two validators, voting to add three others (only accept first two, third needs 3 votes already)",
			validators: []string{"A", "B"},
			votes: []testerVote{
				{fullnode: "A", voted: "C", auth: true},
				{fullnode: "B", voted: "C", auth: true},
				{fullnode: "A", voted: "D", auth: true},
				{fullnode: "B", voted: "D", auth: true},
				{fullnode: "C"},
				{fullnode: "A", voted: "E", auth: true},
				{fullnode: "B", voted: "E", auth: true},
			},
			results: []string{"A", "B", "C", "D"},
		}, {
			name:       "Single fullnode, dropping itself (weird, but one less cornercase by explicitly allowing this)",
			validators: []string{"A"},
			votes: []testerVote{
				{fullnode: "A", voted: "A", auth: false},
			},
			results: []string{},
		}, {
			name:       "Two validators, actually needing mutual consent to drop either of them (not fulfilled)",
			validators: []string{"A", "B"},
			votes: []testerVote{
				{fullnode: "A", voted: "B", auth: false},
			},
			results: []string{"A", "B"},
		}, {
			name:       "Two validators, actually needing mutual consent to drop either of them (fulfilled)",
			validators: []string{"A", "B"},
			votes: []testerVote{
				{fullnode: "A", voted: "B", auth: false},
				{fullnode: "B", voted: "B", auth: false},
			},
			results: []string{"A"},
		}, {
			name:       "Three validators, two of them deciding to drop the third",
			validators: []string{"A", "B", "C"},
			votes: []testerVote{
				{fullnode: "A", voted: "C", auth: false},
				{fullnode: "B", voted: "C", auth: false},
			},
			results: []string{"A", "B"},
		}, {
			name:       "Four validators, consensus of two not being enough to drop anyone",
			validators: []string{"A", "B", "C", "D"},
			votes: []testerVote{
				{fullnode: "A", voted: "C", auth: false},
				{fullnode: "B", voted: "C", auth: false},
			},
			results: []string{"A", "B", "C", "D"},
		}, {
			name:       "Four validators, consensus of three already being enough to drop someone",
			validators: []string{"A", "B", "C", "D"},
			votes: []testerVote{
				{fullnode: "A", voted: "D", auth: false},
				{fullnode: "B", voted: "D", auth: false},
				{fullnode: "C", voted: "D", auth: false},
			},
			results: []string{"A", "B", "C"},
		}, {
			name:       "Authorizations are counted once per fullnode per target",
			validators: []string{"A", "B"},
			votes: []testerVote{
				{fullnode: "A", voted: "C", auth: true},
				{fullnode: "B"},
				{fullnode: "A", voted: "C", auth: true},
				{fullnode: "B"},
				{fullnode: "A", voted: "C", auth: true},
			},
			results: []string{"A", "B"},
		}, {
			name:       "Authorizing multiple accounts concurrently is permitted",
			validators: []string{"A", "B"},
			votes: []testerVote{
				{fullnode: "A", voted: "C", auth: true},
				{fullnode: "B"},
				{fullnode: "A", voted: "D", auth: true},
				{fullnode: "B"},
				{fullnode: "A"},
				{fullnode: "B", voted: "D", auth: true},
				{fullnode: "A"},
				{fullnode: "B", voted: "C", auth: true},
			},
			results: []string{"A", "B", "C", "D"},
		}, {
			name:       "Deauthorizations are counted once per fullnode per target",
			validators: []string{"A", "B"},
			votes: []testerVote{
				{fullnode: "A", voted: "B", auth: false},
				{fullnode: "B"},
				{fullnode: "A", voted: "B", auth: false},
				{fullnode: "B"},
				{fullnode: "A", voted: "B", auth: false},
			},
			results: []string{"A", "B"},
		}, {
			name:       "Deauthorizing multiple accounts concurrently is permitted",
			validators: []string{"A", "B", "C", "D"},
			votes: []testerVote{
				{fullnode: "A", voted: "C", auth: false},
				{fullnode: "B"},
				{fullnode: "C"},
				{fullnode: "A", voted: "D", auth: false},
				{fullnode: "B"},
				{fullnode: "C"},
				{fullnode: "A"},
				{fullnode: "B", voted: "D", auth: false},
				{fullnode: "C", voted: "D", auth: false},
				{fullnode: "A"},
				{fullnode: "B", voted: "C", auth: false},
			},
			results: []string{"A", "B"},
		}, {
			name:       "Votes from deauthorized validators are discarded immediately (deauth votes)",
			validators: []string{"A", "B", "C"},
			votes: []testerVote{
				{fullnode: "C", voted: "B", auth: false},
				{fullnode: "A", voted: "C", auth: false},
				{fullnode: "B", voted: "C", auth: false},
				{fullnode: "A", voted: "B", auth: false},
			},
			results: []string{"A", "B"},
		}, {
			name:       "Votes from deauthorized validators are discarded immediately (auth votes)",
			validators: []string{"A", "B", "C"},
			votes: []testerVote{
				{fullnode: "C", voted: "B", auth: false},
				{fullnode: "A", voted: "C", auth: false},
				{fullnode: "B", voted: "C", auth: false},
				{fullnode: "A", voted: "B", auth: false},
			},
			results: []string{"A", "B"},
		}, {
			name:       "Cascading changes are not allowed, only the the account being voted on may change",
			validators: []string{"A", "B", "C", "D"},
			votes: []testerVote{
				{fullnode: "A", voted: "C", auth: false},
				{fullnode: "B"},
				{fullnode: "C"},
				{fullnode: "A", voted: "D", auth: false},
				{fullnode: "B", voted: "C", auth: false},
				{fullnode: "C"},
				{fullnode: "A"},
				{fullnode: "B", voted: "D", auth: false},
				{fullnode: "C", voted: "D", auth: false},
			},
			results: []string{"A", "B", "C"},
		}, {
			name:       "Changes reaching consensus out of bounds (via a deauth) execute on touch",
			validators: []string{"A", "B", "C", "D"},
			votes: []testerVote{
				{fullnode: "A", voted: "C", auth: false},
				{fullnode: "B"},
				{fullnode: "C"},
				{fullnode: "A", voted: "D", auth: false},
				{fullnode: "B", voted: "C", auth: false},
				{fullnode: "C"},
				{fullnode: "A"},
				{fullnode: "B", voted: "D", auth: false},
				{fullnode: "C", voted: "D", auth: false},
				{fullnode: "A"},
				{fullnode: "C", voted: "C", auth: true},
			},
			results: []string{"A", "B"},
		}, {
			name:       "Changes reaching consensus out of bounds (via a deauth) may go out of consensus on first touch",
			validators: []string{"A", "B", "C", "D"},
			votes: []testerVote{
				{fullnode: "A", voted: "C", auth: false},
				{fullnode: "B"},
				{fullnode: "C"},
				{fullnode: "A", voted: "D", auth: false},
				{fullnode: "B", voted: "C", auth: false},
				{fullnode: "C"},
				{fullnode: "A"},
				{fullnode: "B", voted: "D", auth: false},
				{fullnode: "C", voted: "D", auth: false},
				{fullnode: "A"},
				{fullnode: "B", voted: "C", auth: true},
			},
			results: []string{"A", "B", "C"},
		}, {
			name: "Ensure that pending votes don't survive authorization status changes. ",
			//This corner case can only appear if a fullnode is quickly added, remove and then
			// readded (or the inverse), while one of the original voters dropped. If a
			// past vote is left cached in the system somewhere, this will interfere with
			// the final fullnode outcome.
			validators: []string{"A", "B", "C", "D", "E"},
			votes: []testerVote{
				{fullnode: "A", voted: "F", auth: true}, // Authorize F, 3 votes needed
				{fullnode: "B", voted: "F", auth: true},
				{fullnode: "C", voted: "F", auth: true},
				{fullnode: "D", voted: "F", auth: false}, // Deauthorize F, 4 votes needed (leave A's previous vote "unchanged")
				{fullnode: "E", voted: "F", auth: false},
				{fullnode: "B", voted: "F", auth: false},
				{fullnode: "C", voted: "F", auth: false},
				{fullnode: "D", voted: "F", auth: true}, // Almost authorize F, 2/3 votes needed
				{fullnode: "E", voted: "F", auth: true},
				{fullnode: "B", voted: "A", auth: false}, // Deauthorize A, 3 votes needed
				{fullnode: "C", voted: "A", auth: false},
				{fullnode: "D", voted: "A", auth: false},
				{fullnode: "B", voted: "F", auth: true}, // Finish authorizing F, 3/3 votes needed
			},
			results: []string{"B", "C", "D", "E", "F"},
		}, {
			name:       "Epoch transitions reset all votes to allow chain checkpointing",
			epoch:      3,
			validators: []string{"A", "B"},
			votes: []testerVote{
				{fullnode: "A", voted: "C", auth: true},
				{fullnode: "B"},
				{fullnode: "A"}, // Checkpoint block, (don't vote here, it's validated outside of snapshots)
				{fullnode: "B", voted: "C", auth: true},
			},
			results: []string{"A", "B"},
		},
	}
	// Run through the scenarios and test them
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Create the account pool and generate the initial set of validators
			accounts := newTesterAccountPool()

			validators := make([]common.Address, len(tt.validators))
			for j, fullnode := range tt.validators {
				validators[j] = accounts.address(fullnode)
			}
			for j := 0; j < len(validators); j++ {
				for k := j + 1; k < len(validators); k++ {
					if bytes.Compare(validators[j][:], validators[k][:]) > 0 {
						validators[j], validators[k] = validators[k], validators[j]
					}
				}
			}
			// Create the genesis block with the initial set of validators
			genesis := &core.Genesis{
				Difficulty: defaultDifficulty,
				Mixhash:    types.SportDigest,
				Config:     params.TestChainConfig,
			}
			b := genesis.ToBlock(nil)
			extra, _ := prepareExtra(b.Header(), validators)
			genesis.ExtraData = extra
			// Create a pristine blockchain with the genesis injected
			db := rawdb.NewMemoryDatabase()
			genesis.Commit(db)

			config := sport.DefaultConfig
			if tt.epoch != 0 {
				config.Epoch = tt.epoch
			}
			engine := New(config, accounts.accounts[tt.validators[0]], db).(*backend)
			chain, err := core.NewBlockChain(db, nil, genesis.Config, engine, vm.Config{}, nil)

			// Assemble a chain of headers from the cast votes
			headers := make([]*types.Header, len(tt.votes))
			for j, vote := range tt.votes {
				headers[j] = &types.Header{
					Number:     big.NewInt(int64(j) + 1),
					Time:       uint64(j) * uint64(config.BlockPeriod),
					Coinbase:   accounts.address(vote.voted),
					Difficulty: defaultDifficulty,
					MixDigest:  types.SportDigest,
				}
				extra, _ := prepareExtra(headers[j], validators)
				headers[j].Extra = extra
				if j > 0 {
					headers[j].ParentHash = headers[j-1].Hash()
				}
				if vote.auth {
					copy(headers[j].Nonce[:], nonceAuthVote)
				}
				copy(headers[j].Extra, genesis.ExtraData)
				accounts.sign(headers[j], vote.fullnode)
			}
			// Pass all the headers through clique and ensure tallying succeeds
			head := headers[len(headers)-1]

			snap, err := engine.snapshot(chain, head.Number.Uint64(), head.Hash(), headers)
			if err != nil {
				t.Errorf("test %d: failed to create voting snapshot: %v", i, err)
				return
			}
			// Verify the final list of validators against the expected ones
			validators = make([]common.Address, len(tt.results))
			for j, fullnode := range tt.results {
				validators[j] = accounts.address(fullnode)
			}
			for j := 0; j < len(validators); j++ {
				for k := j + 1; k < len(validators); k++ {
					if bytes.Compare(validators[j][:], validators[k][:]) > 0 {
						validators[j], validators[k] = validators[k], validators[j]
					}
				}
			}
			result := snap.fullnodes()
			if len(result) != len(validators) {
				t.Errorf("test %d: validators mismatch: have %x, want %x", i, result, validators)
				return
			}
			for j := 0; j < len(result); j++ {
				if !bytes.Equal(result[j][:], validators[j][:]) {
					t.Errorf("test %d, fullnode %d: fullnode mismatch: have %x, want %x", i, j, result[j], validators[j])
				}
			}
		})
	}

}

func TestSaveAndLoad(t *testing.T) {
	snap := &Snapshot{
		Epoch:  5,
		Number: 10,
		Hash:   cmn.HexToHash("1234567890"),
		Votes: []*Vote{
			{
				Fullnode:  cmn.StringToAddress("1234567891"),
				BlockNum:  15,
				Address:   cmn.StringToAddress("1234567892"),
				Authorize: false,
			},
		},
		Tally: map[common.Address]Tally{
			cmn.StringToAddress("1234567893"): {
				Authorize: false,
				Votes:     20,
			},
		},
		FullnodeSet: fullnode.NewFullnodeSet([]common.Address{
			cmn.StringToAddress("1234567894"),
			cmn.StringToAddress("1234567895"),
		}, sport.RoundRobin),
	}
	db := rawdb.NewMemoryDatabase()
	err := snap.store(db)
	if err != nil {
		t.Errorf("store snapshot failed: %v", err)
	}

	snap1, err := loadSnapshot(snap.Epoch, db, snap.Hash)
	if err != nil {
		t.Errorf("load snapshot failed: %v", err)
	}
	if snap.Epoch != snap1.Epoch {
		t.Errorf("epoch mismatch: have %v, want %v", snap1.Epoch, snap.Epoch)
	}
	if snap.Hash != snap1.Hash {
		t.Errorf("hash mismatch: have %v, want %v", snap1.Number, snap.Number)
	}
	if !reflect.DeepEqual(snap.Votes, snap.Votes) {
		t.Errorf("votes mismatch: have %v, want %v", snap1.Votes, snap.Votes)
	}
	if !reflect.DeepEqual(snap.Tally, snap.Tally) {
		t.Errorf("tally mismatch: have %v, want %v", snap1.Tally, snap.Tally)
	}
	if !reflect.DeepEqual(snap.FullnodeSet, snap.FullnodeSet) {
		t.Errorf("fullnode set mismatch: have %v, want %v", snap1.FullnodeSet, snap.FullnodeSet)
	}
}
