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

package main

import (
	"fmt"
	"os"

	"gopkg.in/urfave/cli.v1"

	"go-smilo/src/blockchain/smilobft/cmd/smiloutils/src"
)

func main() {
	app := cli.NewApp()
	app.Name = "Smilo Utils CMD"
	app.Usage = "The Smilo Utils command line interface"

	app.Commands = []cli.Command{
		src.TransactionCommand,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
