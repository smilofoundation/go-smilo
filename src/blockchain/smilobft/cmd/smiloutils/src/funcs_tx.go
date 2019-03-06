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
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"go-smilo/src/blockchain/smilobft/core/types"
)

// NewTransactionSigned creates a new transaction and signs it
func NewTransactionSigned(fromAddress common.Address, toAddress *common.Address, amount *big.Int, data []byte) (signedTx *types.Transaction, err error) {
	tx, err := NewTransaction(fromAddress, toAddress, amount, gasLimit, data)
	if err != nil {
		err = fmt.Errorf("failed to createSignedTransaction, createTransaction transaction: %v", err)
		return tx, err
	}

	key, err := crypto.HexToECDSA(privatekey)
	if err != nil {
		fmt.Println("crypto.HexToECDSA, Invalid private key", err)
		return nil, err
	}

	keyAddr := crypto.PubkeyToAddress(key.PublicKey)
	if fromAddress != keyAddr {
		fmt.Println("crypto.PubkeyToAddress, The address of this PK does not match with the desired fromAddress")
		return nil, errors.New("not correct PK for the account")
	}

	signer := types.NewEIP155Signer(chainID)

	signedTx, err = types.SignTx(tx, signer, key)
	if err != nil {
		err = fmt.Errorf("failed to createSignedTransaction, signTransaction : %v", err)
		return tx, err
	}

	GetNextNonce(fromAddress)

	return signedTx, err
}

// NewTransaction will create a unsigned transaction
func NewTransaction(fromAddress common.Address, toAddress *common.Address, amount *big.Int, gasLimit uint64, data []byte) (tx *types.Transaction, err error) {
	var txNonce uint64
	txNonce, err = PendingNonceAt(fromAddress)
	if err != nil {
		fmt.Println("Error: createTransaction, currentNonce, ", err)
		return tx, err
	}

	if gasLimit == 0 {
		gasLimit, err = EstimateGas(fromAddress, toAddress, amount, data)
		if err != nil {
			fmt.Println("Error: createTransaction, estimateGas, ", err)
			return tx, err
		}
	}

	if toAddress == nil {
		tx = types.NewContractCreation(txNonce, amount, gasLimit, gasPrice, data)
	} else {
		tx = types.NewTransaction(txNonce, *toAddress, amount, gasLimit, gasPrice, data)
	}

	return tx, err
}

func TxFrom(tx *types.Transaction) (address common.Address, err error) {
	V, _, _ := tx.RawSignatureValues()
	signer := deriveSigner(V)
	address, err = types.Sender(signer, tx)
	return
}
