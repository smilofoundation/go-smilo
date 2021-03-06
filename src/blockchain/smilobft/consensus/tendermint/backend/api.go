// Copyright 2017 The go-ethereum Authors
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

package backend

import (
	"go-smilo/src/blockchain/smilobft/core/types"

	"github.com/ethereum/go-ethereum/common"

	"go-smilo/src/blockchain/smilobft/consensus"
	"go-smilo/src/blockchain/smilobft/consensus/tendermint/core"
	"go-smilo/src/blockchain/smilobft/rpc"
)

// API is a user facing RPC API to dump BFT state
type API struct {
	chain      consensus.ChainReader
	tendermint core.Backend
}

// GetCommittee retrieves the list of authorized committee at the specified block.
func (api *API) GetCommittee(number *rpc.BlockNumber) (types.Committee, error) {
	committeeSet, err := api.tendermint.Committee(uint64(*number))
	if err != nil {
		return nil, err
	}
	return committeeSet.Committee(), nil
}

// GetCommitteeAtHash retrieves the state snapshot at a given block.
func (api *API) GetCommitteeAtHash(hash common.Hash) (types.Committee, error) {
	header := api.chain.GetHeaderByHash(hash)
	if header == nil {
		return nil, errUnknownBlock
	}
	committeeSet, err := api.tendermint.Committee(header.Number.Uint64())
	if err != nil {
		return nil, err
	}
	return committeeSet.Committee(), nil
}

// Get Autonity contract address
func (api *API) GetContractAddress() common.Address {
	return api.tendermint.GetContractAddress()
}

// Get Autonity contract ABI
func (api *API) GetContractABI() string {
	return api.tendermint.GetContractABI()
}

// Get current white list
func (api *API) GetWhitelist() []string {
	return api.tendermint.WhiteList()
}
