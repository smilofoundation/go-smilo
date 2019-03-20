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
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

type Fullnode interface {
	// Address returns address
	Address() common.Address

	// String representation of Fullnode
	String() string
}

// ----------------------------------------------------------------------------

type Fullnodes []Fullnode

func (slice Fullnodes) Len() int {
	return len(slice)
}

func (slice Fullnodes) Less(i, j int) bool {
	return strings.Compare(slice[i].String(), slice[j].String()) < 0
}

func (slice Fullnodes) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// ----------------------------------------------------------------------------

type FullnodeSet interface {
	// Calculate the speaker
	CalcSpeaker(lastSpeaker common.Address, round uint64)
	// Return the fullnode size
	Size() int
	// Return the fullnode array
	List() []Fullnode
	// Get fullnode by index
	GetByIndex(i uint64) Fullnode
	// Get fullnode by given address
	GetByAddress(addr common.Address) (int, Fullnode)
	// Get current speaker
	GetSpeaker() Fullnode
	// Check whether the fullnode with given address is a speaker
	IsSpeaker(address common.Address) bool
	// Add fullnode
	AddFullnode(address common.Address) bool
	// Remove fullnode
	RemoveFullnode(address common.Address) bool
	// Copy fullnode set
	Copy() FullnodeSet
	// Get the maximum number of faulty nodes
	MaxFaulty() int
    // Minimum nodes to approve on Consensus
	MinApprovers() int
	// Get the extra number of faulty nodes
	E() int
	// Get speaker policy
	Policy() SpeakerPolicy
}

// ----------------------------------------------------------------------------

type ProposalSelector func(FullnodeSet, common.Address, uint64) Fullnode
