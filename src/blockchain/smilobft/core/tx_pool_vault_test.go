package core

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"go-smilo/src/blockchain/smilobft/core/types"
)

func createVaultTx(value, gasPrice *big.Int, data []byte, key *ecdsa.PrivateKey) (*types.Transaction, *big.Int, common.Address) {
	defaultTxPoolGasLimit := uint64(1000000)
	newTx, _ := types.SignTx(types.NewTransaction(0, common.Address{}, value, defaultTxPoolGasLimit, gasPrice, data), types.HomesteadSigner{}, key)
	newTx.SetPrivate()
	balance := new(big.Int).Add(newTx.Value(), new(big.Int).Mul(new(big.Int).SetUint64(newTx.Gas()), newTx.GasPrice()))
	from, _ := deriveSender(newTx)
	return newTx, balance, from
}

func TestVaultTransactions(t *testing.T) {

	testCases := []struct {
		name          string
		IsPrivate       bool
		value         *big.Int
		gasPrice      *big.Int
		data          []byte
		addBalance    *big.Int
		expectedError error
	}{
		{
			name:          "vault signed transfer of value 0 is not allowed due to double spending issues",
			IsPrivate:       true,
			value:         common.Big0,
			gasPrice:      common.Big0,
			data:          nil,
			addBalance:    big.NewInt(0),
			expectedError: ErrEtherValueUnsupported,
		},
		{
			name:          "vault signed transfer of value 3 is not allowed due to double spending issues",
			IsPrivate:       true,
			value:         common.Big3,
			gasPrice:      common.Big0,
			data:          nil,
			addBalance:    big.NewInt(0),
			expectedError: ErrEtherValueUnsupported,
		},
	}

	for _, test := range testCases {

		t.Run(test.name, func(t *testing.T) {

			pool, key := setupSmiloTxPool()
			defer pool.Stop()

			newTX, balance, from := createVaultTx(test.value, test.gasPrice, test.data, key)
			if test.IsPrivate {
				newTX.SetPrivate()
			}

			if test.addBalance != nil {
				pool.currentState.AddBalance(from, balance, test.addBalance)
			}

			err := pool.AddRemote(newTX)
			require.Equal(t, test.expectedError, err)

		})
	}

}

func TestValidateTx_whenValueNonZeroTransferForVaultTransaction(t *testing.T) {
	pool, key := setupSmiloTxPool()
	defer pool.Stop()
	arbitraryValue := common.Big3
	arbitraryTx, balance, from := createVaultTx(arbitraryValue, common.Big0, nil, key)
	pool.currentState.AddBalance(from, balance, big.NewInt(0))

	if err := pool.AddRemote(arbitraryTx); err != ErrEtherValueUnsupported {
		t.Error("expected: ", ErrEtherValueUnsupported, "; got:", err)
	}
}

func TestValidateTx_whenValueNonZeroWithSmartContractForVaultTransaction(t *testing.T) {
	pool, key := setupSmiloTxPool()
	defer pool.Stop()
	arbitraryValue := common.Big3
	arbitraryTx, balance, from := createVaultTx(arbitraryValue, common.Big0, []byte("arbitrary bytecode"), key)
	pool.currentState.AddBalance(from, balance, big.NewInt(0))

	if err := pool.AddRemote(arbitraryTx); err != ErrEtherValueUnsupported {
		t.Error("expected: ", ErrEtherValueUnsupported, "; got:", err)
	}
}
