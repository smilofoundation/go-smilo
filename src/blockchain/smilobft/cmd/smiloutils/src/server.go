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
	"math/big"
	"time"

	"gopkg.in/urfave/cli.v1"

	"go-smilo/src/blockchain/smilobft/ethclient"
)

var (
	transactionFlag = cli.StringFlag{
		Name:  "transaction",
		Usage: "Hex string for Sport extraData",
	}

	connectionFlag = cli.StringFlag{
		Name:  "connection",
		Usage: "Fullnodes connection, eg: --connection=http://localhost:22000 ",
	}

	timeoutFlag = cli.IntFlag{
		Name:  "timeout",
		Usage: "timeout in secs",
		Value: 30,
	}

	passphraseFlag = cli.StringFlag{
		Name:  "passphrase",
		Usage: "passphrase",
		Value: "",
	}

	privatekeyFlag = cli.StringFlag{
		Name:  "privatekey",
		Usage: "privatekey",
		Value: "",
	}

	gaspriceFlag = cli.Uint64Flag{
		Name:  "gasprice",
		Usage: "gasprice",
		Value: 0x00,
	}

	TransactionCommand = cli.Command{
		Name:  "transaction",
		Usage: "do things with a transaction",
		Subcommands: []cli.Command{
			{
				Action:    CancelTransaction,
				Name:      "cancel",
				Usage:     "cancel txid",
				ArgsUsage: "<tx id>",
				Flags: []cli.Flag{
					connectionFlag,
					transactionFlag,
					timeoutFlag,
					passphraseFlag,
					privatekeyFlag,
					gaspriceFlag,
				},
				Description: `Cancel a transaction.`,
			},
			{
				Action:    UpTransaction,
				Name:      "up",
				Usage:     "up gas for a txid",
				ArgsUsage: "<tx id>",
				Flags: []cli.Flag{
					connectionFlag,
					transactionFlag,
					timeoutFlag,
					passphraseFlag,
					privatekeyFlag,
					gaspriceFlag,
				},
				Description: `Up a transaction.`,
			},
		},
	}

	client     *ethclient.Client
	chainID    *big.Int
	timeout    = 30 * time.Second
	gasPrice   *big.Int
	gasLimit   uint64
	nonce      int64
	privatekey string
)

func LocalContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}
