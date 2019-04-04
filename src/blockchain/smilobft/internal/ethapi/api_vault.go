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

package ethapi

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"go-smilo/src/blockchain/smilobft/core/types"
	"go-smilo/src/blockchain/smilobft/vault"
)

// SendRawTxArgs represents the arguments to submit a new signed private transaction into the transaction pool.
type VaultSendRawTxArgs struct {
	SharedWith []string `json:"sharedWith"`
}

// SendRawTransactionVault will add the signed transaction to the Vault and to the transaction pool.
// The sender is responsible for signing the transaction and using the correct nonce.
func (s *PublicTransactionPoolAPI) SendRawTransactionVault(ctx context.Context, encodedTx hexutil.Bytes, args VaultSendRawTxArgs) (common.Hash, error) {
	if vault.VaultInstance == nil {
		return common.Hash{}, fmt.Errorf("vault is not enabled")
	}

	tx := new(types.Transaction)
	if err := rlp.DecodeBytes(encodedTx, tx); err != nil {
		return common.Hash{}, err
	}

	data := tx.Data()
	isVault := args.SharedWith != nil

	if isVault {
		if len(data) > 0 {
			log.Info("sending vault tx", "data", fmt.Sprintf("%x", data), "vaultfrom", args.SharedWith, "sharedwith", args.SharedWith)
			data, err := vault.VaultInstance.PostRawTransaction(data, args.SharedWith)
			log.Info("sent vault tx", "data", fmt.Sprintf("%x", data), "vaultfrom", args.SharedWith, "sharedwith", args.SharedWith)

			if err != nil {
				return common.Hash{}, err
			}
		}
	} else {
		return common.Hash{}, fmt.Errorf("transaction is not vault type")
	}

	return submitTransaction(ctx, s.b, tx, isVault)
}

// Get the Vault Transaction content
func (s *PublicBlockChainAPI) GetVaultTransaction(digestHex string) (data string, err error) {
	if vault.VaultInstance == nil {
		err = fmt.Errorf("vault is not enabled")
		return data, err
	}

	if len(digestHex) < 3 {
		err = fmt.Errorf("invalid digest hex")
		return data, err
	}
	if digestHex[:2] == "0x" {
		digestHex = digestHex[2:]
	}
	var b []byte
	b, err = hex.DecodeString(digestHex)
	if err != nil {
		return data, err
	}
	if len(b) != 64 {
		err = fmt.Errorf("expected a digest of length 64, but got %d", len(b))
		return data, err
	}
	var responseData []byte
	responseData, err = vault.VaultInstance.Get(b)
	if err != nil {
		return data, err
	}
	data = fmt.Sprintf("0x%x", responseData)
	return data, nil
}
