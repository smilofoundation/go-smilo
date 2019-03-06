// Copyright 2019 The go-smilo Authors
// This file is part of the go-smilo library.
//
// The go-smilo library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-smilo library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-smilo library. If not, see <http://www.gnu.org/licenses/>.

package src

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

//GetNextNonce Will get the next nonce for the account
func GetNextNonce(address common.Address) (nextNonce uint64, err error) {
	if nonce == -1 {
		_, err = PendingNonceAt(address)
		if err != nil {
			return nextNonce, err
		}
	}
	nonce++
	nextNonce = uint64(nonce)
	return nextNonce, err
}

//PendingNonceAt will get the next pending nonce
func PendingNonceAt(address common.Address) (currentNonce uint64, err error) {
	if nonce == -1 {
		var tmpNonce uint64
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		tmpNonce, err = client.PendingNonceAt(ctx, address)
		if err != nil {
			err = fmt.Errorf("failed to obtain nonce for %s: %v", address.Hex(), err)
			return currentNonce, err
		}
		nonce = int64(tmpNonce)
		currentNonce = uint64(nonce)
	} else {
		currentNonce = uint64(nonce)
	}
	return currentNonce, err
}
