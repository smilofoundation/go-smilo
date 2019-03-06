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
	"github.com/ethereum/go-ethereum/common"

	"go-smilo/src/blockchain/smilobft/consensus/sport"
)

func calcSeed(fullnodeSet sport.FullnodeSet, speaker common.Address, round uint64) uint64 {
	offset := 0
	if idx, val := fullnodeSet.GetByAddress(speaker); val != nil {
		offset = idx
	}
	return uint64(offset) + round
}

func emptyAddress(addr common.Address) bool {
	return addr == common.Address{}
}

func roundRobinSpeaker(fullnodeSet sport.FullnodeSet, speaker common.Address, round uint64) sport.Fullnode {
	if fullnodeSet.Size() == 0 {
		return nil
	}
	seed := uint64(0)
	if emptyAddress(speaker) {
		seed = round
	} else {
		seed = calcSeed(fullnodeSet, speaker, round) + 1
	}
	pick := seed % uint64(fullnodeSet.Size())
	return fullnodeSet.GetByIndex(pick)
}
