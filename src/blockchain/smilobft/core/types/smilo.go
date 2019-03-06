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

package types

import (
	"errors"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	SportDigest = common.HexToHash("0x636861696e20706c6174666f726d2077697468206120636f6e736369656e6365")

	SportExtraVanity = 32
	SportExtraSeal   = 65

	ErrInvalidSportHeaderExtra = errors.New("invalid sport header extra-data")
)

type SportExtra struct {
	Fullnodes     []common.Address
	Seal          []byte
	CommittedSeal [][]byte
}

// EncodeRLP serializes ist into the Ethereum RLP format.
func (ist *SportExtra) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{
		ist.Fullnodes,
		ist.Seal,
		ist.CommittedSeal,
	})
}

// DecodeRLP implements rlp.Decoder, and load the sport fields from a RLP stream.
func (ist *SportExtra) DecodeRLP(s *rlp.Stream) error {
	var sportExtra struct {
		Fullnodes     []common.Address
		Seal          []byte
		CommittedSeal [][]byte
	}
	if err := s.Decode(&sportExtra); err != nil {
		return err
	}
	ist.Fullnodes, ist.Seal, ist.CommittedSeal = sportExtra.Fullnodes, sportExtra.Seal, sportExtra.CommittedSeal
	return nil
}

// ExtractSportExtra extracts all values of the SportExtra from the header. It returns an
// error if the length of the given extra-data is less than 32 bytes or the extra-data can not
// be decoded.
func ExtractSportExtra(h *Header) (*SportExtra, error) {
	if len(h.Extra) < SportExtraVanity {
		return nil, ErrInvalidSportHeaderExtra
	}

	var sportExtra *SportExtra
	err := rlp.DecodeBytes(h.Extra[SportExtraVanity:], &sportExtra)
	if err != nil {
		return nil, err
	}
	return sportExtra, nil
}

// SportFilteredHeader returns a filtered header which some information (like seal, committed seals)
// are clean to fulfill the Sport hash rules. It returns nil if the extra-data cannot be
// decoded/encoded by rlp.
func SportFilteredHeader(h *Header, keepSeal bool) *Header {
	newHeader := CopyHeader(h)
	sportExtra, err := ExtractSportExtra(newHeader)
	if err != nil {
		return nil
	}

	if !keepSeal {
		sportExtra.Seal = []byte{}
	}
	sportExtra.CommittedSeal = [][]byte{}

	payload, err := rlp.EncodeToBytes(&sportExtra)
	if err != nil {
		return nil
	}

	newHeader.Extra = append(newHeader.Extra[:SportExtraVanity], payload...)

	return newHeader
}
