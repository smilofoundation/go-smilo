// Copyright 2019 The go-smilo Authors
// Copyright 2015 The go-ethereum Authors
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

// Package ethapi implements the general Ethereum API functions.
package ethapi

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"go-smilo/src/blockchain/smilobft/rpc"

	"go-smilo/src/blockchain/smilobft/core/vm"
	"go-smilo/src/blockchain/smilobft/vault"
)

// GetSmiloPayload returns the contents of a private transaction
func (s *PublicBlockChainAPI) GetSmiloPayload(digestHex string) (string, error) {
	if vault.VaultInstance == nil {
		return "", fmt.Errorf("vault is not enabled")
	}
	if len(digestHex) < 3 {
		return "", fmt.Errorf("invalid digest hex, len=%d", len(digestHex))
	}
	if digestHex[:2] == "0x" {
		digestHex = digestHex[2:]
	}
	b, err := hex.DecodeString(digestHex)
	if err != nil {
		return "", err
	}
	if len(b) != 64 {
		return "", fmt.Errorf("expected a Smilo digest of length 64, but got %d", len(b))
	}
	data, err := vault.VaultInstance.Get(b)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("0x%x", data), nil
}

// GetSmiloPay returns the smiloPay from the given address and blockNum
func (s *PublicBlockChainAPI) GetSmiloPay(ctx context.Context, address common.Address, blockNum rpc.BlockNumber) (*big.Int, error) {
	state, _, err := s.b.StateAndHeaderByNumber(ctx, blockNum)
	if state == nil || err != nil {
		return nil, err
	}
	header, err := s.b.HeaderByNumber(context.Background(), rpc.LatestBlockNumber)
	if err != nil {
		log.Error("PublicBlockChainAPI.GetSmiloPay, ", address, blockNum, err)
		return nil, err
	}
	b := state.GetSmiloPay(address, header.Number)
	return b, nil
}

// SendVaultTransaction will POST data to local blackbox node if data is valid; used by PublicTransactionPoolAPI.SendTransaction
func SendVaultTransactionWithExtraCheck(args SendTxArgs) (d hexutil.Bytes, err error) {
	if vault.VaultInstance == nil {
		return d, fmt.Errorf("failed to get VaultInstance, is Vault node running ?? ")
	} else if args.Value != nil && args.Value.ToInt().Sign() != 0 {
		return d, vm.ErrReadOnlyValueTransfer
	}
	var data []byte

	if args.Data != nil {
		data = []byte(*args.Data)
	} else {
		log.Info("args.data is nil")
	}

	//Send transaction Blackbox node
	if len(data) > 0 {
		log.Info("sending vault tx", "data", fmt.Sprintf("%x", data), "vaultfrom", args.VaultFrom, "sharedwith", args.SharedWith)
		data, err = vault.VaultInstance.Post(data, args.VaultFrom, args.SharedWith)
		log.Info("sent vault tx", "data", fmt.Sprintf("%x", data), "vaultfrom", args.VaultFrom, "sharedwith", args.SharedWith)
		if err != nil {
			return nil, err
		}
	}
	d = hexutil.Bytes(data)

	return d, nil
}

// SendVaultTransaction will POST data to local blackbox node if data is valid; used by PublicTransactionPoolAPI.SendTransaction
func SendVaultTransaction(args SendTxArgs) (d hexutil.Bytes, err error) {
	if args.Value != nil && args.Value.ToInt().Sign() != 0 {
		return d, vm.ErrReadOnlyValueTransfer
	}

	data := []byte(*args.Data)
	if len(data) > 0 {
		log.Info("sending vault tx", "data", fmt.Sprintf("%x", data), "VaultFrom", args.VaultFrom, "SharedWith", args.SharedWith)
		data, err := vault.VaultInstance.Post(data, args.VaultFrom, args.SharedWith)
		log.Info("sent vault tx", "data", fmt.Sprintf("%x", data), "VaultFrom", args.VaultFrom, "SharedWith", args.SharedWith)
		if err != nil {
			return nil, err
		}
	}

	d = hexutil.Bytes(data)

	return d, nil
}
