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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/hashicorp/golang-lru"

	"go-smilo/src/blockchain/smilobft/consensus"
	"go-smilo/src/blockchain/smilobft/consensus/sport"
	"go-smilo/src/blockchain/smilobft/consensus/sport/smilobftcore"
	"go-smilo/src/blockchain/smilobft/core/types"
	"go-smilo/src/blockchain/smilobft/ethdb"
)

const (
	// fetcherID is the ID indicates the block is from smilobft engine
	fetcherID = "smilobft"
)

// New creates an Ethereum backend for smilobft core engine.
func New(config *sport.Config, privateKey *ecdsa.PrivateKey, db ethdb.Database) consensus.SmiloBFT {
	// Allocate the snapshot caches and create the engine
	recents, _ := lru.NewARC(inmemorySnapshots)
	recentMessages, _ := lru.NewARC(inmemoryPeers)
	knownMessages, _ := lru.NewARC(inmemoryMessages)
	backend := &backend{
		config:           config,
		smilobftEventMux: new(event.TypeMux),
		privateKey:       privateKey,
		address:          crypto.PubkeyToAddress(privateKey.PublicKey),
		logger:           log.New(),
		db:               db,
		commitCh:         make(chan *types.Block, 1),
		recents:          recents,
		candidates:       make(map[common.Address]bool),
		coreStarted:      false,
		recentMessages:   recentMessages,
		knownMessages:    knownMessages,
	}
	backend.core = smilobftcore.New(backend, backend.config)
	return backend
}

// ----------------------------------------------------------------------------

// CalcDifficulty (clique override) always return zero
func (sb *backend) CalcDifficulty(chain consensus.ChainReader, time uint64, parent *types.Header) *big.Int {
	return new(big.Int)
}

// Address (clique override) implements smilo.Backend.Address
func (sb *backend) Address() common.Address {
	return sb.address
}

func (sb *backend) Close() error {
	return nil
}
