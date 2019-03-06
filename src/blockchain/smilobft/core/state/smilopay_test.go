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

package state

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSmiloPay(t *testing.T) {
	resultSmiloPay := []*big.Int{
		big.NewInt(1007999999999999),
		big.NewInt(2516494466531301),
		big.NewInt(3141333333333333),
		big.NewInt(3620789058968723),
		big.NewInt(4024988933062602),
		big.NewInt(4381096170846271),
		big.NewInt(4703041722813604),
		big.NewInt(4999101212558870),
		big.NewInt(5274666666666666),
		big.NewInt(5533483399593904),
	}
	prevBlock := big.NewInt(100)
	newBlock := big.NewInt(110)
	prevsmiloPay := big.NewInt(1000000000000000)
	for i := 0; i < 10; i++ {

		balance := new(big.Int).Mul(big.NewInt(1e+15), big.NewInt(int64(i*20000)))
		//fmt.Println("balance, ", new(big.Int).Div(balance,big.NewInt(1e+15)))
		smiloPay := CalculateSmiloPay(prevBlock, newBlock, prevsmiloPay, balance)
		//fmt.Println(resultSmiloPay[i], smiloPay)
		require.Equal(t, resultSmiloPay[i], smiloPay)
	}
}

func TestSmiloPayMax(t *testing.T) {
	resultSmiloPay := []*big.Int{
		big.NewInt(5000000000000000),
		big.NewInt(8162277660168379),
		big.NewInt(9472135954999579),
		big.NewInt(10477225575051661),
		big.NewInt(11324555320336758),
		big.NewInt(12071067811865475),
		big.NewInt(12745966692414833),
		big.NewInt(13366600265340755),
		big.NewInt(13944271909999158),
		big.NewInt(14486832980505138),
	}
	for i := 0; i < 10; i++ {
		balance := new(big.Int).Mul(big.NewInt(1e+15), big.NewInt(int64(i*1000))) // 1000000000000000000000 Wei = 1000 Ether = 1000 Smilo
		//fmt.Println("balance : ", balance)
		maxSmiloPay, _ := MaxSmiloPay(balance)
		require.NotEmpty(t, maxSmiloPay)
		require.Equal(t, resultSmiloPay[i], maxSmiloPay) // Result in WEI
		//fmt.Println("Max SmiloPay in WEI : ", maxSmiloPay)
		//fmt.Println("SmiloPay in WEI : ", smiloPay)

	}
}
