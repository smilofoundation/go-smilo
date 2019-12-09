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

package smilobftcore

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"

	"go-smilo/src/blockchain/smilobft/consensus/sportdao"
)

// PrepareCommittedSeal returns a committed seal for the given hash
func PrepareCommittedSeal(hash common.Hash) []byte {
	var buf bytes.Buffer
	buf.Write(hash.Bytes())
	buf.Write([]byte{byte(msgCommit)})
	return buf.Bytes()
}

func Encode(val interface{}) ([]byte, error) {
	return rlp.EncodeToBytes(val)
}

func toPriority(msgCode uint64, view *sportdao.View) float32 {
	if msgCode == msgRoundChange {
		// For msgRoundChange, set the message priority based on its sequence
		return -float32(view.Sequence.Uint64() * 1000)
	}
	// FIXME: round will be reset as 0 while new sequence
	// 10 * Round limits the range of message code is from 0 to 9
	// 1000 * Sequence limits the range of round is from 0 to 99
	return -float32(view.Sequence.Uint64()*1000 + view.Round.Uint64()*10 + uint64(msgPriority[msgCode]))
}
