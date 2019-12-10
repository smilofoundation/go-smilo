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
	"go-smilo/src/blockchain/smilobft/cmn"
	"go-smilo/src/blockchain/smilobft/consensus/sportdao"
	"go-smilo/src/blockchain/smilobft/core/vm"
	"go-smilo/src/blockchain/smilobft/params"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	lru "github.com/hashicorp/golang-lru"

	"go-smilo/src/blockchain/smilobft/consensus"
	"go-smilo/src/blockchain/smilobft/consensus/sportdao/smilobftcore"
	"go-smilo/src/blockchain/smilobft/core/types"
	"go-smilo/src/blockchain/smilobft/ethdb"
)

const (
	// fetcherID is the ID indicates the block is from smilobft engine
	fetcherID = "smilobftdao"
)

// New creates an Ethereum backend for smilobft core engine.
func New(config *sportdao.Config, privateKey *ecdsa.PrivateKey, db ethdb.Database, chainConfig *params.ChainConfig, vmConfig *vm.Config) *Backend {
	// Allocate the snapshot caches and create the engine
	recents, _ := lru.NewARC(inmemorySnapshots)
	recentMessages, _ := lru.NewARC(inmemoryPeers)
	knownMessages, _ := lru.NewARC(inmemoryMessages)

	if config.MaxTimeout == 0 {
		log.Debug("karaidiasa, sportdao, MaxTimeout is nil o/ ")
		config.MaxTimeout = sportdao.DefaultConfig.MaxTimeout
	}

	backend := &Backend{
		config:           config,
		smilobftEventMux: new(cmn.TypeMux),
		privateKey:       privateKey,
		address:          crypto.PubkeyToAddress(privateKey.PublicKey),
		logger:           log.New(),
		db:               db,
		commitChBlock:    make(chan *types.Block, 1),
		recents:          recents,
		candidates:       make(map[common.Address]bool),
		coreStarted:      false,
		recentMessages:   recentMessages,
		knownMessages:    knownMessages,
		vmConfig:         vmConfig,

	}
	backend.core = smilobftcore.New(backend, backend.config)
	return backend
}

// ----------------------------------------------------------------------------

// CalcDifficulty (clique override) always return zero
func (sb *Backend) CalcDifficulty(chain consensus.ChainReader, time uint64, parent *types.Header) *big.Int {
	return new(big.Int)
}

// Address (clique override) implements smilo.Backend.Address
func (sb *Backend) Address() common.Address {
	return sb.address
}

func (sb *Backend) Close() error {
	return nil
}
