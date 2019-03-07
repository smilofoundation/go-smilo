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
	"github.com/orinocopay/go-etherutils"
	"fmt"
)

func TestSmiloPay(t *testing.T) {
	resultSmiloPay := []*big.Int{
		big.NewInt(1079999999999999),
		big.NewInt(16164944665313013),
		big.NewInt(22413333333333333),
		big.NewInt(27207890589687233),
		big.NewInt(31249889330626027),
		big.NewInt(34810961708462712),
		big.NewInt(38030417228136048),
		big.NewInt(40991012125588708),
		big.NewInt(43746666666666666),
		big.NewInt(46334833995939041),
	}
	prevBlock := big.NewInt(100)
	newBlock := big.NewInt(110)
	prevsmiloPay := big.NewInt(1000000000000000)
	for i := 0; i < 10; i++ {
		newbalance, _ := etherutils.StringToWei(fmt.Sprintf("%d0 ether", i))
		smiloPay := CalculateSmiloPay(prevBlock, newBlock, prevsmiloPay, newbalance)
		fmt.Println("SmiloPayExp: ", resultSmiloPay[i], " SmiloPay", smiloPay) //require.Equal(t, resultSmiloPay[i], smiloPay)
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
		newbalance, _ := etherutils.StringToWei(fmt.Sprintf("%d ether", i))

		maxSmiloPay, _ := MaxSmiloPay(newbalance)
		require.NotEmpty(t, maxSmiloPay)
		require.Equal(t, resultSmiloPay[i], maxSmiloPay) // Result in WEI

	}
}

func TestSmiloPayMaxHundredTen(t *testing.T) {
	balance, _ := etherutils.StringToWei("110 ether")
	maxSmiloPay, _ := MaxSmiloPay(balance)
	require.NotEmpty(t, maxSmiloPay)
	require.Equal(t, maxSmiloPay, big.NewInt(38166247903553998))
}

func TestSmiloPaySpeedLarge(t *testing.T) {
	prevBlock := big.NewInt(100)
	newBlock := big.NewInt(110)
	prevsmiloPay := big.NewInt(0)
	balance, _ := etherutils.StringToWei("100000000 ether")
	smiloPay := CalculateSmiloPay(prevBlock, newBlock, prevsmiloPay, balance)
	require.Equal(t, smiloPay, big.NewInt(35457331097124265))
}

func TestSmiloPayCalculations(t *testing.T) {

	smallTxPrice, _ := etherutils.StringToWei("0.000021 ether")
	averageTxPrice, _ := etherutils.StringToWei("0.000084 ether")
	bigTxPrice, _ := etherutils.StringToWei("0.00042 ether")

	smilo := []*big.Int{
		big.NewInt(0),
		big.NewInt(1),
		big.NewInt(10),
		big.NewInt(50),
		big.NewInt(100),
		big.NewInt(110),
		big.NewInt(500),
		big.NewInt(1000),
		big.NewInt(10000),
		big.NewInt(100000),
		big.NewInt(1000000),
		big.NewInt(10000000),
		big.NewInt(100000000),
		big.NewInt(1000000000),
		big.NewInt(10000000000),
		big.NewInt(100000000000),
		big.NewInt(1000000000000),
	}

	for i := 0; i < 17; i++ {
		newbalance, _ := etherutils.StringToWei(fmt.Sprintf("%d ether", smilo[i]))
		fmt.Println("Balance Smilo : ", smilo[i])


		maxSmiloPay, _ := MaxSmiloPay(newbalance)
		require.NotEmpty(t, maxSmiloPay)
		fmt.Println("MaxSmiloPay : ", etherutils.WeiToString(maxSmiloPay, true))
		fmt.Println("MaxSmallTx : ", new(big.Int).Div(maxSmiloPay, smallTxPrice)) // 1 Gwei * 21000 Gas
		fmt.Println("MaxAverageTx : ", new(big.Int).Div(maxSmiloPay, averageTxPrice)) // 4 Gwei * 21000 Gas
		fmt.Println("MaxBigTx : ", new(big.Int).Div(maxSmiloPay, bigTxPrice)) // 20 Gwei * 21000 Gas


		prevBlock := big.NewInt(100)
		newBlock := big.NewInt(101)
		prevsmiloPay := big.NewInt(0)
		smiloPay := CalculateSmiloPay(prevBlock, newBlock, prevsmiloPay, newbalance)
		fmt.Println("SmiloPaySpeed: ", etherutils.WeiToString(smiloPay, true), "/ Block")
		fmt.Println("MaxSmallTx/block : ", new(big.Int).Div(smiloPay, smallTxPrice)) // 1 Gwei * 21000 Gas
		fmt.Println("MaxAverageTx/block : ", new(big.Int).Div(smiloPay, averageTxPrice)) // 4 Gwei * 21000 Gas
		fmt.Println("MaxBigTx/block : ", new(big.Int).Div(smiloPay, bigTxPrice)) // 20 Gwei * 21000 Gas

		fmt.Println("Blocks till full : ", new(big.Int).Div(maxSmiloPay, smiloPay))

		fmt.Println()
		fmt.Println()
	}
}