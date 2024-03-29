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

package eth

import (
	"context"
	"errors"
	"go-smilo/src/blockchain/smilobft/contracts/autonity_tendermint"
	"math/big"

	"go-smilo/src/blockchain/smilobft/cmn"
	"go-smilo/src/blockchain/smilobft/contracts/autonity"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/event"

	"go-smilo/src/blockchain/smilobft/rpc"

	"go-smilo/src/blockchain/smilobft/accounts"
	"go-smilo/src/blockchain/smilobft/core"
	"go-smilo/src/blockchain/smilobft/core/bloombits"
	"go-smilo/src/blockchain/smilobft/core/rawdb"
	"go-smilo/src/blockchain/smilobft/core/state"
	"go-smilo/src/blockchain/smilobft/core/types"
	"go-smilo/src/blockchain/smilobft/core/vm"
	"go-smilo/src/blockchain/smilobft/eth/downloader"
	"go-smilo/src/blockchain/smilobft/eth/gasprice"
	"go-smilo/src/blockchain/smilobft/ethdb"
	"go-smilo/src/blockchain/smilobft/params"
)

// EthAPIBackend implements ethapi.Backend for full nodes
type EthAPIBackend struct {
	extRPCEnabled bool
	eth           *Smilo
	gpo           *gasprice.Oracle

	// Quorum
	//
	// hex node id from node public key
	hexNodeId string
}

// ChainConfig returns the active chain configuration.
func (b *EthAPIBackend) ChainConfig() *params.ChainConfig {
	return b.eth.blockchain.Config()
}

func (b *EthAPIBackend) CurrentBlock() *types.Block {
	return b.eth.blockchain.CurrentBlock()
}

func (b *EthAPIBackend) SetHead(number uint64) {
	b.eth.protocolManager.downloader.Cancel()
	b.eth.blockchain.SetHead(number)
}

func (b *EthAPIBackend) HeaderByNumber(_ context.Context, number rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if number == rpc.PendingBlockNumber {
		block := b.eth.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		return b.eth.blockchain.CurrentBlock().Header(), nil
	}
	return b.eth.blockchain.GetHeaderByNumber(uint64(number)), nil
}

func (b *EthAPIBackend) HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.HeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.eth.blockchain.GetHeaderByHash(hash)
		if header == nil {
			return nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.eth.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		return header, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *EthAPIBackend) HeaderByHash(_ context.Context, hash common.Hash) (*types.Header, error) {
	return b.eth.blockchain.GetHeaderByHash(hash), nil
}

func (b *EthAPIBackend) BlockByNumber(_ context.Context, number rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if number == rpc.PendingBlockNumber {
		block := b.eth.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		return b.eth.blockchain.CurrentBlock(), nil
	}
	return b.eth.blockchain.GetBlockByNumber(uint64(number)), nil
}

func (b *EthAPIBackend) BlockByHash(_ context.Context, hash common.Hash) (*types.Block, error) {
	return b.eth.blockchain.GetBlockByHash(hash), nil
}

func (b *EthAPIBackend) BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.BlockByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.eth.blockchain.GetHeaderByHash(hash)
		if header == nil {
			return nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.eth.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		block := b.eth.blockchain.GetBlock(hash, header.Number.Uint64())
		if block == nil {
			return nil, errors.New("header found, but block body is missing")
		}
		return block, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *EthAPIBackend) StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (vm.SmiloAPIState, *types.Header, error) {
	// Pending state is only known by the miner
	if number == rpc.PendingBlockNumber {
		block, publicState, privateState := b.eth.miner.Pending()
		return EthAPIState{publicState, privateState}, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, nil, err
	}
	if header == nil {
		return nil, nil, errors.New("header not found")
	}
	stateDb, privateState, err := b.eth.BlockChain().StateAt(header.Root)
	return EthAPIState{stateDb, privateState}, header, err

}

func (b *EthAPIBackend) StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (vm.SmiloAPIState, *types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.StateAndHeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header, err := b.HeaderByHash(ctx, hash)
		if err != nil {
			return nil, nil, err
		}
		if header == nil {
			return nil, nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.eth.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, nil, errors.New("hash is not currently canonical")
		}
		stateDb, privateState, err := b.eth.BlockChain().StateAt(header.Root)
		return EthAPIState{stateDb, privateState}, header, err

	}
	return nil, nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *EthAPIBackend) GetReceipts(_ context.Context, hash common.Hash) (types.Receipts, error) {
	return b.eth.blockchain.GetReceiptsByHash(hash), nil
}

func (b *EthAPIBackend) GetLogs(_ context.Context, hash common.Hash) ([][]*types.Log, error) {
	receipts := b.eth.blockchain.GetReceiptsByHash(hash)
	if receipts == nil {
		return nil, nil
	}
	logs := make([][]*types.Log, len(receipts))
	for i, receipt := range receipts {
		logs[i] = receipt.Logs
	}
	return logs, nil
}

func (b *EthAPIBackend) GetTd(blockHash common.Hash) *big.Int {
	return b.eth.blockchain.GetTdByHash(blockHash)
}

func (b *EthAPIBackend) GetEVM(_ context.Context, msg core.Message, state vm.SmiloAPIState, header *types.Header, vmCfg vm.Config) (*vm.EVM, func() error, error) {
	statedb := state.(EthAPIState)
	from := statedb.state.GetOrNewStateObject(msg.From())
	from.SetBalance(math.MaxBig256, header.Number)
	from.SetSmiloPay(math.MaxBig256)
	vmError := func() error { return nil }

	context := core.NewEVMContext(msg, header, b.eth.BlockChain(), nil)

	// Set the private state to public state if contract address is not present in the private state
	to := common.Address{}
	if msg.To() != nil {
		to = *msg.To()
	}

	// Should be public, if address is not present in private state
	privateState := statedb.privateState
	if !privateState.Exist(to) {
		privateState = statedb.state
	}

	return vm.NewEVM(context, statedb.state, statedb.privateState, b.eth.chainConfig, vmCfg), vmError, nil
}

func (b *EthAPIBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.eth.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *EthAPIBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.eth.BlockChain().SubscribeChainEvent(ch)
}

func (b *EthAPIBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.eth.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *EthAPIBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.eth.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *EthAPIBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.eth.BlockChain().SubscribeLogsEvent(ch)
}

func (b *EthAPIBackend) SendTx(_ context.Context, signedTx *types.Transaction) error {
	// validation for node need to happen here and cannot be done as a part of
	// validateTx in tx_pool.go as tx_pool validation will happen in every node
	if b.hexNodeId != "" && !types.ValidateNodeForTxn(b.hexNodeId, signedTx.From()) {
		return errors.New("cannot send transaction from this node")
	}
	return b.eth.txPool.AddLocal(signedTx)
}

func (b *EthAPIBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := b.eth.txPool.Pending()
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *EthAPIBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.eth.txPool.Get(hash)
}

func (b *EthAPIBackend) GetTransaction(_ context.Context, txHash common.Hash) (*types.Transaction, common.Hash, uint64, uint64, error) {
	tx, blockHash, blockNumber, index := rawdb.ReadTransaction(b.eth.ChainDb(), txHash)
	return tx, blockHash, blockNumber, index, nil
}

func (b *EthAPIBackend) GetPoolNonce(_ context.Context, addr common.Address) (uint64, error) {
	return b.eth.txPool.Nonce(addr), nil
}

func (b *EthAPIBackend) Stats() (pending int, queued int) {
	return b.eth.txPool.Stats()
}

func (b *EthAPIBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.eth.TxPool().Content()
}

func (b *EthAPIBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.eth.TxPool().SubscribeNewTxsEvent(ch)
}

func (b *EthAPIBackend) Downloader() *downloader.Downloader {
	return b.eth.Downloader()
}

func (b *EthAPIBackend) ProtocolVersion() int {
	return b.eth.EthVersion()
}

func (b *EthAPIBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	//if smilo and gas is false
	if b.ChainConfig().IsSmilo && !b.ChainConfig().IsGas {
		return big.NewInt(0), nil
	} else {
		return b.gpo.SuggestPrice(ctx)
	}
}

func (b *EthAPIBackend) ChainDb() ethdb.Database {
	return b.eth.ChainDb()
}

func (b *EthAPIBackend) EventMux() *cmn.TypeMux {
	return b.eth.EventMux()
}

func (b *EthAPIBackend) AccountManager() *accounts.Manager {
	return b.eth.AccountManager()
}

func (b *EthAPIBackend) AutonityContract() *autonity.Contract {
	return b.eth.blockchain.GetAutonityContract()
}

func (b *EthAPIBackend) AutonityContractTendermint() *autonity_tendermint.Contract {
	return b.eth.blockchain.GetAutonityContractTendermint()
}

func (b *EthAPIBackend) ExtRPCEnabled() bool {
	return b.extRPCEnabled
}

func (b *EthAPIBackend) RPCGasCap() *big.Int {
	return b.eth.config.RPCGasCap
}

func (b *EthAPIBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.eth.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *EthAPIBackend) ServiceFilter(_ context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.eth.bloomRequests)
	}
}

func (b *EthAPIBackend) GetSolcPath() string {
	solcpath := b.eth.config.SolcPath
	return solcpath
}

func (b *EthAPIBackend) GetSmiloCodeAnalysisPath() string {
	codeAnalysisPath := b.eth.config.SmiloCodeAnalysisPath
	return codeAnalysisPath
}

// used by Quorum
type EthAPIState struct {
	state, privateState *state.StateDB
}

func (s EthAPIState) GetBalance(addr common.Address) *big.Int {
	if s.privateState.Exist(addr) {
		return s.privateState.GetBalance(addr)
	}
	return s.state.GetBalance(addr)
}

func (s EthAPIState) GetCode(addr common.Address) []byte {
	if s.privateState.Exist(addr) {
		return s.privateState.GetCode(addr)
	}
	return s.state.GetCode(addr)
}

func (s EthAPIState) SetNonce(addr common.Address, nonce uint64) {
	if s.privateState.Exist(addr) {
		s.privateState.SetNonce(addr, nonce)
	} else {
		s.state.SetNonce(addr, nonce)
	}
}

func (s EthAPIState) SetCode(addr common.Address, code []byte) {
	if s.privateState.Exist(addr) {
		s.privateState.SetCode(addr, code)
	} else {
		s.state.SetCode(addr, code)
	}
}

func (s EthAPIState) SetBalance(addr common.Address, balance, blockNumber *big.Int) {
	if s.privateState.Exist(addr) {
		s.privateState.SetBalance(addr, balance, blockNumber)
	} else {
		s.state.SetBalance(addr, balance, blockNumber)
	}
}

func (s EthAPIState) SetStorage(addr common.Address, storage map[common.Hash]common.Hash) {
	if s.privateState.Exist(addr) {
		s.privateState.SetStorage(addr, storage)
	} else {
		s.state.SetStorage(addr, storage)
	}
}

func (s EthAPIState) SetState(addr common.Address, key, value common.Hash) {
	if s.privateState.Exist(addr) {
		s.privateState.SetState(addr, key, value)
	} else {
		s.state.SetState(addr, key, value)
	}
}

func (s EthAPIState) GetState(a common.Address, b common.Hash) common.Hash {
	if s.privateState.Exist(a) {
		return s.privateState.GetState(a, b)
	}
	return s.state.GetState(a, b)
}

func (s EthAPIState) GetNonce(addr common.Address) uint64 {
	if s.privateState.Exist(addr) {
		return s.privateState.GetNonce(addr)
	}
	return s.state.GetNonce(addr)
}

// AddBalance implemented to satisfy SmiloAPIState
func (s EthAPIState) AddBalance(addr common.Address, amount *big.Int, blockNumber *big.Int) {
	if s.privateState.Exist(addr) {
		s.privateState.AddBalance(addr, amount, blockNumber)
	} else {
		s.state.AddBalance(addr, amount, blockNumber)
	}
}

// SubSmiloPay implemented to satisfy SmiloAPIState
func (s EthAPIState) SubSmiloPay(addr common.Address, amount *big.Int, blockNumber *big.Int) {
	if s.privateState.Exist(addr) {
		s.privateState.SubSmiloPay(addr, amount, blockNumber)
	} else {
		s.state.SubSmiloPay(addr, amount, blockNumber)
	}
}

// AddSmiloPay implemented to satisfy SmiloAPIState
func (s EthAPIState) AddSmiloPay(addr common.Address, amount *big.Int) {
	if s.privateState.Exist(addr) {
		s.privateState.AddSmiloPay(addr, amount)
	} else {
		s.state.AddSmiloPay(addr, amount)
	}
}

// GetSmiloPay implemented to satisfy SmiloAPIState
func (s EthAPIState) GetSmiloPay(addr common.Address, blockNumber *big.Int) *big.Int {
	if s.privateState.Exist(addr) {
		return s.privateState.GetSmiloPay(addr, blockNumber)
	}
	return s.state.GetSmiloPay(addr, blockNumber)
}

// SubBalance implemented to satisfy SmiloAPIState
func (s EthAPIState) SubBalance(addr common.Address, amount, blockNumber *big.Int) {
	if s.privateState.Exist(addr) {
		s.privateState.SubBalance(addr, amount, blockNumber)
	} else {
		s.state.SubBalance(addr, amount, blockNumber)
	}
}

func (s EthAPIState) GetProof(addr common.Address) ([][]byte, error) {
	if s.privateState.Exist(addr) {
		return s.privateState.GetProof(addr)
	}
	return s.state.GetProof(addr)
}

func (s EthAPIState) GetStorageProof(addr common.Address, h common.Hash) ([][]byte, error) {
	if s.privateState.Exist(addr) {
		return s.privateState.GetStorageProof(addr, h)
	}
	return s.state.GetStorageProof(addr, h)
}

func (s EthAPIState) StorageTrie(addr common.Address) state.Trie {
	if s.privateState.Exist(addr) {
		return s.privateState.StorageTrie(addr)
	}
	return s.state.StorageTrie(addr)
}

func (s EthAPIState) Error() error {
	if s.privateState.Error() != nil {
		return s.privateState.Error()
	}
	return s.state.Error()
}

func (s EthAPIState) GetCodeHash(addr common.Address) common.Hash {
	if s.privateState.Exist(addr) {
		return s.privateState.GetCodeHash(addr)
	}
	return s.state.GetCodeHash(addr)
}

func (b *EthAPIBackend) IsSelfInWhitelist() error {
	return b.eth.protocolManager.IsSelfInWhitelist()
}
