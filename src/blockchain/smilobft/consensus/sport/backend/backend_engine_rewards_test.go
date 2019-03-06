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

package backend

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBlockRewards(t *testing.T) {

	testCases := []struct {
		name           string
		blockNum       *big.Int
		expectedReward *big.Int
	}{
		{
			name:           "test block 0 start",
			blockNum:       big.NewInt(0),
			expectedReward: big.NewInt(4e18),
		},
		{
			name:           "test block 20000000-1 end",
			blockNum:       big.NewInt(20000000 - 1),
			expectedReward: big.NewInt(4e18),
		},

		{
			name:           "test block 20000000 start",
			blockNum:       big.NewInt(20000000),
			expectedReward: big.NewInt(2e18),
		},
		{
			name:           "test block 40000000-1 end",
			blockNum:       big.NewInt(40000000 - 1),
			expectedReward: big.NewInt(2e18),
		},

		{
			name:           "test block 40000000 start",
			blockNum:       big.NewInt(40000000),
			expectedReward: big.NewInt(175e16),
		},
		{
			name:           "test block 60000000-1 end",
			blockNum:       big.NewInt(60000000 - 1),
			expectedReward: big.NewInt(175e16),
		},

		{
			name:           "test block 60000000 start",
			blockNum:       big.NewInt(60000000),
			expectedReward: big.NewInt(150e16),
		},
		{
			name:           "test block 80000000-1 end",
			blockNum:       big.NewInt(80000000 - 1),
			expectedReward: big.NewInt(150e16),
		},

		{
			name:           "test block 80000000 start",
			blockNum:       big.NewInt(80000000),
			expectedReward: big.NewInt(125e16),
		},
		{
			name:           "test block 100000000-1 end",
			blockNum:       big.NewInt(100000000 - 1),
			expectedReward: big.NewInt(125e16),
		},

		{
			name:           "test block 100000000 start",
			blockNum:       big.NewInt(100000000),
			expectedReward: big.NewInt(100e16),
		},
		{
			name:           "test block 120000000-1 end",
			blockNum:       big.NewInt(120000000 - 1),
			expectedReward: big.NewInt(100e16),
		},

		{
			name:           "test block 120000000 start",
			blockNum:       big.NewInt(120000000),
			expectedReward: big.NewInt(80e16),
		},
		{
			name:           "test block 140000000-1 end",
			blockNum:       big.NewInt(140000000 - 1),
			expectedReward: big.NewInt(80e16),
		},

		{
			name:           "test block 140000000 start",
			blockNum:       big.NewInt(140000000),
			expectedReward: big.NewInt(60e16),
		},
		{
			name:           "test block 160000000-1 end",
			blockNum:       big.NewInt(160000000 - 1),
			expectedReward: big.NewInt(60e16),
		},

		{
			name:           "test block 160000000 start",
			blockNum:       big.NewInt(160000000),
			expectedReward: big.NewInt(40e16),
		},
		{
			name:           "test block 180000000-1 end",
			blockNum:       big.NewInt(180000000 - 1),
			expectedReward: big.NewInt(40e16),
		},

		{
			name:           "test block 180000000 start",
			blockNum:       big.NewInt(180000000),
			expectedReward: big.NewInt(20e16),
		},
		{
			name:           "test block 200000000-1 end",
			blockNum:       big.NewInt(200000000 - 1),
			expectedReward: big.NewInt(20e16),
		},

		{
			name:           "test block 200000000 start",
			blockNum:       big.NewInt(200000000),
			expectedReward: big.NewInt(10e16),
		},
		{
			name:           "test block 400000000-1 end",
			blockNum:       big.NewInt(400000000 - 1),
			expectedReward: big.NewInt(10e16),
		},

		{
			name:           "test block 400000000 start",
			blockNum:       big.NewInt(400000000),
			expectedReward: big.NewInt(5e16),
		},
		{
			name:           "test block 800000000-1 end",
			blockNum:       big.NewInt(800000000 - 1),
			expectedReward: big.NewInt(5e16),
		},

		{
			name:           "test block 800000000 start",
			blockNum:       big.NewInt(800000000),
			expectedReward: big.NewInt(25e15),
		},
		{
			name:           "test block 1600000000-1 end",
			blockNum:       big.NewInt(1600000000 - 1),
			expectedReward: big.NewInt(25e15),
		},

		{
			name:           "test block 1600000000 start",
			blockNum:       big.NewInt(1600000000),
			expectedReward: big.NewInt(0),
		},
		{
			name:           "test block 2000000000-1 end",
			blockNum:       big.NewInt(2000000000 - 1),
			expectedReward: big.NewInt(0),
		},
	}

	for _, test := range testCases {

		t.Run(test.name, func(t *testing.T) {
			reward := getSmiloBlockReward(test.blockNum)
			//fmt.Println("reward", reward.Int64(), "blockNum", test.blockNum, "expectedReward", test.expectedReward)
			require.Equal(t, test.expectedReward.Int64(), reward.Int64(), "Failed to get proper reward for block %d ", test.blockNum)

		})
	}

}
