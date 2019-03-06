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
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"go-smilo/src/blockchain/smilobft/core/types"

	"gopkg.in/urfave/cli.v1"

	"go-smilo/src/blockchain/smilobft/cmn"
)

var (
	extraDataFlag = cli.StringFlag{
		Name:  "extradata",
		Usage: "Hex string for Sport extraData",
	}

	fullnodesFlag = cli.StringFlag{
		Name:  "fullnodes",
		Usage: "Fullnodes for Sport extraData",
	}

	vanityFlag = cli.StringFlag{
		Name:  "vanity",
		Usage: "Vanity for Sport extraData",
		Value: "0x00",
	}

	stringToHashFlag = cli.StringFlag{
		Name:  "string",
		Usage: "string to be hashed",
		Value: "Smilo, The hybrid blockchain platform with a conscience",
	}

	ExtraCommand = cli.Command{
		Name:  "extra",
		Usage: "Sport extraData",
		Subcommands: []cli.Command{
			{
				Action:    Decode,
				Name:      "decode",
				Usage:     "Decode Sport extraData",
				ArgsUsage: "--extradata <extra data>",
				Flags: []cli.Flag{
					extraDataFlag,
				},
				Description: `Decodes extraData.`,
			},
			{
				Action:    Encode,
				Name:      "encode",
				Usage:     "Encode Sport extraData",
				ArgsUsage: "--fullnodes 0x7cB791430,0x2f65A895 --vanity 0x00",
				Flags: []cli.Flag{
					fullnodesFlag,
					vanityFlag,
				},
				Description: `Encode vanity / fullnodes to extraData.`,
			},
			{
				Action:    MixHash,
				Name:      "mixhash",
				Usage:     "Generate Sport mixhash",
				ArgsUsage: "--string <string>",
				Flags: []cli.Flag{
					stringToHashFlag,
				},
				Description: `Generate sport mixhash`,
			},
		},
	}
)

func Encode(ctx *cli.Context) error {
	fullnodes := ctx.String(fullnodesFlag.Name)
	if len(fullnodes) == 0 {
		return cli.NewExitError("Fullnodes are required", 1)
	}

	extraData, err := GenerateExtraFromFullnodes(ctx.String(vanityFlag.Name), fullnodes)
	if err != nil {
		return cli.NewExitError("Failed generate extraData from fullnodes", 0)
	}
	fmt.Println("Encoded Sport extra-data:", extraData)
	return nil
}

func MixHash(ctx *cli.Context) error {
	targetMixhash := ctx.String(stringToHashFlag.Name)
	if len(targetMixhash) == 0 {
		return cli.NewExitError("A valid string is required", 1)
	}

	mixHash := cmn.StringToHash(targetMixhash)
	fmt.Println("Generated Sport mixhash:", mixHash.Hex())
	return nil
}

func Decode(ctx *cli.Context) error {
	if !ctx.IsSet(extraDataFlag.Name) {
		return cli.NewExitError("extraData is required", 1)
	}

	extraString := ctx.String(extraDataFlag.Name)

	extra, err := hexutil.Decode(extraString)
	if err != nil {
		log.Error("Decode, hexutil.Decode error %v", err)
		return err
	}

	smiloExtra, err := types.ExtractSportExtra(&types.Header{Extra: extra})
	if err != nil {
		log.Error("Decode, ExtractSportExtra error %v", err)
		return err
	}
	vanity, smiloExtra, err := extra[:types.SportExtraVanity], smiloExtra, nil
	if err != nil {
		return err
	}

	fmt.Println("vanity: ", "0x"+common.Bytes2Hex(vanity))

	for _, v := range smiloExtra.Fullnodes {
		fmt.Println("fullnode: ", v.Hex())
	}

	if len(smiloExtra.Seal) != 0 {
		fmt.Println("seal:", "0x"+common.Bytes2Hex(smiloExtra.Seal))
	}

	for _, seal := range smiloExtra.CommittedSeal {
		fmt.Println("committed seal: ", "0x"+common.Bytes2Hex(seal))
	}

	return nil
}

func GenerateExtraFromFullnodes(vanity string, fullnodesStr string) (string, error) {
	result := strings.Split(fullnodesStr, ",")
	for i, r := range result {
		result[i] = strings.TrimSpace(r)
	}

	fullnodes := make([]common.Address, len(result))
	for i, v := range result {
		fullnodes[i] = common.HexToAddress(v)
	}

	newVanity, err := hexutil.Decode(vanity)
	if err != nil {
		return "", err
	}

	if len(newVanity) < types.SportExtraVanity {
		newVanity = append(newVanity, bytes.Repeat([]byte{0x00}, types.SportExtraVanity-len(newVanity))...)
	}
	newVanity = newVanity[:types.SportExtraVanity]

	ist := &types.SportExtra{
		Fullnodes:     fullnodes,
		Seal:          make([]byte, types.SportExtraSeal),
		CommittedSeal: [][]byte{},
	}

	payload, err := rlp.EncodeToBytes(&ist)
	if err != nil {
		return "", err
	}

	return "0x" + common.Bytes2Hex(append(newVanity, payload...)), nil

}

func main() {
	app := cli.NewApp()
	app.Name = "extradata"
	app.Usage = "The Smilo ExtraData command line interface"

	app.Commands = []cli.Command{
		ExtraCommand,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
