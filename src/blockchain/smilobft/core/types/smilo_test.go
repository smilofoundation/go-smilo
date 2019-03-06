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

package types_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/rlp"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"golang.org/x/crypto/sha3"

	"go-smilo/src/blockchain/smilobft/core/types"
)

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewLegacyKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

func TestHeaderHash(t *testing.T) {
	expectedExtra := common.FromHex("0x0000000000000000000000000000000000000000000000000000000000000000f89af8549444add0ec310f115a0e603b2d7db9f067778eaf8a94294fc7e8f22b3bcdcf955dd7ff3ba2ed833f8212946beaaed781d2d2ab6350f5c4566a2c6eaac407a6948be76812f765c24641ec63dc2852b378aba2b440b8410000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0")
	expectedHash := common.HexToHash("0xb5a2b4148728a1973247c129ecce1cc04739d62d80b57512a14a9364c7ee9c4d")

	// for sport consensus
	header := &types.Header{MixDigest: types.SportDigest, Extra: expectedExtra}
	if !reflect.DeepEqual(header.Hash(), expectedHash) {
		t.Errorf("expected: %v, but got: %v", expectedHash.Hex(), header.Hash().Hex())
	}

	// append useless information to extra-data
	unexpectedExtra := append(expectedExtra, []byte{1, 2, 3}...)
	header.Extra = unexpectedExtra
	if !reflect.DeepEqual(header.Hash(), rlpHash(header)) {
		t.Errorf("expected: %v, but got: %v", rlpHash(header).Hex(), header.Hash().Hex())
	}
}

func TestExtractToSport(t *testing.T) {
	testCases := []struct {
		vanity         []byte
		sportRawData   []byte
		expectedResult *types.SportExtra
		expectedErr    error
	}{
		{
			// normal case
			bytes.Repeat([]byte{0x00}, types.SportExtraVanity),
			hexutil.MustDecode("0xf858f8549444add0ec310f115a0e603b2d7db9f067778eaf8a94294fc7e8f22b3bcdcf955dd7ff3ba2ed833f8212946beaaed781d2d2ab6350f5c4566a2c6eaac407a6948be76812f765c24641ec63dc2852b378aba2b44080c0"),
			&types.SportExtra{
				Fullnodes: []common.Address{
					common.BytesToAddress(hexutil.MustDecode("0x44add0ec310f115a0e603b2d7db9f067778eaf8a")),
					common.BytesToAddress(hexutil.MustDecode("0x294fc7e8f22b3bcdcf955dd7ff3ba2ed833f8212")),
					common.BytesToAddress(hexutil.MustDecode("0x6beaaed781d2d2ab6350f5c4566a2c6eaac407a6")),
					common.BytesToAddress(hexutil.MustDecode("0x8be76812f765c24641ec63dc2852b378aba2b440")),
				},
				Seal:          []byte{},
				CommittedSeal: [][]byte{},
			},
			nil,
		},
		{
			// insufficient vanity
			bytes.Repeat([]byte{0x00}, types.SportExtraVanity-1),
			nil,
			nil,
			types.ErrInvalidSportHeaderExtra,
		},
	}
	for _, test := range testCases {
		h := &types.Header{Extra: append(test.vanity, test.sportRawData...)}
		sportExtra, err := types.ExtractSportExtra(h)
		if err != test.expectedErr {
			t.Errorf("expected: %v, but got: %v", test.expectedErr, err)
		}
		if !reflect.DeepEqual(sportExtra, test.expectedResult) {
			t.Errorf("expected: %v, but got: %v", test.expectedResult, sportExtra)
		}
	}
}
