// Copyright 2019 The go-smilo Authors
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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"

	"go-smilo/src/blockchain/smilobft/core/types"
)

// GetFullnodes return a array of fullnodes for a block
func (api *API) GetFullnodes(blockNum *rpc.BlockNumber) (address []common.Address, err error) {
	if blockNum == nil {
		return nil, errUnknownBlock
	}
	// get header (latest or actual)
	var actualHeader *types.Header
	if blockNum != nil || *blockNum != rpc.LatestBlockNumber {
		actualHeader = api.chain.GetHeaderByNumber(uint64(blockNum.Int64()))
	} else {
		actualHeader = api.chain.CurrentHeader()
	}
	//validate header
	if actualHeader == nil {
		return nil, errUnknownBlock
	}
	//create snapshot from header
	snap, err := api.smilo.snapshot(api.chain, actualHeader.Number.Uint64(), actualHeader.Hash(), nil)
	if err != nil {
		return nil, err
	}
	return snap.fullnodes(), nil
}

// GetFullnodesByHash return a array of fullnodes for a bash
func (api *API) GetFullnodesByHash(hash common.Hash) ([]common.Address, error) {
	if (hash == common.Hash{}) {
		return nil, errUnknownBlock
	}
	actualHeader := api.chain.GetHeaderByHash(hash)
	if actualHeader == nil {
		return nil, errUnknownBlock
	}
	blockNum := actualHeader.Number.Uint64()
	h := actualHeader.Hash()
	snap, err := api.smilo.snapshot(api.chain, blockNum, h, nil)
	if err != nil {
		return nil, err
	}
	return snap.fullnodes(), nil
}
