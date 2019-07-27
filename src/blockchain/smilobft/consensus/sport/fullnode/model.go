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

package fullnode

import (
	"github.com/ethereum/go-ethereum/log"
	"sync"

	"github.com/ethereum/go-ethereum/common"

	"sort"

	"go-smilo/src/blockchain/smilobft/consensus/sport"
)

// ----------------------------------------------------------------------------

type fullnode struct {
	address common.Address
}

func NewFullNode(addr common.Address) sport.Fullnode {
	return &fullnode{
		address: addr,
	}
}

// ----------------------------------------------------------------------------

type fullnodeSet struct {
	fullnodes sport.Fullnodes
	policy    sport.SpeakerPolicy

	speaker    sport.Fullnode
	fullnodeMu sync.RWMutex
	selector   sport.BlockProposalSelector
}

func NewFullnodeSet(addrs []common.Address, policy sport.SpeakerPolicy) sport.FullnodeSet {
	return newFullnodeSet(addrs, policy)
}

func newFullnodeSet(addrs []common.Address, policy sport.SpeakerPolicy) *fullnodeSet {
	fullnodeSet := &fullnodeSet{}

	fullnodeSet.policy = policy
	// init fullnodes
	fullnodeSet.fullnodes = make([]sport.Fullnode, len(addrs))
	for i, addr := range addrs {
		fullnodeSet.fullnodes[i] = NewFullNode(addr)
	}
	// sort fullnode
	sort.Sort(fullnodeSet.fullnodes)
	// init speaker
	if fullnodeSet.Size() > 0 {
		fullnodeSet.speaker = fullnodeSet.GetByIndex(0)
		log.Debug("newFullnodeSet, Going to set initial speaker, ", "new speaker", fullnodeSet.speaker.String())
	}
	fullnodeSet.selector = roundRobinSpeaker

	return fullnodeSet
}

// ----------------------------------------------------------------------------
