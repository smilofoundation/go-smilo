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
	"errors"

	"go-smilo/src/blockchain/smilobft/consensus"
)

const (
	smilobftMsg = 0x11
	NewBlockMsg = 0x07
)

var (
	// errDecodeFailed is returned when decode message fails
	errDecodeFailed = errors.New("fail to decode smilobft message")
)

// Protocol (clique override) implements consensus.Engine.Protocol
func (sb *backend) Protocol() consensus.Protocol {
	return consensus.Protocol{
		Name:     "smilobft",
		Versions: []uint{64},
		Lengths:  []uint64{18},
	}
}
