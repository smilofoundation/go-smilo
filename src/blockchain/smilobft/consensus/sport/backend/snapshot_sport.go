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
	"bytes"

	"github.com/ethereum/go-ethereum/common"
)

// fullnodes retrieves the list of authorized fullnodes in ascending order.
func (s *Snapshot) fullnodes() []common.Address {
	fullnodes := make([]common.Address, 0, s.FullnodeSet.Size())
	for _, fullnode := range s.FullnodeSet.List() {
		fullnodes = append(fullnodes, fullnode.Address())
	}
	for i := 0; i < len(fullnodes); i++ {
		for j := i + 1; j < len(fullnodes); j++ {
			if bytes.Compare(fullnodes[i][:], fullnodes[j][:]) > 0 {
				fullnodes[i], fullnodes[j] = fullnodes[j], fullnodes[i]
			}
		}
	}
	return fullnodes
}
