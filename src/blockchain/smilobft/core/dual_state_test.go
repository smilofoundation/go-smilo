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

package core

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"

	"go-smilo/src/blockchain/smilobft/core/state"
	"go-smilo/src/blockchain/smilobft/core/types"
	"go-smilo/src/blockchain/smilobft/core/vm"
	"go-smilo/src/blockchain/smilobft/ethdb"
	"go-smilo/src/blockchain/smilobft/params"
)

var dualStateTestHeader = types.Header{
	Number:     new(big.Int),
	Time:       new(big.Int).SetUint64(43),
	Difficulty: new(big.Int).SetUint64(1000488),
	GasLimit:   4700000,
}

//[1] PUSH1 0x01 (out size)
//[3] PUSH1 0x00 (out offset)
//[5] PUSH1 0x00 (in size)
//[7] PUSH1 0x00 (in offset)
//[9] PUSH1 0x00 (value)
//[30] PUSH20 0x0200000000000000000000000000000000000000 (to)
//[34] PUSH3 0x0186a0 (gas)
//[35] CALL
//[37] PUSH1 0x00
//[38] MLOAD
//[40] PUSH1 0x00
//[41] SSTORE
//[42] STOP

func TestDualState(t *testing.T) {

	testCases := []struct {
		name              string
		firstStateCode    string
		secondStateCode   string
		callMessage       callmsg
		expectedHash      common.Hash
		expectedStateAddr common.Address
		expectedState     string

		firstState  string
		secondState string
	}{
		{
			name:            "test dual state vault to public",
			firstStateCode:  "600a6000526001601ff300",
			secondStateCode: "60016000600060006000730200000000000000000000000000000000000000620186a0f160005160005500",
			callMessage: callmsg{
				addr:     common.Address{},
				to:       &common.Address{1},
				value:    big.NewInt(1),
				gas:      1000000,
				gasPrice: new(big.Int),
				data:     nil,
			},
			expectedState:     "vault",
			expectedHash:      common.Hash{10},
			expectedStateAddr: common.Address{1},

			firstState:  "public",
			secondState: "vault",
		},
		{
			name:            "test dual state public to vault",
			firstStateCode:  "600a6000526001601ff300",
			secondStateCode: "60016000600060006000730200000000000000000000000000000000000000620186a0f160005160005500",
			callMessage: callmsg{
				addr:     common.Address{},
				to:       &common.Address{1},
				value:    big.NewInt(1),
				gas:      1000000,
				gasPrice: new(big.Int),
				data:     nil,
			},
			expectedState:     "public",
			expectedHash:      common.Hash{},
			expectedStateAddr: common.Address{1},

			firstState:  "vault",
			secondState: "public",
		},
		{
			name:            "test dual state read only",
			firstStateCode:  "600a60005500",
			secondStateCode: "60016000600060006000730200000000000000000000000000000000000000620186a0f160005160005500",
			callMessage: callmsg{
				addr:     common.Address{},
				to:       &common.Address{1},
				value:    big.NewInt(1),
				gas:      1000000,
				gasPrice: new(big.Int),
				data:     nil,
			},
			expectedState:     "public",
			expectedHash:      common.Hash{0},
			expectedStateAddr: common.Address{2},

			firstState:  "public",
			secondState: "vault",
		},
	}

	for _, test := range testCases {

		t.Run(test.name, func(t *testing.T) {

			callAddr := common.Address{1}

			db := ethdb.NewMemDatabase()

			vaultState, _ := state.New(common.Hash{}, state.NewDatabase(db))
			publicState, _ := state.New(common.Hash{}, state.NewDatabase(db))

			if test.firstState == "vault" {
				vaultState.SetCode(common.Address{2}, common.Hex2Bytes(test.firstStateCode))
			} else if test.firstState == "public" {
				publicState.SetCode(common.Address{2}, common.Hex2Bytes(test.firstStateCode))
			}

			if test.secondState == "vault" {
				vaultState.SetCode(*test.callMessage.to, common.Hex2Bytes(test.secondStateCode))
			} else if test.secondState == "public" {
				publicState.SetCode(*test.callMessage.to, common.Hex2Bytes(test.secondStateCode))
			}

			author := common.Address{}
			msg := test.callMessage

			ctx := NewEVMContext(msg, &dualStateTestHeader, nil, &author)
			env := vm.NewEVM(ctx, publicState, vaultState, &params.ChainConfig{}, vm.Config{})
			env.Call(vm.AccountRef(author), callAddr, msg.data, msg.gas, new(big.Int), true)

			if test.expectedState == "vault" {
				value := vaultState.GetState(test.expectedStateAddr, common.Hash{})
				require.Equal(t, test.expectedHash, value)
			} else if test.expectedState == "public" {
				value := publicState.GetState(test.expectedStateAddr, common.Hash{})
				require.Equal(t, test.expectedHash, value)
			}

		})
	}
}

var (
	calleeAddress      = common.Address{2}
	calleeContractCode = "600a6000526001601ff300" // a function that returns 10
	callerAddress      = common.Address{1}
	// a functionn that calls the callee's function at its address and return the same value
	//000000: PUSH1 0x01
	//000002: PUSH1 0x00
	//000004: PUSH1 0x00
	//000006: PUSH1 0x00
	//000008: PUSH20 0x0200000000000000000000000000000000000000
	//000029: PUSH3 0x0186a0
	//000033: STATICCALL
	//000034: PUSH1 0x01
	//000036: PUSH1 0x00
	//000038: RETURN
	//000039: STOP
	callerContractCode = "6001600060006000730200000000000000000000000000000000000000620186a0fa60016000f300"
)

func verifyStaticCall(t *testing.T, privateState *state.StateDB, publicState *state.StateDB, expectedHash common.Hash) {
	author := common.Address{}
	msg := callmsg{
		addr:     author,
		to:       &callerAddress,
		value:    big.NewInt(1),
		gas:      1000000,
		gasPrice: new(big.Int),
		data:     nil,
	}

	ctx := NewEVMContext(msg, &dualStateTestHeader, nil, &author)
	env := vm.NewEVM(ctx, publicState, privateState, &params.ChainConfig{
		ByzantiumBlock: new(big.Int),
	}, vm.Config{})

	ret, _, err := env.Call(vm.AccountRef(author), callerAddress, msg.data, msg.gas, new(big.Int), true)

	if err != nil {
		t.Fatalf("Call error: %s", err)
	}
	value := common.Hash{ret[0]}
	if value != expectedHash {
		t.Errorf("expected %x got %x", expectedHash, value)
	}
}

func TestStaticCall_whenPublicToPublic(t *testing.T) {
	db := ethdb.NewMemDatabase()

	publicState, _ := state.New(common.Hash{}, state.NewDatabase(db))
	publicState.SetCode(callerAddress, common.Hex2Bytes(callerContractCode))
	publicState.SetCode(calleeAddress, common.Hex2Bytes(calleeContractCode))

	verifyStaticCall(t, publicState, publicState, common.Hash{10})
}

func TestStaticCall_whenPublicToPrivateInTheParty(t *testing.T) {
	db := ethdb.NewMemDatabase()

	privateState, _ := state.New(common.Hash{}, state.NewDatabase(db))
	privateState.SetCode(calleeAddress, common.Hex2Bytes(calleeContractCode))

	publicState, _ := state.New(common.Hash{}, state.NewDatabase(db))
	publicState.SetCode(callerAddress, common.Hex2Bytes(callerContractCode))

	verifyStaticCall(t, privateState, publicState, common.Hash{10})
}

func TestStaticCall_whenPublicToPrivateNotInTheParty(t *testing.T) {

	db := ethdb.NewMemDatabase()

	privateState, _ := state.New(common.Hash{}, state.NewDatabase(db))

	publicState, _ := state.New(common.Hash{}, state.NewDatabase(db))
	publicState.SetCode(callerAddress, common.Hex2Bytes(callerContractCode))

	verifyStaticCall(t, privateState, publicState, common.Hash{0})
}

func TestStaticCall_whenPrivateToPublic(t *testing.T) {
	db := ethdb.NewMemDatabase()

	privateState, _ := state.New(common.Hash{}, state.NewDatabase(db))
	privateState.SetCode(callerAddress, common.Hex2Bytes(callerContractCode))

	publicState, _ := state.New(common.Hash{}, state.NewDatabase(db))
	publicState.SetCode(calleeAddress, common.Hex2Bytes(calleeContractCode))

	verifyStaticCall(t, privateState, publicState, common.Hash{10})
}
