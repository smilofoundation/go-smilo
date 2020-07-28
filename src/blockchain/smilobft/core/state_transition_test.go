// Copyright 2014 The go-ethereum Authors
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

package core

import (
	"fmt"
	"go-smilo/src/blockchain/smilobft/private"
	"math/big"
	"testing"

	"go-smilo/src/blockchain/smilobft/core/rawdb"

	"github.com/stretchr/testify/require"

	"go-smilo/src/blockchain/smilobft/core/state"
	"go-smilo/src/blockchain/smilobft/core/vm"
	"go-smilo/src/blockchain/smilobft/params"

	"github.com/ethereum/go-ethereum/common"
)

func verifyGasPoolCalculation(t *testing.T, pm private.BlackboxVault) {
	saved := private.VaultInstance
	defer func() {
		private.VaultInstance = saved
	}()
	private.VaultInstance = pm

	txGasLimit := uint64(100000)
	gasPool := new(GasPool).AddGas(200000)
	// this payload would give us 25288 intrinsic gas
	arbitraryEncryptedPayload := "4ab80888354582b92ab442a317828386e4bf21ea4a38d1a9183fbb715f199475269d7686939017f4a6b28310d5003ebd8e012eade530b79e157657ce8dd9692a"
	expectedGasPool := new(GasPool).AddGas(174712) // only intrinsic gas is deducted

	db := rawdb.NewMemoryDatabase()
	privateState, _ := state.New(common.Hash{}, state.NewDatabase(db))
	publicState, _ := state.New(common.Hash{}, state.NewDatabase(db))
	msg := privateCallMsg{
		callmsg: callmsg{
			addr:     common.Address{2},
			to:       &common.Address{},
			value:    new(big.Int),
			gas:      txGasLimit,
			gasPrice: big.NewInt(0),
			data:     common.Hex2Bytes(arbitraryEncryptedPayload),
		},
	}
	ctx := NewEVMContext(msg, &dualStateTestHeader, nil, &common.Address{})
	evm := vm.NewEVM(ctx, publicState, privateState, params.SmiloTestChainConfig, vm.Config{})
	arbitraryBalance := big.NewInt(100000000)
	publicState.SetBalance(evm.Coinbase, arbitraryBalance, big.NewInt(1))
	publicState.SetBalance(msg.From(), arbitraryBalance, big.NewInt(1))

	testObject := NewStateTransition(evm, msg, gasPool)

	_, _, failed, err := testObject.TransitionDb()

	require.NoError(t, err)
	require.False(t, failed)

	require.Equal(t, new(big.Int).SetUint64(expectedGasPool.Gas()), new(big.Int).SetUint64(gasPool.Gas()), "gas pool must be calculated correctly")
	require.Equal(t, arbitraryBalance, publicState.GetBalance(evm.Coinbase), "balance must not be changed")
	require.Equal(t, arbitraryBalance, publicState.GetBalance(msg.From()), "balance must not be changed")
}

func TestStateTransitionPrivate(t *testing.T) {

	for _, x := range []struct {
		description   string
		blackboxVault *StubPrivateTransactionManager
	}{
		{
			description: "non party node processing",
			blackboxVault: &StubPrivateTransactionManager{
				responses: map[string][]interface{}{
					"Get": {
						[]byte{},
						nil,
					},
				},
			},
		},
		{
			description: "party node processing",
			blackboxVault: &StubPrivateTransactionManager{
				responses: map[string][]interface{}{
					"Get": {
						common.Hex2Bytes("600a6000526001601ff300"),
						nil,
					},
				},
			},
		},
	} {
		t.Run(x.description, func(t *testing.T) {
			verifyGasPoolCalculation(t, x.blackboxVault)
		})
	}
}

type privateCallMsg struct {
	callmsg
}

func (pm privateCallMsg) IsPrivate() bool { return true }

type StubPrivateTransactionManager struct {
	responses map[string][]interface{}
}

// Post is equivalent to Send in Quorum
func (spm *StubPrivateTransactionManager) Post(data []byte, from string, to []string) ([]byte, error) {
	return nil, fmt.Errorf("to be implemented")
}

// PostRawTransaction is equivalent to SendSignedTx in Quorum
func (spm *StubPrivateTransactionManager) PostRawTransaction(data []byte, to []string) ([]byte, error) {
	return nil, fmt.Errorf("to be implemented")
}

// Get is equivalent to Receive in Quorum
func (spm *StubPrivateTransactionManager) Get(data []byte) ([]byte, error) {
	res := spm.responses["Get"]
	if err, ok := res[1].(error); ok {
		return nil, err
	}
	if ret, ok := res[0].([]byte); ok {
		return ret, nil
	}
	return nil, nil
}
