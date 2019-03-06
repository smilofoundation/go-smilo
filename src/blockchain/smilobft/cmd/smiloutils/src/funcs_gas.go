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
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"go-smilo/src/blockchain/smilobft"
)

// EstimateGas will estimate the gas cost
func EstimateGas(fromAddress common.Address, toAddress *common.Address, amount *big.Int, data []byte) (gas uint64, err error) {
	msg := smilobft.CallMsg{From: fromAddress, To: toAddress, Value: amount, Data: data}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	gas, err = client.EstimateGas(ctx, msg)
	if err != nil {
		fmt.Println("estimateGas, client.EstimateGas, ", err)
		return gas, err
	}
	return gas, err
}
