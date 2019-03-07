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
		big.NewInt(1004999999999999),
		big.NewInt(1026081851067789),
		big.NewInt(1034814239699997),
		big.NewInt(1041514837167011),
		big.NewInt(1047163702135578),
		big.NewInt(1052140452079103),
		big.NewInt(1056639777949432),
		big.NewInt(1060777335102271),
		big.NewInt(1064628479399994),
		big.NewInt(1068245553203367),
	}
	prevBlock := big.NewInt(100)
	newBlock := big.NewInt(110)
	prevsmiloPay := big.NewInt(1000000000000000)
	for i := 0; i < 10; i++ {
		newbalance, _ := etherutils.StringToWei(fmt.Sprintf("%d0 ether", i))
		smiloPay := CalculateSmiloPay(prevBlock, newBlock, prevsmiloPay, newbalance)
		require.Equal(t, resultSmiloPay[i], smiloPay)
	}
}

func TestSmiloPayMax(t *testing.T) {
	resultSmiloPay := []*big.Int{
		big.NewInt(5000000000000000),
		big.NewInt(5100000000000000),
		big.NewInt(5141421356237309),
		big.NewInt(5173205080756887),
		big.NewInt(5200000000000000),
		big.NewInt(5223606797749979),
		big.NewInt(5244948974278317),
		big.NewInt(5264575131106459),
		big.NewInt(5282842712474619),
		big.NewInt(5300000000000000),
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
	require.Equal(t, big.NewInt(6048808848170151), maxSmiloPay)
}

func TestSmiloPaySpeedLarge(t *testing.T) {
	prevBlock := big.NewInt(100)
	newBlock := big.NewInt(110)
	prevsmiloPay := big.NewInt(0)
	balance, _ := etherutils.StringToWei("110 ether")
	smiloPay := CalculateSmiloPay(prevBlock, newBlock, prevsmiloPay, balance)
	require.Equal(t, big.NewInt(74920589878010), smiloPay)
}

func TestSmiloPaySpeedVeryLarge(t *testing.T) {
	prevBlock := big.NewInt(100)
	newBlock := big.NewInt(110)
	prevsmiloPay := big.NewInt(0)
	balance, _ := etherutils.StringToWei("100000000 ether")
	smiloPay := CalculateSmiloPay(prevBlock, newBlock, prevsmiloPay, balance)
	require.Equal(t, big.NewInt(66671666666), new(big.Int).Div(smiloPay, big.NewInt(1e6)))
}

func TestSmiloPayCalculations(t *testing.T) {

	smallTxPrice, _ := etherutils.StringToWei("0.000021 ether")
	averageTxPrice, _ := etherutils.StringToWei("0.000084 ether")
	bigTxPrice, _ := etherutils.StringToWei("0.00042 ether")

	testCases := []struct {
		name                 string
		balance              *big.Int
		maxGas               *big.Int
		maxSmiloPay          string
		maxSmallTx           *big.Int
		maxAverageTx         *big.Int
		maxBigTx             *big.Int
		recoverySpeed        string
		maxSmallTxPerBlock   *big.Int
		maxAverageTxPerBlock *big.Int
		maxBigTxPerBlock     *big.Int
		blocksTillFull       *big.Int
	}{
		{
			name:                 "0",
			balance:              big.NewInt(0),
			maxSmiloPay:          "0.005 Ether",
			maxSmallTx:           big.NewInt(238),
			maxAverageTx:         big.NewInt(59),
			maxBigTx:             big.NewInt(11),
			recoverySpeed:        "499.999999999 GWei",
			maxSmallTxPerBlock:   big.NewInt(0),
			maxAverageTxPerBlock: big.NewInt(0),
			maxBigTxPerBlock:     big.NewInt(0),
			blocksTillFull:       big.NewInt(10000),
		},
		{
			name:                 "1",
			balance:              big.NewInt(1),
			maxSmiloPay:          "0.0051 Ether",
			recoverySpeed:        "0.000001166666666666 Ether",
			maxSmallTx:           big.NewInt(242),
			maxAverageTx:         big.NewInt(60),
			maxBigTx:             big.NewInt(12),
			maxSmallTxPerBlock:   big.NewInt(0),
			maxAverageTxPerBlock: big.NewInt(0),
			maxBigTxPerBlock:     big.NewInt(0),
			blocksTillFull:       big.NewInt(4371),
		},
		{
			name:                 "10",
			balance:              big.NewInt(10),
			maxSmiloPay:          "0.005316227766016838 Ether",
			recoverySpeed:        "0.000002608185106778 Ether",
			maxSmallTx:           big.NewInt(253),
			maxAverageTx:         big.NewInt(63),
			maxBigTx:             big.NewInt(12),
			maxSmallTxPerBlock:   big.NewInt(0),
			maxAverageTxPerBlock: big.NewInt(0),
			maxBigTxPerBlock:     big.NewInt(0),
			blocksTillFull:       big.NewInt(2038),
		},
		{
			name:                 "20",
			balance:              big.NewInt(20),
			maxSmiloPay:          "0.005447213595499958 Ether",
			recoverySpeed:        "0.000003481423969999 Ether",
			maxSmallTx:           big.NewInt(259),
			maxAverageTx:         big.NewInt(64),
			maxBigTx:             big.NewInt(12),
			maxSmallTxPerBlock:   big.NewInt(0),
			maxAverageTxPerBlock: big.NewInt(0),
			maxBigTxPerBlock:     big.NewInt(0),
			blocksTillFull:       big.NewInt(1564),
		},
		{
			name:                 "30",
			balance:              big.NewInt(30),
			maxSmiloPay:          "0.005547722557505166 Ether",
			recoverySpeed:        "0.000004151483716701 Ether",
			maxSmallTx:           big.NewInt(264),
			maxAverageTx:         big.NewInt(66),
			maxBigTx:             big.NewInt(13),
			maxSmallTxPerBlock:   big.NewInt(0),
			maxAverageTxPerBlock: big.NewInt(0),
			maxBigTxPerBlock:     big.NewInt(0),
			blocksTillFull:       big.NewInt(1336),
		},
		{
			name:                 "40",
			balance:              big.NewInt(40),
			maxSmiloPay:          "0.005632455532033675 Ether",
			recoverySpeed:        "0.000004716370213557 Ether",
			maxSmallTx:           big.NewInt(268),
			maxAverageTx:         big.NewInt(67),
			maxBigTx:             big.NewInt(13),
			maxSmallTxPerBlock:   big.NewInt(0),
			maxAverageTxPerBlock: big.NewInt(0),
			maxBigTxPerBlock:     big.NewInt(0),
			blocksTillFull:       big.NewInt(1194),
		},
		{
			name:                 "50",
			balance:              big.NewInt(50),
			maxSmiloPay:          "0.005707106781186547 Ether",
			recoverySpeed:        "0.00000521404520791 Ether",
			maxSmallTx:           big.NewInt(271),
			maxAverageTx:         big.NewInt(67),
			maxBigTx:             big.NewInt(13),
			maxSmallTxPerBlock:   big.NewInt(0),
			maxAverageTxPerBlock: big.NewInt(0),
			maxBigTxPerBlock:     big.NewInt(0),
			blocksTillFull:       big.NewInt(1094),
		},
		{
			name:                 "100",
			balance:              big.NewInt(100),
			maxSmiloPay:          "0.006 Ether",
			recoverySpeed:        "0.000007166666666666 Ether",
			maxSmallTx:           big.NewInt(285),
			maxAverageTx:         big.NewInt(71),
			maxBigTx:             big.NewInt(14),
			maxSmallTxPerBlock:   big.NewInt(0),
			maxAverageTxPerBlock: big.NewInt(0),
			maxBigTxPerBlock:     big.NewInt(0),
			blocksTillFull:       big.NewInt(837),
		},
		{
			name:                 "200",
			balance:              big.NewInt(200),
			maxSmiloPay:          "0.006414213562373095 Ether",
			recoverySpeed:        "0.00000992809041582 Ether",
			maxSmallTx:           big.NewInt(305),
			maxAverageTx:         big.NewInt(76),
			maxBigTx:             big.NewInt(15),
			maxSmallTxPerBlock:   big.NewInt(0),
			maxAverageTxPerBlock: big.NewInt(0),
			maxBigTxPerBlock:     big.NewInt(0),
			blocksTillFull:       big.NewInt(646),
		},
		{
			name:                 "300",
			balance:              big.NewInt(300),
			maxSmiloPay:          "0.006732050807568877 Ether",
			recoverySpeed:        "0.000012047005383792 Ether",
			maxSmallTx:           big.NewInt(320),
			maxAverageTx:         big.NewInt(80),
			maxBigTx:             big.NewInt(16),
			maxSmallTxPerBlock:   big.NewInt(0),
			maxAverageTxPerBlock: big.NewInt(0),
			maxBigTxPerBlock:     big.NewInt(0),
			blocksTillFull:       big.NewInt(558),
		},
		{
			name:                 "400",
			balance:              big.NewInt(400),
			maxSmiloPay:          "0.007 Ether",
			recoverySpeed:        "0.000013833333333333 Ether",
			maxSmallTx:           big.NewInt(333),
			maxAverageTx:         big.NewInt(83),
			maxBigTx:             big.NewInt(16),
			maxSmallTxPerBlock:   big.NewInt(0),
			maxAverageTxPerBlock: big.NewInt(0),
			maxBigTxPerBlock:     big.NewInt(0),
			blocksTillFull:       big.NewInt(506),
		},
		{
			name:                 "500",
			balance:              big.NewInt(500),
			maxSmiloPay:          "0.007236067977499789 Ether",
			recoverySpeed:        "0.000015407119849998 Ether",
			maxSmallTx:           big.NewInt(344),
			maxAverageTx:         big.NewInt(86),
			maxBigTx:             big.NewInt(17),
			maxSmallTxPerBlock:   big.NewInt(0),
			maxAverageTxPerBlock: big.NewInt(0),
			maxBigTxPerBlock:     big.NewInt(0),
			blocksTillFull:       big.NewInt(469),
		},
		{
			name:                 "1000",
			balance:              big.NewInt(1000),
			maxSmiloPay:          "0.008162277660168379 Ether",
			recoverySpeed:        "0.000021581851067789 Ether",
			maxSmallTx:           big.NewInt(388),
			maxAverageTx:         big.NewInt(97),
			maxBigTx:             big.NewInt(19),
			maxSmallTxPerBlock:   big.NewInt(1),
			maxAverageTxPerBlock: big.NewInt(0),
			maxBigTxPerBlock:     big.NewInt(0),
			blocksTillFull:       big.NewInt(378),
		},
	}

	for _, test := range testCases {

		t.Run(test.name, func(t *testing.T) {

			newbalance, _ := etherutils.StringToWei(fmt.Sprintf("%d ether", test.balance))
			t.Log("Balance Smilo : ", test.balance)

			maxSmiloPay, _ := MaxSmiloPay(newbalance)
			require.NotEmpty(t, maxSmiloPay)

			maxSmiloPayStr := etherutils.WeiToString(maxSmiloPay, true)
			t.Log("MaxSmiloPay : ", maxSmiloPayStr)
			require.Equal(t, test.maxSmiloPay, maxSmiloPayStr)

			maxSmallTx := new(big.Int).Div(maxSmiloPay, smallTxPrice)
			t.Log("MaxSmallTx : ", maxSmallTx) // 1 Gwei * 21000 Gas
			require.Equal(t, test.maxSmallTx, maxSmallTx)

			maxAverageTx := new(big.Int).Div(maxSmiloPay, averageTxPrice)
			t.Log("MaxAverageTx : ", maxAverageTx) // 4 Gwei * 21000 Gas
			require.Equal(t, test.maxAverageTx, maxAverageTx)

			maxBigTx := new(big.Int).Div(maxSmiloPay, bigTxPrice)
			t.Log("MaxBigTx : ", maxBigTx) // 20 Gwei * 21000 Gas
			require.Equal(t, test.maxBigTx, maxBigTx)

			prevBlock := big.NewInt(100)
			newBlock := big.NewInt(101)
			prevsmiloPay := big.NewInt(0)
			smiloPay := CalculateSmiloPay(prevBlock, newBlock, prevsmiloPay, newbalance)

			recoverySpeed := etherutils.WeiToString(smiloPay, true)
			t.Log("RecoverySpeed: ", recoverySpeed)
			require.Equal(t, test.recoverySpeed, recoverySpeed)

			maxSmallTxPerBlock := new(big.Int).Div(smiloPay, smallTxPrice)
			t.Log("MaxSmallTx/block : ", maxSmallTxPerBlock) // 1 Gwei * 21000 Gas
			require.Equal(t, test.maxSmallTxPerBlock, maxSmallTxPerBlock)

			maxAverageTxPerBlock := new(big.Int).Div(smiloPay, averageTxPrice)
			t.Log("MaxAverageTx/block : ", maxAverageTxPerBlock) // 4 Gwei * 21000 Gas
			require.Equal(t, test.maxAverageTxPerBlock, maxAverageTxPerBlock)

			maxBigTxPerBlock := new(big.Int).Div(smiloPay, bigTxPrice)
			t.Log("MaxBigTx/block : ", maxBigTxPerBlock) // 20 Gwei * 21000 Gas
			require.Equal(t, test.maxBigTxPerBlock, maxBigTxPerBlock)

			blocksTillFull := new(big.Int).Div(maxSmiloPay, smiloPay)
			t.Log("Blocks till full : ", blocksTillFull)
			require.Equal(t, test.blocksTillFull, blocksTillFull)

		})
	}
}
