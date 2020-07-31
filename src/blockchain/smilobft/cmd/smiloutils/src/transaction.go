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
	"go-smilo/src/blockchain/smilobft/accounts/abi/bind"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/orinocopay/go-etherutils"
	"gopkg.in/urfave/cli.v1"

	"go-smilo/src/blockchain/smilobft/core/types"
	"go-smilo/src/blockchain/smilobft/ethclient"
)

func GetValidTXGasPrice(ctx *cli.Context) (validTX *types.Transaction, gasprice int64, err error) {
	transaction := ctx.String(transactionFlag.Name)
	if len(transaction) == 0 {
		return validTX, gasprice, cli.NewExitError("transaction is required", 1)
	}

	connection := ctx.String(connectionFlag.Name)
	if len(connection) == 0 {
		return validTX, gasprice, cli.NewExitError("connection is required", 1)
	}
	gasprice = ctx.Int64(gaspriceFlag.Name)

	privatekey = ctx.String(privatekeyFlag.Name)

	fmt.Println("Will cancel the transaction ", "transaction", transaction, "connection", connection)

	client, err = ethclient.Dial(connection)
	if err != nil {
		return validTX, gasprice, cli.NewExitError("Could not dial to Smilo node", 1)
	}

	thisctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	chainID, err = client.NetworkID(thisctx)
	if err != nil {
		return validTX, gasprice, cli.NewExitError("Could not get NetworkID of Smilo node", 1)
	}

	fmt.Println("Connection to Smilo node OK", "chainID", chainID, "transaction", transaction, "connection", connection)

	tx := common.HexToHash(transaction)
	thisctx, cancel = LocalContext()
	defer cancel()

	avalidTX, pending, err := client.TransactionByHash(thisctx, tx)

	if avalidTX == nil || avalidTX.Hash() != tx || err != nil {
		fmt.Println("failed to obtain transaction", tx.Hex(), err)
		return validTX, gasprice, cli.NewExitError("Failed to obtain transaction", 1)
	} else if !pending {
		fmt.Printf("Transaction %s has already been mined \n", tx.Hex())
		return validTX, gasprice, cli.NewExitError("Transaction %s has already been mined", 1)
	}

	//set it back
	validTX = avalidTX

	fmt.Println("validTX, ", validTX)

	return validTX, gasprice, nil
}

func ProcessValidTXAndGas(validTX *types.Transaction, gasprice int64, minGasPrice *big.Int, cmdStr string) error {
	if gasprice == 0 {
		// No gas price supplied; use the calculated minimum
		gasPrice = minGasPrice
	} else {
		gasPrice = big.NewInt(gasprice)
		// Gas price supplied; ensure it is at least 10% more than the current gas price
		if gasPrice.Cmp(minGasPrice) >= 0 {
			fmt.Printf("Gas price must be at least %s", etherutils.WeiToString(minGasPrice, true))
			return cli.NewExitError(fmt.Sprintf("Gas price must be at least %s", etherutils.WeiToString(minGasPrice, true)), 1)
		}
	}

	// Create and sign the transaction
	fromAddress, err := TxFrom(validTX)
	if err != nil {
		fmt.Println("Failed to obtain from address")
		return cli.NewExitError("Failed to obtain from address", 1)

	}

	nonce = int64(validTX.Nonce())

	signedTx, err := NewTransactionSigned(fromAddress, &fromAddress, nil, nil)
	if err != nil {
		fmt.Printf("Failed to createSignedTransaction %s", validTX.Hash().Hex())
		return cli.NewExitError("Failed to createSignedTransaction", 1)
	}

	thisctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	fmt.Println("signedTx, ", signedTx)
	err = client.SendTransaction(thisctx, signedTx, bind.PrivateTxArgs{})
	if err != nil {
		fmt.Println("Failed to send transaction, ", validTX.Hash().Hex(), err)
		return cli.NewExitError("Signed transaction failed SendTransaction, ", 1)
	}

	fmt.Println("cmd", "transaction",
		"Command", cmdStr,
		"Address", fromAddress.Hex(),
		"NetworkID", chainID,
		"Gas", signedTx.Gas(),
		"Gas Price", signedTx.GasPrice().String(),
		"Transaction Hash", signedTx.Hash().Hex(),
	)

	return nil
}

// CancelTransaction will cancel a transaction based on the tx and pk
func CancelTransaction(ctx *cli.Context) error {

	validTX, gasprice, err := GetValidTXGasPrice(ctx)
	if err != nil {
		return cli.NewExitError("Could not get GetStuff with Smilo node", 1)
	}

	minGasPrice := big.NewInt(0).Add(big.NewInt(0).Add(validTX.GasPrice(), big.NewInt(0).Div(validTX.GasPrice(), big.NewInt(10))), big.NewInt(10))

	err = ProcessValidTXAndGas(validTX, gasprice, minGasPrice, "cancel")
	if err != nil {
		return cli.NewExitError("Could not get DoMoreSuff with Smilo node", 1)
	}

	return nil
}

// UpTransaction will up the gas for transaction based on the tx and pk
func UpTransaction(ctx *cli.Context) error {

	validTX, gasprice, err := GetValidTXGasPrice(ctx)
	if err != nil {
		return cli.NewExitError("Could not get GetValidTXGasPrice with Smilo node", 1)
	}

	minGasPrice := big.NewInt(0).Add(big.NewInt(0).Add(validTX.GasPrice(), big.NewInt(0).Div(validTX.GasPrice(), big.NewInt(10))), big.NewInt(10))

	err = ProcessValidTXAndGas(validTX, gasprice, minGasPrice, "up")
	if err != nil {
		return cli.NewExitError("Could not ProcessValidTXAndGas with Smilo node", 1)
	}

	return nil
}
