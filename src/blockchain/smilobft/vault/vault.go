// Copyright 2019 The go-smilo Authors
// Copyright 2016 The go-ethereum Authors
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

package vault

import (
	"os"

	"github.com/ethereum/go-ethereum/log"

	"go-smilo/src/blockchain/smilobft/vault/blackbox"
)

func GetBlackboxVault(targetIPC string) BlackboxVault {
	log.Debug("################ GetBlackboxVault, ", "targetIPC", targetIPC)
	config := os.Getenv(targetIPC)
	if config != "" {
		return blackbox.CreateNew(config)
	} else {
		log.Error("################ ERROR: GetBlackboxVault, targetIPC is blank ???, ", "targetIPC", targetIPC)
	}
	return nil
}

var VaultInstance = GetBlackboxVault("VAULT_IPC")
