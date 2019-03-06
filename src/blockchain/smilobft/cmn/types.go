// Copyright 2015 The go-ethereum Authors
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

package cmn

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

func StringToHash(s string) common.Hash { return common.BytesToHash([]byte(s)) }
func BigToHash(b *big.Int) common.Hash  { return common.BytesToHash(b.Bytes()) }
func HexToHash(s string) common.Hash    { return common.BytesToHash(FromHex(s)) }

func StringToAddress(s string) common.Address { return common.BytesToAddress([]byte(s)) }
func BigToAddress(b *big.Int) common.Address  { return common.BytesToAddress(b.Bytes()) }
func HexToAddress(s string) common.Address    { return common.BytesToAddress(FromHex(s)) }

func EmptyHash(h common.Hash) bool {
	return h == common.Hash{}
}
