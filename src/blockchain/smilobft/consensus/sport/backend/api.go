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
	"github.com/ethereum/go-ethereum/log"

	"go-smilo/src/blockchain/smilobft/rpc"

	"math/big"

	"go-smilo/src/blockchain/smilobft/consensus"
	"go-smilo/src/blockchain/smilobft/core"
	"go-smilo/src/blockchain/smilobft/core/types"
)

// API is a user facing RPC API to dump smilobft state
type API struct {
	chain consensus.ChainReader
	smilo *backend
}

// GetSnapshot (clique override) retrieves the state snapshot at a given block.
func (api *API) GetSnapshot(number *rpc.BlockNumber) (*Snapshot, error) {
	// Retrieve the requested block number (or current if none requested)
	var header *types.Header
	if number == nil || *number == rpc.LatestBlockNumber {
		header = api.chain.CurrentHeader()
	} else {
		header = api.chain.GetHeaderByNumber(uint64(number.Int64()))
	}
	// Ensure we have an actually valid block and return its snapshot
	if header == nil {
		return nil, errUnknownBlock
	}
	return api.smilo.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil)
}

// GetSnapshotAtHash (clique override) retrieves the state snapshot at a given block.
func (api *API) GetSnapshotAtHash(hash common.Hash) (*Snapshot, error) {
	header := api.chain.GetHeaderByHash(hash)
	if header == nil {
		return nil, errUnknownBlock
	}
	return api.smilo.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil)
}

// Proposals (clique override) return a array of candidates that aim to become full-nodes (proposals)
func (api *API) Proposals() map[common.Address]bool {
	api.smilo.candidatesLock.RLock()
	defer api.smilo.candidatesLock.RUnlock()

	candidateToFullnodeArray := make(map[common.Address]bool)
	for addr, candidate := range api.smilo.candidates {
		candidateToFullnodeArray[addr] = candidate
	}
	return candidateToFullnodeArray
}

// Propose (clique override) injects a new authorization candidate that the fullnode will attempt to push through.
func (api *API) Propose(address common.Address, auth bool) {
	api.smilo.candidatesLock.Lock()
	defer api.smilo.candidatesLock.Unlock()

	// BEGIN SMILO SPECIFICS
	requireSmilos := new(big.Int).Mul(big.NewInt(api.smilo.config.MinFunds), big.NewInt(1e18))

	//check that the address have 10k smilos in it
	statedb, _, err := api.chain.State()
	if err != nil {
		log.Error("Could not propose new candidate, got error with statedb", "error", err, "address", address, "auth", auth)
		return
	} else if statedb.GetBalance(address).Cmp(requireSmilos) < 0 {
		log.Error("Could not propose new candidate", "error", core.ErrInsufficientFunds.Error(), "address", address, "auth", auth, "MinFunds", api.smilo.config.MinFunds, "balance", statedb.GetBalance(address))
		return
	}

	api.smilo.candidates[address] = auth
}

// Discard (clique override) drops a currently running candidate, stopping the fullnode from casting further votes (either for or against).
func (api *API) Discard(address common.Address) {
	api.smilo.candidatesLock.Lock()
	defer api.smilo.candidatesLock.Unlock()

	delete(api.smilo.candidates, address)
}
