package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"go-smilo/src/blockchain/smilobft/core/state"
	"go-smilo/src/blockchain/smilobft/core/types"
	"go-smilo/src/blockchain/smilobft/core/vm"
)

// defaultEVMProvider implements autonity.EVMProvider
type defaultEVMProvider struct {
	bc *BlockChain
}

func (p *defaultEVMProvider) EVM(header *types.Header, origin common.Address, statedb *state.StateDB) *vm.EVM {
	coinbase, _ := types.Ecrecover(header)
	evmContext := vm.Context{
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
		GetHash:     GetHashFn(header, p.bc),
		Origin:      origin,
		Coinbase:    coinbase,
		BlockNumber: header.Number,
		Time:        new(big.Int).SetUint64(header.Time),
		GasLimit:    header.GasLimit,
		Difficulty:  header.Difficulty,
		GasPrice:    new(big.Int).SetUint64(0x0),
	}
	vmConfig := *p.bc.GetVMConfig()
	evm := vm.NewEVM(evmContext, statedb,statedb, p.bc.Config(), vmConfig)
	return evm
}
