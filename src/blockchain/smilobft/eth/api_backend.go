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
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"

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

func (b *EthAPIBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
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

func (b *EthAPIBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return b.eth.blockchain.GetHeaderByHash(hash), nil
}

func (b *EthAPIBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
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

func (b *EthAPIBackend) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.eth.blockchain.GetBlockByHash(hash), nil
}

func (b *EthAPIBackend) StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (vm.SmiloAPIState, *types.Header, error) {
	// Pending state is only known by the miner
	if number == rpc.PendingBlockNumber {
		block, publicState, vaultState := b.eth.miner.Pending()
		return EthAPIState{publicState, vaultState}, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, nil, err
	}
	if header == nil {
		return nil, nil, errors.New("header not found")
	}
	stateDb, vaultState, err := b.eth.BlockChain().StateAt(header.Root)
	return EthAPIState{stateDb, vaultState}, header, err
}

func (b *EthAPIBackend) GetBlock(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.eth.blockchain.GetBlockByHash(hash), nil
}

func (b *EthAPIBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return b.eth.blockchain.GetReceiptsByHash(hash), nil
}

func (b *EthAPIBackend) GetLogs(ctx context.Context, hash common.Hash) ([][]*types.Log, error) {
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

func (b *EthAPIBackend) GetEVM(ctx context.Context, msg core.Message, state vm.SmiloAPIState, header *types.Header, vmCfg vm.Config) (*vm.EVM, func() error, error) {
	statedb := state
	from := statedb.(EthAPIState).State.GetOrNewStateObject(msg.From())
	from.SetBalance(math.MaxBig256, header.Number)
	from.SetSmiloPay(math.MaxBig256)
	vmError := func() error { return nil }

	context := core.NewEVMContext(msg, header, b.eth.BlockChain(), nil)

	to := common.Address{}
	if msg.To() != nil {
		to = *msg.To()
	}

	// Should be public, if address is not present in private state
	privateState := statedb.(EthAPIState).VaultState
	if !privateState.Exist(to) {
		privateState = statedb.(EthAPIState).State
	}

	return vm.NewEVM(context, statedb.(EthAPIState).State, statedb.(EthAPIState).VaultState, b.eth.chainConfig, vmCfg), vmError, nil
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

func (b *EthAPIBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
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

func (b *EthAPIBackend) GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, common.Hash, uint64, uint64, error) {
	tx, blockHash, blockNumber, index := rawdb.ReadTransaction(b.eth.ChainDb(), txHash)
	return tx, blockHash, blockNumber, index, nil
}

func (b *EthAPIBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
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

func (b *EthAPIBackend) EventMux() *event.TypeMux {
	return b.eth.EventMux()
}

func (b *EthAPIBackend) AccountManager() *accounts.Manager {
	return b.eth.AccountManager()
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

func (b *EthAPIBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
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

type EthAPIState struct {
	State, VaultState *state.StateDB
}

func (ethApiState EthAPIState) SetCode(addr common.Address, code []byte) {
	if ethApiState.VaultState.Exist(addr) {
		stateObject := ethApiState.VaultState.GetOrNewStateObject(addr)
		if stateObject != nil {
			stateObject.SetCode(crypto.Keccak256Hash(code), code)
		}
	} else {
		stateObject := ethApiState.State.GetOrNewStateObject(addr)
		if stateObject != nil {
			stateObject.SetCode(crypto.Keccak256Hash(code), code)
		}
	}
}

func (ethApiState EthAPIState) SetState(addr common.Address, key, value common.Hash) {
	if ethApiState.VaultState.Exist(addr) {
		stateObject := ethApiState.VaultState.GetOrNewStateObject(addr)
		if stateObject != nil {
			stateObject.SetState(ethApiState.VaultState.Database(), key, value)
		}
	} else {
		stateObject := ethApiState.State.GetOrNewStateObject(addr)
		if stateObject != nil {
			stateObject.SetState(ethApiState.State.Database(), key, value)
		}
	}
}

func (ethApiState EthAPIState) SetBalance(addr common.Address, amount, blockNumber *big.Int) {
	if ethApiState.VaultState.Exist(addr) {
		stateObject := ethApiState.VaultState.GetOrNewStateObject(addr)
		if stateObject != nil {
			stateObject.SetBalance(amount, blockNumber)
		}
	} else {
		stateObject := ethApiState.State.GetOrNewStateObject(addr)
		if stateObject != nil {
			stateObject.SetBalance(amount, blockNumber)
		}
	}
}

func (ethApiState EthAPIState) SetStorage(addr common.Address, storage map[common.Hash]common.Hash) {
	if ethApiState.VaultState.Exist(addr) {
		stateObject := ethApiState.VaultState.GetOrNewStateObject(addr)
		if stateObject != nil {
			stateObject.SetStorage(storage)
		}
	} else {
		stateObject := ethApiState.State.GetOrNewStateObject(addr)
		if stateObject != nil {
			stateObject.SetStorage(storage)
		}
	}
}

func (ethApiState EthAPIState) SetNonce(addr common.Address, nonce uint64) {
	if ethApiState.VaultState.Exist(addr) {
		stateObject := ethApiState.VaultState.GetOrNewStateObject(addr)
		if stateObject != nil {
			stateObject.SetNonce(nonce)
		}
	} else {
		stateObject := ethApiState.State.GetOrNewStateObject(addr)
		if stateObject != nil {
			stateObject.SetNonce(nonce)
		}
	}
}

func (ethApiState EthAPIState) GetBalance(addr common.Address) *big.Int {
	if ethApiState.VaultState.Exist(addr) {
		return ethApiState.VaultState.GetBalance(addr)
	}
	return ethApiState.State.GetBalance(addr)
}

// AddBalance implemented to satisfy SmiloAPIState
func (ethApiState EthAPIState) AddBalance(addr common.Address, amount *big.Int, blockNumber *big.Int) {
	if ethApiState.VaultState.Exist(addr) {
		ethApiState.VaultState.AddBalance(addr, amount, blockNumber)
	} else {
		ethApiState.State.AddBalance(addr, amount, blockNumber)
	}
}

// SubSmiloPay implemented to satisfy SmiloAPIState
func (ethApiState EthAPIState) SubSmiloPay(addr common.Address, amount *big.Int, blockNumber *big.Int) {
	if ethApiState.VaultState.Exist(addr) {
		ethApiState.VaultState.SubSmiloPay(addr, amount, blockNumber)
	} else {
		ethApiState.State.SubSmiloPay(addr, amount, blockNumber)
	}
}

// AddSmiloPay implemented to satisfy SmiloAPIState
func (ethApiState EthAPIState) AddSmiloPay(addr common.Address, amount *big.Int) {
	if ethApiState.VaultState.Exist(addr) {
		ethApiState.VaultState.AddSmiloPay(addr, amount)
	} else {
		ethApiState.State.AddSmiloPay(addr, amount)
	}
}

// GetSmiloPay implemented to satisfy SmiloAPIState
func (ethApiState EthAPIState) GetSmiloPay(addr common.Address, blockNumber *big.Int) *big.Int {
	if ethApiState.VaultState.Exist(addr) {
		return ethApiState.VaultState.GetSmiloPay(addr, blockNumber)
	}
	return ethApiState.State.GetSmiloPay(addr, blockNumber)
}

// SubBalance implemented to satisfy SmiloAPIState
func (ethApiState EthAPIState) SubBalance(addr common.Address, amount, blockNumber *big.Int) {
	if ethApiState.VaultState.Exist(addr) {
		ethApiState.VaultState.SubBalance(addr, amount, blockNumber)
	} else {
		ethApiState.State.SubBalance(addr, amount, blockNumber)
	}
}

func (ethApiState EthAPIState) GetCode(addr common.Address) []byte {
	if ethApiState.VaultState.Exist(addr) {
		return ethApiState.VaultState.GetCode(addr)
	}
	return ethApiState.State.GetCode(addr)
}

func (ethApiState EthAPIState) GetState(a common.Address, b common.Hash) common.Hash {
	if ethApiState.VaultState.Exist(a) {
		return ethApiState.VaultState.GetState(a, b)
	}
	return ethApiState.State.GetState(a, b)
}

func (ethApiState EthAPIState) GetNonce(addr common.Address) uint64 {
	if ethApiState.VaultState.Exist(addr) {
		return ethApiState.VaultState.GetNonce(addr)
	}
	return ethApiState.State.GetNonce(addr)
}

func (ethApiState EthAPIState) GetProof(addr common.Address) ([][]byte, error) {
	if ethApiState.VaultState.Exist(addr) {
		return ethApiState.VaultState.GetProof(addr)
	}
	return ethApiState.State.GetProof(addr)
}

func (ethApiState EthAPIState) GetStorageProof(addr common.Address, h common.Hash) ([][]byte, error) {
	if ethApiState.VaultState.Exist(addr) {
		return ethApiState.VaultState.GetStorageProof(addr, h)
	}
	return ethApiState.State.GetStorageProof(addr, h)
}

func (ethApiState EthAPIState) StorageTrie(addr common.Address) state.Trie {
	if ethApiState.VaultState.Exist(addr) {
		return ethApiState.VaultState.StorageTrie(addr)
	}
	return ethApiState.State.StorageTrie(addr)
}

func (ethApiState EthAPIState) Error() error {
	if ethApiState.VaultState.Error() != nil {
		return ethApiState.VaultState.Error()
	}
	return ethApiState.State.Error()
}

func (ethApiState EthAPIState) GetCodeHash(addr common.Address) common.Hash {
	if ethApiState.VaultState.Exist(addr) {
		return ethApiState.VaultState.GetCodeHash(addr)
	}
	return ethApiState.State.GetCodeHash(addr)
}
