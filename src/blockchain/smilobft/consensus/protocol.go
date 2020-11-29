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

// Package consensus implements different Ethereum consensus engines.
package consensus

import (
	"github.com/ethereum/go-ethereum/common"

	"go-smilo/src/blockchain/smilobft/core/types"
)

// Constants to match up protocol versions and messages
const (
	eth63 = 63
	eth64 = 64
	//Istanbul64 = 64
	//Istanbul99 = 99
)

var (
	CliqueProtocol = Protocol{
		Name:     "clique",
		Versions: []uint{eth64, eth63},
		Lengths:  map[uint]uint64{eth64: 17, eth63: 17},
	}

	// Default: Keep up-to-date with eth/protocol.go
	EthProtocol = Protocol{
		Name:     "eth",
		Versions: []uint{eth64, eth63},
		Lengths:  map[uint]uint64{eth64: 17, eth63: 17},
	}

	NorewardsProtocol = Protocol{
		Name:     "Norewards",
		Versions: []uint{0},
		Lengths:  map[uint]uint64{0: 0},
	}

	IstanbulProtocol = Protocol{
		Name:     "istanbul",
		Versions: []uint{eth64, eth63},
		Lengths:  map[uint]uint64{eth64: 18, eth63: 18},
	}

	TendermintProtocol = Protocol{
		Name:     "tendermint",
		Versions: []uint{eth64, eth63},
		Lengths:  map[uint]uint64{eth64: 18, eth63: 18},
	}

	SportProtocol = Protocol{
		Name:     "smilobft",
		Versions: []uint{eth64, eth63},
		Lengths:  map[uint]uint64{eth64: 18, eth63: 18},
	}

	SportDAOProtocol = Protocol{
		Name:     "smilobftdao",
		Versions: []uint{eth64, eth63},
		Lengths:  map[uint]uint64{eth64: 18, eth63: 18},
	}
)

// Protocol defines the protocol of the consensus
type Protocol struct {
	// Official short name of the protocol used during capability negotiation.
	Name string
	// Supported versions of the eth protocol (first is primary).
	Versions []uint
	// Number of implemented message corresponding to different protocol versions.
	Lengths map[uint]uint64
}

// Broadcaster defines the interface to enqueue blocks to fetcher and find peer
type Broadcaster interface {
	// Enqueue add a block into fetcher queue
	Enqueue(id string, block *types.Block)
	// FindPeers retrieves peers by addresses
	FindPeers(map[common.Address]struct{}) map[common.Address]Peer
}

// Peer defines the interface to communicate with peer
type Peer interface {
	// Send sends the message to this peer
	Send(msgcode uint64, data interface{}) error
	String() string
}
