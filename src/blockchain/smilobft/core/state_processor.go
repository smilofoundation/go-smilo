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

package core

import (
	"errors"
	"github.com/ethereum/go-ethereum/log"
	"go-smilo/src/blockchain/smilobft/contracts/autonity"
	"go-smilo/src/blockchain/smilobft/core/types"
	"math/big"

	"go-smilo/src/blockchain/smilobft/core/state"
	"go-smilo/src/blockchain/smilobft/core/vm"
	"go-smilo/src/blockchain/smilobft/params"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"go-smilo/src/blockchain/smilobft/consensus"
	"go-smilo/src/blockchain/smilobft/consensus/misc"
)

// StateProcessor is a basic Processor, which takes care of transitioning
// state from one point to another.
//
// StateProcessor implements Processor.
type StateProcessor struct {
	config           *params.ChainConfig // Chain configuration options
	bc               *BlockChain         // Canonical block chain
	engine           consensus.Engine    // Consensus engine used for block rewards
	autonityContract *autonity.Contract
}

// NewStateProcessor initialises a new StateProcessor.
func NewStateProcessor(config *params.ChainConfig, bc *BlockChain, engine consensus.Engine) *StateProcessor {
	return &StateProcessor{
		config: config,
		bc:     bc,
		engine: engine,
	}
}

func (p *StateProcessor) SetAutonityContract(contract *autonity.Contract) {
	p.autonityContract = contract
}

// Process processes the state changes according to the Ethereum rules by running
// the transaction messages using the statedb and applying any rewards to both
// the processor (coinbase) and any included uncles.
//
// Process returns the receipts and logs accumulated during the process and
// returns the amount of gas that was used in the process. If any of the
// transactions failed to execute due to insufficient gas it will return an error.
func (p *StateProcessor) Process(block *types.Block, statedb, vaultState *state.StateDB, cfg vm.Config) (types.Receipts, types.Receipts, []*types.Log, uint64, error) {
	var (
		receipts types.Receipts
		usedGas  = new(uint64)
		header   = block.Header()
		allLogs  []*types.Log
		gp       = new(GasPool).AddGas(block.GasLimit())

		vaultReceipts types.Receipts
	)
	// Mutate the the block and state according to any hard-fork specs
	if p.config.DAOForkSupport && p.config.DAOForkBlock != nil && p.config.DAOForkBlock.Cmp(block.Number()) == 0 {
		misc.ApplyDAOHardFork(statedb, block.Number())
	}

	var contractMinGasPrice = new(big.Int)
	if (p.bc.Config().Istanbul != nil || p.bc.Config().SportDAO != nil || p.bc.Config().Tendermint != nil) && p.autonityContract != nil {
		minGasPrice, err := p.autonityContract.GetMinimumGasPrice(block, statedb, vaultState)
		if err == nil {
			contractMinGasPrice.SetUint64(minGasPrice)
		}
	} else {
		msg := "Wont set Istanbul Tendermint SportDAO GetMinimumGasPrice, is this correct ? "
		log.Warn(msg)
		//panic(msg)
	}
	// Iterate over and process the individual transactions
	for i, tx := range block.Transactions() {
		statedb.Prepare(tx.Hash(), block.Hash(), i)

		if (p.bc.Config().Istanbul != nil || p.bc.Config().SportDAO != nil || p.bc.Config().Tendermint != nil) && p.autonityContract != nil {
			if contractMinGasPrice.Uint64() != 0 {
				if tx.GasPrice().Cmp(contractMinGasPrice) == -1 {
					return nil, nil, nil, 0, errors.New("autonityContract, gas price must be greater minGasPrice")
				}
			}
		} else {
			msg := "Wont set Istanbul Tendermint SportDAO Process, is this correct ? "
			log.Warn(msg)
			//panic(msg)
		}

		vaultState.Prepare(tx.Hash(), block.Hash(), i)

		receipt, vaultReceipt, _, err := ApplyTransaction(p.config, p.bc, nil, gp, statedb, vaultState, header, tx, usedGas, cfg)
		if err != nil {
			return nil, nil, nil, 0, err
		}
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)

		// if the vault receipt is nil this means the tx was public
		// and we do not need to apply the additional logic.
		if vaultReceipt != nil {
			vaultReceipts = append(vaultReceipts, vaultReceipt)
			allLogs = append(allLogs, vaultReceipt.Logs...)
		}
	}
	if (p.bc.chainConfig.Istanbul != nil || p.bc.chainConfig.SportDAO != nil || p.bc.chainConfig.Tendermint != nil) && p.autonityContract != nil {
		err := p.autonityContract.ApplyPerformRedistribution(block.Transactions(), receipts, block.Header(), statedb)
		if err != nil {
			log.Error("Could not ApplyPerformRedistribution on smart contract, ", "err", err)
			return nil, nil, nil, 0, err
		}
	} else {
		msg := "Wont set Istanbul Tendermint SportDAO ApplyPerformRedistribution, is this correct ? "
		log.Warn(msg)
		//panic(msg)
	}
	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	p.engine.Finalize(p.bc, header, statedb, block.Transactions(), block.Uncles(), receipts)

	return receipts, vaultReceipts, allLogs, *usedGas, nil
}

// ApplyTransaction attempts to apply a transaction to the given state database
// and uses the input parameters for its environment. It returns the receipt
// for the transaction, gas used and an error if the transaction failed,
// indicating the block was invalid.
func ApplyTransaction(config *params.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb, vaultState *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, cfg vm.Config) (*types.Receipt, *types.Receipt, uint64, error) {
	//if Smilo is enabled and transaction is Vault, set the VaultStateDB = StateDB
	if !config.IsSmilo || !tx.IsVault() {
		vaultState = statedb
	}

	if !config.IsGas && tx.GasPrice() != nil && tx.GasPrice().Cmp(common.Big0) > 0 {
		return nil, nil, 0, ErrInvalidGasPrice
	}

	msg, err := tx.AsMessage(types.MakeSigner(config, header.Number))
	if err != nil {
		return nil, nil, 0, err
	}
	// Create a new context to be used in the EVM environment
	context := NewEVMContext(msg, header, bc, author)
	// Create a new environment which holds all relevant information
	// about the transaction and calling mechanisms.
	vmenv := vm.NewEVM(context, statedb, vaultState, config, cfg)

	// Apply the transaction to the current state (included in the env)
	_, gas, failed, err := ApplyMessage(vmenv, msg, gp)
	if err != nil {
		return nil, nil, 0, err
	}
	// Update the state with pending changes
	var root []byte
	if config.IsByzantium(header.Number) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(config.IsEIP158(header.Number)).Bytes()
	}
	*usedGas += gas

	// Vault transactions when Smilo is enable will ignore failures
	publicFailed := !(config.IsSmilo && tx.IsVault()) && failed

	// Create a new receipt for the transaction, storing the intermediate root and gas used by the tx
	// based on the eip phase, we're passing whether the root touch-delete accounts.
	receipt := types.NewReceipt(root, publicFailed, *usedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = gas
	// if the transaction created a contract, store the creation address in the receipt.
	if msg.To() == nil {
		receipt.ContractAddress = crypto.CreateAddress(vmenv.Context.Origin, tx.Nonce())
	}
	// Set the receipt logs and create a bloom for filtering
	receipt.Logs = statedb.GetLogs(tx.Hash())
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	receipt.BlockHash = statedb.BlockHash()
	receipt.BlockNumber = header.Number
	receipt.TransactionIndex = uint(statedb.TxIndex())

	var vaultReceipt *types.Receipt

	// If Smilo is enabled and transaction is Vault, generate Vault Receipt data
	if config.IsSmilo && tx.IsVault() {
		var vaultRoot []byte
		if config.IsByzantium(header.Number) {
			vaultState.Finalise(true)
		} else {
			vaultRoot = vaultState.IntermediateRoot(config.IsEIP158(header.Number)).Bytes()
		}
		vaultReceipt = types.NewReceipt(vaultRoot, failed, *usedGas)
		vaultReceipt.TxHash = tx.Hash()
		vaultReceipt.GasUsed = gas
		if msg.To() == nil {
			vaultReceipt.ContractAddress = crypto.CreateAddress(vmenv.Context.Origin, tx.Nonce())
		}

		vaultReceipt.Logs = vaultState.GetLogs(tx.Hash())
		vaultReceipt.Bloom = types.CreateBloom(types.Receipts{vaultReceipt})
	}

	return receipt, vaultReceipt, gas, err
}
