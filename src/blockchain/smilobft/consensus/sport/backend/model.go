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
	"crypto/ecdsa"
	"encoding/json"
	"go-smilo/src/blockchain/smilobft/consensus/clique"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	lru "github.com/hashicorp/golang-lru"

	"go-smilo/src/blockchain/smilobft/consensus"
	"go-smilo/src/blockchain/smilobft/consensus/sport"
	"go-smilo/src/blockchain/smilobft/consensus/sport/fullnode"
	"go-smilo/src/blockchain/smilobft/consensus/sport/smilobftcore"
	"go-smilo/src/blockchain/smilobft/core/types"
	"go-smilo/src/blockchain/smilobft/ethdb"
)

// backend is the override for Clique backend, with extra functions for Smilo BFT
type backend struct {
	config           *sport.Config
	smilobftEventMux *event.TypeMux
	privateKey       *ecdsa.PrivateKey
	address          common.Address
	signFn           clique.SignerFn // Signer function to authorize hashes with

	core         smilobftcore.Engine
	logger       log.Logger
	db           ethdb.Database
	chain        consensus.ChainReader
	currentBlock func() *types.Block
	hasBadBlock  func(hash common.Hash) bool

	// the channels for smilobft engine notifications
	commitChBlock     chan *types.Block
	proposedBlockHash common.Hash
	sealMu            sync.Mutex
	lock              sync.RWMutex // Protects the signer fields

	coreStarted bool
	coreMu      sync.RWMutex

	// Current list of candidates we are pushing
	candidates map[common.Address]bool
	// Protects the signer fields
	candidatesLock sync.RWMutex
	// Snapshots for recent block to speed up reorgs
	recents *lru.ARCCache

	// event subscription for ChainHeadEvent event
	broadcaster consensus.Broadcaster

	recentMessages *lru.ARCCache // the cache of peer's messages
	knownMessages  *lru.ARCCache // the cache of self messages
}

// ----------------------------------------------------------------------------

// Vote represents a single vote that an authorized fullnode made to modify the
// list of authorizations.
type Vote struct {
	Fullnode  common.Address `json:"fullnode"`  // Authorized fullnode that cast this vote
	BlockNum  uint64         `json:"block"`     // Block number the vote was cast in (expire old votes)
	Address   common.Address `json:"address"`   // Account being voted on to change its authorization
	Authorize bool           `json:"authorize"` // Whether to authorize or deauthorize the voted account
}

// ----------------------------------------------------------------------------

// Tally is a simple vote tally to keep the current score of votes. Votes that
// go against the proposal aren't counted since it's equivalent to not voting.
type Tally struct {
	Authorize bool `json:"authorize"` // Whether the vote it about authorizing or kicking someone
	Votes     int  `json:"votes"`     // Number of votes until now wanting to pass the proposal
}

// ----------------------------------------------------------------------------

// Snapshot is the state of the authorization voting at a given point in time.
type Snapshot struct {
	Epoch uint64 // The number of blocks after which to checkpoint and reset the pending votes

	Number      uint64                   // Block number where the snapshot was created
	Hash        common.Hash              // Block hash where the snapshot was created
	Votes       []*Vote                  // List of votes cast in chronological order
	Tally       map[common.Address]Tally // Current vote tally to avoid recalculating
	FullnodeSet sport.FullnodeSet        // Set of authorized fullnodes at this moment
}

// ----------------------------------------------------------------------------

type snapshotJSON struct {
	Epoch  uint64                   `json:"epoch"`
	Number uint64                   `json:"number"`
	Hash   common.Hash              `json:"hash"`
	Votes  []*Vote                  `json:"votes"`
	Tally  map[common.Address]Tally `json:"tally"`

	// for fullnode set
	Fullnodes []common.Address    `json:"fullnodes"`
	Policy    sport.SpeakerPolicy `json:"policy"`
}

func (s *Snapshot) toJSONStruct() *snapshotJSON {
	return &snapshotJSON{
		Epoch:     s.Epoch,
		Number:    s.Number,
		Hash:      s.Hash,
		Votes:     s.Votes,
		Tally:     s.Tally,
		Fullnodes: s.fullnodes(),
		Policy:    s.FullnodeSet.Policy(),
	}
}

// Unmarshal from a json byte array
func (s *Snapshot) UnmarshalJSON(b []byte) error {
	var j snapshotJSON
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}

	s.Epoch = j.Epoch
	s.Number = j.Number
	s.Hash = j.Hash
	s.Votes = j.Votes
	s.Tally = j.Tally
	s.FullnodeSet = fullnode.NewFullnodeSet(j.Fullnodes, j.Policy)
	return nil
}

// Marshal to a json byte array
func (s *Snapshot) MarshalJSON() ([]byte, error) {
	j := s.toJSONStruct()
	return json.Marshal(j)
}

// ----------------------------------------------------------------------------
