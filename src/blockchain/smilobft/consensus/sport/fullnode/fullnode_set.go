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
	"reflect"
	"sort"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum/go-ethereum/common"

	"math"

	"go-smilo/src/blockchain/smilobft/consensus/sport"
)

func (fullnodeSet *fullnodeSet) Size() int {
	fullnodeSet.fullnodeMu.RLock()
	defer fullnodeSet.fullnodeMu.RUnlock()
	return len(fullnodeSet.fullnodes)
}

func (fullnodeSet *fullnodeSet) List() []sport.Fullnode {
	fullnodeSet.fullnodeMu.RLock()
	defer fullnodeSet.fullnodeMu.RUnlock()
	return fullnodeSet.fullnodes
}

func (fullnodeSet *fullnodeSet) GetByIndex(i uint64) sport.Fullnode {
	fullnodeSet.fullnodeMu.RLock()
	defer fullnodeSet.fullnodeMu.RUnlock()
	if i < uint64(fullnodeSet.Size()) {
		return fullnodeSet.fullnodes[i]
	}
	return nil
}

func (fullnodeSet *fullnodeSet) GetByAddress(addr common.Address) (int, sport.Fullnode) {
	for i, val := range fullnodeSet.List() {
		if addr == val.Address() {
			return i, val
		}
	}
	return -1, nil
}

func (fullnodeSet *fullnodeSet) GetSpeaker() sport.Fullnode {
	return fullnodeSet.speaker
}

func (fullnodeSet *fullnodeSet) IsSpeaker(address common.Address) bool {
	_, val := fullnodeSet.GetByAddress(address)
	return reflect.DeepEqual(fullnodeSet.GetSpeaker(), val)
}

func (fullnodeSet *fullnodeSet) CalcSpeaker(lastSpeaker common.Address, round uint64) {
	fullnodeSet.fullnodeMu.RLock()
	defer fullnodeSet.fullnodeMu.RUnlock()
	fullnodeSet.speaker = fullnodeSet.selector(fullnodeSet, lastSpeaker, round)
	log.Debug("CalcSpeaker, Selected speaker ", "speaker", fullnodeSet.speaker)
}

func (fullnodeSet *fullnodeSet) AddFullnode(address common.Address) bool {
	fullnodeSet.fullnodeMu.Lock()
	defer fullnodeSet.fullnodeMu.Unlock()
	for _, v := range fullnodeSet.fullnodes {
		if v.Address() == address {
			return false
		}
	}
	fullnodeSet.fullnodes = append(fullnodeSet.fullnodes, NewFullNode(address))
	// TODO: we may not need to re-sort it again
	// sort fullnode
	sort.Sort(fullnodeSet.fullnodes)
	return true
}

func (fullnodeSet *fullnodeSet) RemoveFullnode(address common.Address) bool {
	fullnodeSet.fullnodeMu.Lock()
	defer fullnodeSet.fullnodeMu.Unlock()

	for i, v := range fullnodeSet.fullnodes {
		if v.Address() == address {
			fullnodeSet.fullnodes = append(fullnodeSet.fullnodes[:i], fullnodeSet.fullnodes[i+1:]...)
			return true
		}
	}
	return false
}

func (fullnodeSet *fullnodeSet) Copy() sport.FullnodeSet {
	fullnodeSet.fullnodeMu.RLock()
	defer fullnodeSet.fullnodeMu.RUnlock()

	addresses := make([]common.Address, 0, len(fullnodeSet.fullnodes))
	for _, v := range fullnodeSet.fullnodes {
		addresses = append(addresses, v.Address())
	}
	return NewFullnodeSet(addresses, fullnodeSet.policy)
}

func (fullnodeSet *fullnodeSet) MaxFaulty() int {
	return int(math.Ceil(float64(fullnodeSet.Size()+1)/3.0)) - 1
}

func (fullnodeSet *fullnodeSet) MinApprovers() int {
	return int(math.Ceil(float64(2*fullnodeSet.Size()) / 3.0))
}

func (fullnodeSet *fullnodeSet) E() int { return 1 }

func (fullnodeSet *fullnodeSet) Policy() sport.SpeakerPolicy { return fullnodeSet.policy }
