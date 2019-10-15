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


func New(addr common.Address) sport.Fullnode {
	return &fullnode{
		address: addr,
	}
}

func NewSet(addrs []common.Address, policy sport.SpeakerPolicy) sport.FullnodeSet {
	return newFullnodeSet(addrs, policy)
}

func ExtractValidators(extraData []byte) []common.Address {
	// get the validator addresses
	addrs := make([]common.Address, (len(extraData) / common.AddressLength))
	for i := 0; i < len(addrs); i++ {
		copy(addrs[i][:], extraData[i*common.AddressLength:])
	}

	return addrs
}

func (theFullNode *fullnode) Address() common.Address {
	return theFullNode.address
}

func (theFullNode *fullnode) String() string {
	return theFullNode.Address().String()
}
