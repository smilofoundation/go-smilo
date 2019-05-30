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

package sport

import (
	"crypto/ecdsa"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
)

// Backend provides application specific functions for Sport core
type Backend interface {

	// Address returns the owner's address
	Address() common.Address

	// Fullnodes returns the fullnode set
	Fullnodes(blockproposal BlockProposal) FullnodeSet

	// EventMux returns the event mux in backend
	EventMux() *event.TypeMux

	// Broadcast sends a message to all fullnodes (include self)
	Broadcast(fullnodeSet FullnodeSet, payload []byte) error

	// Gossip sends a message to all fullnodes (exclude self)
	Gossip(fullnodeSet FullnodeSet, payload []byte) error

	// Commit delivers an approved proposal to backend.
	// The delivered proposal will be put into blockchain.
	Commit(blockproposal BlockProposal, seals [][]byte) error

	// Verify verifies the proposal. If a consensus.ErrFutureBlock error is returned,
	// the time difference of the proposal and current time is also returned.
	Verify(BlockProposal) (time.Duration, error)

	// Sign signs input data with the backend's private key
	Sign([]byte) ([]byte, error)

	// Get PrivateKey
	GetPrivateKey() *ecdsa.PrivateKey

	// CheckSignature verifies the signature by checking if it's signed by
	// the given fullnode
	CheckSignature(data []byte, addr common.Address, sig []byte) error

	// LastBlockProposal retrieves latest committed proposal and the address of speaker
	LastBlockProposal() (BlockProposal, common.Address)

	// HasBlockProposal checks if the combination of the given hash and height matches any existing blocks
	HasBlockProposal(hash common.Hash, number *big.Int) bool

	// GetSpeaker returns the speaker of the given block height
	GetSpeaker(number uint64) common.Address

	// ParentFullnodes returns the fullnode set of the given proposal's parent block
	ParentFullnodes(proposal BlockProposal) FullnodeSet

	// HasBadBlock returns whether the block with the hash is a bad block
	HasBadBlockProposal(hash common.Hash) bool

	Close() error
}
