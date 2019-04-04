// Copyright 2019 The go-smilo Authors
// Copyright 2014 The go-ethereum Authors
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

package vm

import (
	"math/big"
	"sync/atomic"

	"go-smilo/src/blockchain/smilobft/cmn"

	"go-smilo/src/blockchain/smilobft/core/state"
	"go-smilo/src/blockchain/smilobft/params"

	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

// note: Smilo, States, and Value Transfer
//
// In Smilo there is a tricky issue in one specific case when there is call from vault state to public state:
// * The state db is selected based on the callee (public)
// * With every call there is an associated value transfer -- in our case this is 0
// * Thus, there is an implicit transfer of 0 value from the caller to callee on the public state
// * However in our scenario the caller is vault
// * Thus, the transfer creates a ghost of the vault account on the public state with no value, code, or storage
//
// The solution is to skip this transfer of 0 value under Smilo

// emptyCodeHash is used by create to ensure deployment is disallowed to already
// deployed contract addresses (relevant after the account abstraction).
var emptyCodeHash = crypto.Keccak256Hash(nil)

type (
	// CanTransferFunc is the signature of a transfer guard function
	CanTransferFunc func(StateDB, common.Address, *big.Int) bool
	// TransferFunc is the signature of a transfer function
	TransferFunc func(StateDB, common.Address, common.Address, *big.Int, *big.Int)
	// GetHashFunc returns the nth block hash in the blockchain
	// and is used by the BLOCKHASH EVM op code.
	GetHashFunc func(uint64) common.Hash
)

// run runs the given contract and takes care of running precompiles with a fallback to the byte code interpreter.
func run(evm *EVM, contract *Contract, input []byte, readOnly, isVault bool) ([]byte, error) {
	if contract.CodeAddr != nil {
		precompiles := PrecompiledContractsHomestead
		if evm.ChainConfig().IsByzantium(evm.BlockNumber) {
			precompiles = PrecompiledContractsByzantium
		}
		if p := precompiles[*contract.CodeAddr]; p != nil {
			return RunPrecompiledContract(p, input, contract)
		}
	}
	for _, interpreter := range evm.interpreters {
		if interpreter.CanRun(contract.Code) {
			if evm.interpreter != interpreter {
				// Ensure that the interpreter pointer is set back
				// to its current value upon return.
				defer func(i Interpreter) {
					evm.interpreter = i
				}(evm.interpreter)
				evm.interpreter = interpreter
			}
			return interpreter.Run(contract, input, readOnly, isVault)
		}
	}
	return nil, ErrNoCompatibleInterpreter
}

// Context provides the EVM with auxiliary information. Once provided
// it shouldn't be modified.
type Context struct {
	// CanTransfer returns whether the account contains
	// sufficient ether to transfer the value
	CanTransfer CanTransferFunc
	// Transfer transfers ether from one account to the other
	Transfer TransferFunc
	// GetHash returns the hash corresponding to n
	GetHash GetHashFunc

	// Message information
	Origin   common.Address // Provides information for ORIGIN
	GasPrice *big.Int       // Provides information for GASPRICE

	// Block information
	Coinbase    common.Address // Provides information for COINBASE
	GasLimit    uint64         // Provides information for GASLIMIT
	BlockNumber *big.Int       // Provides information for NUMBER
	Time        *big.Int       // Provides information for TIME
	Difficulty  *big.Int       // Provides information for DIFFICULTY
}

type PublicState StateDB
type VaultState StateDB

// EVM is the Ethereum Virtual Machine base object and provides
// the necessary tools to run a contract on the given state with
// the provided context. It should be noted that any error
// generated through any of the calls should be considered a
// revert-state-and-consume-all-gas operation, no checks on
// specific errors should ever be performed. The interpreter makes
// sure that any errors generated are to be considered faulty code.
//
// The EVM should never be reused and is not thread safe.
type EVM struct {
	// Context provides auxiliary blockchain related information
	Context
	// StateDB gives access to the underlying state
	StateDB StateDB
	// Depth is the current call stack
	depth int

	// chainConfig contains information about the current chain
	chainConfig *params.ChainConfig
	// chain rules contains the chain rules for the current epoch
	chainRules params.Rules
	// virtual machine configuration options used to initialise the
	// evm.
	vmConfig Config
	// global (to this context) ethereum virtual machine
	// used throughout the execution of the tx.
	interpreters []Interpreter
	interpreter  Interpreter
	// abort is used to abort the EVM calling operations
	// NOTE: must be set atomically
	abort int32
	// callGasTemp holds the gas available for the current call. This is needed because the
	// available gas is calculated in gasCall* according to the 63/64 rule and later
	// applied in opCall*.
	callGasTemp uint64

	// Smilo additions:
	publicState       PublicState
	vaultState        VaultState
	states            [1027]*state.StateDB
	currentStateDepth uint
	// Smilo read only state. Inside Vault State towards Public State read.
	smiloReadOnly bool
	readOnlyDepth uint
}

// NewEVM returns a new EVM. The returned EVM is not thread safe and should
// only ever be used *once*.
func NewEVM(ctx Context, statedb, vaultState StateDB, chainConfig *params.ChainConfig, vmConfig Config) *EVM {
	evm := &EVM{
		Context:      ctx,
		StateDB:      statedb,
		vmConfig:     vmConfig,
		chainConfig:  chainConfig,
		chainRules:   chainConfig.Rules(ctx.BlockNumber),
		interpreters: make([]Interpreter, 0, 1),

		publicState: statedb,
		vaultState:  vaultState,
	}

	if chainConfig.IsEWASM(ctx.BlockNumber) {
		// to be implemented by EVM-C and Wagon PRs.
		// if vmConfig.EWASMInterpreter != "" {
		//  extIntOpts := strings.Split(vmConfig.EWASMInterpreter, ":")
		//  path := extIntOpts[0]
		//  options := []string{}
		//  if len(extIntOpts) > 1 {
		//    options = extIntOpts[1..]
		//  }
		//  evm.interpreters = append(evm.interpreters, NewEVMVCInterpreter(evm, vmConfig, options))
		// } else {
		// 	evm.interpreters = append(evm.interpreters, NewEWASMInterpreter(evm, vmConfig))
		// }
		panic("No supported ewasm interpreter yet.")
	}

	evm.Push(vaultState)

	// vmConfig.EVMInterpreter will be used by EVM-C, it won't be checked here
	// as we always want to have the built-in EVM as the failover option.
	evm.interpreters = append(evm.interpreters, NewEVMInterpreter(evm, vmConfig))
	evm.interpreter = evm.interpreters[0]

	return evm
}

// Cancel cancels any running EVM operation. This may be called concurrently and
// it's safe to be called multiple times.
func (evm *EVM) Cancel() {
	atomic.StoreInt32(&evm.abort, 1)
}

// Interpreter returns the current interpreter
func (evm *EVM) Interpreter() Interpreter {
	return evm.interpreter
}

// Call executes the contract associated with the addr with the given input as
// parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
func (evm *EVM) Call(caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int, isVault bool) (ret []byte, leftOverGas uint64, err error) {
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, gas, nil
	}

	isVaultOnDB, thisstate := getPrivateOrPublicStateDB(evm, addr)
	evm.Push(thisstate)
	defer func() { evm.Pop() }()
	if isVault != isVaultOnDB {
		log.Debug("&*&*&*&*&*& evm.Call, ErrIsVaultDiffThenIsVaultOnDB, ", "from", caller.Address().Hex(), "to", addr.Hex(), "gas", gas, "value", value, "input", cmn.Bytes2Hex(input), "evm.smiloReadOnly", evm.smiloReadOnly, "isVault", isVault, "isVaultOnDB", isVaultOnDB)
		isVault = isVaultOnDB
	}

	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}

	// Fail if we're trying to transfer more than the available balance
	// transfer is only allowed on publicState, update it here to allow estimateGas to work
	if !evm.Context.CanTransfer(evm.publicState, caller.Address(), value) {
		log.Debug("EVM.Call, CanTransfer, ErrInsufficientBalance, ", "from", caller.Address().Hex(), "to", addr.Hex(), "gas", gas, "value", value, "input", cmn.Bytes2Hex(input), "evm.smiloReadOnly", evm.smiloReadOnly, "isVault", isVault)
		return nil, gas, ErrInsufficientBalance
	}

	var (
		to       = AccountRef(addr)
		snapshot = evm.StateDB.Snapshot()
	)
	if !evm.StateDB.Exist(addr) {
		precompiles := PrecompiledContractsHomestead
		if evm.ChainConfig().IsByzantium(evm.BlockNumber) {
			precompiles = PrecompiledContractsByzantium
		}
		if precompiles[addr] == nil && evm.ChainConfig().IsEIP158(evm.BlockNumber) && value.Sign() == 0 {
			// Calling a non existing account, don't do anything, but ping the tracer
			if evm.vmConfig.Debug && evm.depth == 0 {
				evm.vmConfig.Tracer.CaptureStart(caller.Address(), addr, false, input, gas, value)
				evm.vmConfig.Tracer.CaptureEnd(ret, 0, 0, nil)
			}
			return nil, gas, nil
		}
		evm.StateDB.CreateAccount(addr)
	}

	if evm.ChainConfig().IsSmilo {
		if value.Sign() != 0 {
			if evm.smiloReadOnly && isVault {
				log.Debug("***************** EVM.Call, Transfer, ", "error", ErrReadOnlyValueTransfer, "isVault", isVault, "IsSmilo", evm.ChainConfig().IsSmilo, "value", value.Sign(), "smiloReadOnly", evm.smiloReadOnly)
				return nil, gas, ErrReadOnlyValueTransfer
			}
			// zero and (not read only or not vault), can transfer
			// transfer is only allowed on publicState, update it here to allow estimateGas to work
			evm.Transfer(evm.publicState, caller.Address(), to.Address(), value, evm.BlockNumber)
			log.Trace("EVM.Call, Transfer, ", "isVault", isVault, "IsSmilo", evm.ChainConfig().IsSmilo, "value", value.Sign(), "smiloReadOnly", evm.smiloReadOnly)
		} else {
			log.Trace("EVM.Call, Transfer, IGNORE, ", "isVault", isVault, "IsSmilo", evm.ChainConfig().IsSmilo, "value", value.Sign(), "smiloReadOnly", evm.smiloReadOnly)
		}
	} else {
		// transfer is only allowed on publicState, update it here to allow estimateGas to work
		evm.Transfer(evm.publicState, caller.Address(), to.Address(), value, evm.BlockNumber)
		log.Trace("EVM.Call, Transfer, ", "isVault", isVault, "IsSmilo", evm.ChainConfig().IsSmilo, "value", value.Sign(), "smiloReadOnly", evm.smiloReadOnly)
	}

	// Initialise a new contract and set the code that is to be used by the EVM.
	// The contract is a scoped environment for this execution context only.
	contract := NewContract(caller, to, value, gas)
	contract.SetCallCode(&addr, evm.StateDB.GetCodeHash(addr), evm.StateDB.GetCode(addr))

	// Even if the account has no code, we need to continue because it might be a precompile
	start := time.Now()

	// Capture the tracer start/end events in debug mode
	if evm.vmConfig.Debug && evm.depth == 0 {
		evm.vmConfig.Tracer.CaptureStart(caller.Address(), addr, false, input, gas, value)

		defer func() { // Lazy evaluation of the parameters
			evm.vmConfig.Tracer.CaptureEnd(ret, gas-contract.Gas, time.Since(start), err)
		}()
	}
	ret, err = run(evm, contract, input, false, isVault)

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// CallCode executes the contract associated with the addr with the given input
// as parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
//
// CallCode differs from Call in the sense that it executes the given address'
// code with the caller as context.
func (evm *EVM) CallCode(caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int, isVault bool) (ret []byte, leftOverGas uint64, err error) {
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, gas, nil
	}

	isVaultOnDB, thisstate := getPrivateOrPublicStateDB(evm, addr)
	evm.Push(thisstate)
	defer func() { evm.Pop() }()
	if isVault != isVaultOnDB {
		log.Debug("&*&*&*&*&*& evm.CallCode, ErrIsVaultDiffThenIsVaultOnDB, ", "from", caller.Address().Hex(), "to", addr.Hex(), "gas", gas, "value", value, "input", cmn.Bytes2Hex(input), "evm.smiloReadOnly", evm.smiloReadOnly, "isVault", isVault, "isVaultOnDB", isVaultOnDB)
		isVault = isVaultOnDB
	}

	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}
	// Fail if we're trying to transfer more than the available balance
	if !evm.CanTransfer(evm.StateDB, caller.Address(), value) {
		return nil, gas, ErrInsufficientBalance
	}

	var (
		snapshot = evm.StateDB.Snapshot()
		to       = AccountRef(caller.Address())
	)
	// initialise a new contract and set the code that is to be used by the
	// EVM. The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(caller, to, value, gas)
	contract.SetCallCode(&addr, evm.StateDB.GetCodeHash(addr), evm.StateDB.GetCode(addr))

	ret, err = run(evm, contract, input, false, isVault)
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// DelegateCall executes the contract associated with the addr with the given input
// as parameters. It reverses the state in case of an execution error.
//
// DelegateCall differs from CallCode in the sense that it executes the given address'
// code with the caller as context and the caller is set to the caller of the caller.
func (evm *EVM) DelegateCall(caller ContractRef, addr common.Address, input []byte, gas uint64, isVault bool) (ret []byte, leftOverGas uint64, err error) {
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, gas, nil
	}

	isVaultOnDB, thisstate := getPrivateOrPublicStateDB(evm, addr)
	evm.Push(thisstate)
	defer func() { evm.Pop() }()
	if isVault != isVaultOnDB {
		log.Debug("&*&*&*&*&*& evm.DelegateCall, ErrIsVaultDiffThenIsVaultOnDB, ", "from", caller.Address().Hex(), "to", addr.Hex(), "gas", gas, "input", cmn.Bytes2Hex(input), "evm.smiloReadOnly", evm.smiloReadOnly, "isVault", isVault, "isVaultOnDB", isVaultOnDB)
		isVault = isVaultOnDB
	}

	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}

	var (
		snapshot = evm.StateDB.Snapshot()
		to       = AccountRef(caller.Address())
	)

	// Initialise a new contract and make initialise the delegate values
	contract := NewContract(caller, to, nil, gas).AsDelegate()
	contract.SetCallCode(&addr, evm.StateDB.GetCodeHash(addr), evm.StateDB.GetCode(addr))

	ret, err = run(evm, contract, input, false, isVault)
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// StaticCall executes the contract associated with the addr with the given input
// as parameters while disallowing any modifications to the state during the call.
// Opcodes that attempt to perform such modifications will result in exceptions
// instead of performing the modifications.
func (evm *EVM) StaticCall(caller ContractRef, addr common.Address, input []byte, gas uint64, isVault bool) (ret []byte, leftOverGas uint64, err error) {
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, gas, nil
	}
	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}

	isVaultOnDB, stateDB := getPrivateOrPublicStateDB(evm, addr)
	if isVault != isVaultOnDB {
		log.Debug("&*&*&*&*&*& evm.StaticCall, ErrIsVaultDiffThenIsVaultOnDB, ", "from", caller.Address().Hex(), "to", addr.Hex(), "gas", gas, "input", cmn.Bytes2Hex(input), "evm.smiloReadOnly", evm.smiloReadOnly, "isVault", isVault, "isVaultOnDB", isVaultOnDB)
		isVault = isVaultOnDB
	}

	var (
		to       = AccountRef(addr)
		snapshot = evm.StateDB.Snapshot()
	)
	// Initialise a new contract and set the code that is to be used by the
	// EVM. The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(caller, to, new(big.Int), gas)
	contract.SetCallCode(&addr, stateDB.GetCodeHash(addr), stateDB.GetCode(addr))

	// We do an AddBalance of zero here, just in order to trigger a touch.
	// This doesn't matter on Mainnet, where all empties are gone at the time of Byzantium,
	// but is the correct thing to do and matters on other networks, in tests, and potential
	// future scenarios
	stateDB.AddBalance(addr, bigZero, evm.BlockNumber)

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in Homestead this also counts for code storage gas errors.
	ret, err = run(evm, contract, input, true, isVault)
	if err != nil {
		stateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

type codeAndHash struct {
	code []byte
	hash common.Hash
}

func (c *codeAndHash) Hash() common.Hash {
	if c.hash == (common.Hash{}) {
		c.hash = crypto.Keccak256Hash(c.code)
	}
	return c.hash
}

// create creates a new contract using code as deployment code.
func (evm *EVM) create(caller ContractRef, codeAndHash *codeAndHash, gas uint64, value *big.Int, address common.Address, isVault bool) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error) {

	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if evm.depth > int(params.CallCreateDepth) {
		return nil, common.Address{}, gas, ErrDepth
	}
	if !evm.CanTransfer(evm.StateDB, caller.Address(), value) {
		return nil, common.Address{}, gas, ErrInsufficientBalance
	}

	// Get the right state in case of a dual state environment. If a sender
	// is a transaction (depth == 0) use the public state to derive the address
	// and increment the nonce of the public state. If the sender is a contract
	// (depth > 0) use the vault state to derive the nonce and increment the
	// nonce on the vault state only.
	//
	// If the transaction went to a public contract the vault and public state
	// are the same.
	var isVaultOnDB bool
	var creatorStateDb StateDB
	if evm.Depth() > 0 {
		creatorStateDb = evm.vaultState
		isVaultOnDB = true
	} else {
		creatorStateDb = evm.publicState
	}

	if isVault != isVaultOnDB {
		log.Debug("&*&*&*&*&*& evm.create, ErrIsVaultDiffThenIsVaultOnDB, ", "from", caller.Address().Hex(), "to", address.Hex(), "gas", gas, "value", value, "evm.smiloReadOnly", evm.smiloReadOnly, "isVault", isVault, "isVaultOnDB", isVaultOnDB)
		isVault = isVaultOnDB
	}

	nonce := creatorStateDb.GetNonce(caller.Address())
	creatorStateDb.SetNonce(caller.Address(), nonce+1)

	// Ensure there's no existing contract already at the designated address
	contractHash := evm.StateDB.GetCodeHash(address)
	if evm.StateDB.GetNonce(address) != 0 || (contractHash != (common.Hash{}) && contractHash != emptyCodeHash) {
		return nil, common.Address{}, 0, ErrContractAddressCollision
	}
	// Create a new account on the state
	snapshot := evm.StateDB.Snapshot()
	evm.StateDB.CreateAccount(address)
	if evm.ChainConfig().IsEIP158(evm.BlockNumber) {
		evm.StateDB.SetNonce(address, 1)
	}

	if evm.ChainConfig().IsSmilo {
		if value.Sign() != 0 {
			if evm.smiloReadOnly && isVault {
				log.Debug("***************** EVM.Create, Transfer, ", "error", ErrReadOnlyValueTransfer, "isVault", isVault, "IsSmilo", evm.ChainConfig().IsSmilo, "value", value.Sign(), "smiloReadOnly", evm.smiloReadOnly)
				return nil, common.Address{}, gas, ErrReadOnlyValueTransfer
			}
			//zero and (not read only or not vault) , can transfer
			evm.Transfer(evm.StateDB, caller.Address(), address, value, evm.BlockNumber)
			log.Trace("EVM.Create, Transfer, ", "isVault", isVault, "IsSmilo", evm.ChainConfig().IsSmilo, "value", value.Sign(), "smiloReadOnly", evm.smiloReadOnly)
		} else {
			log.Trace("EVM.Create, Transfer, IGNORE, ", "isVault", isVault, "IsSmilo", evm.ChainConfig().IsSmilo, "value", value.Sign(), "smiloReadOnly", evm.smiloReadOnly)
		}
	} else {
		evm.Transfer(evm.StateDB, caller.Address(), contractAddr, value, evm.BlockNumber)
		log.Trace("EVM.Create, Transfer, ", "isVault", isVault, "IsSmilo", evm.ChainConfig().IsSmilo, "value", value.Sign(), "smiloReadOnly", evm.smiloReadOnly)
	}

	// initialise a new contract and set the code that is to be used by the
	// EVM. The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(caller, AccountRef(address), value, gas)
	contract.SetCodeOptionalHash(&address, codeAndHash)

	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, address, gas, nil
	}

	if evm.vmConfig.Debug && evm.depth == 0 {
		evm.vmConfig.Tracer.CaptureStart(caller.Address(), address, true, codeAndHash.code, gas, value)
	}
	start := time.Now()

	ret, err = run(evm, contract, nil, false, isVault)

	// check whether the max code size has been exceeded
	maxCodeSizeExceeded := evm.ChainConfig().IsEIP158(evm.BlockNumber) && len(ret) > params.MaxCodeSize
	// if the contract creation ran successfully and no errors were returned
	// calculate the gas required to store the code. If the code could not
	// be stored due to not enough gas set an error and let it be handled
	// by the error checking condition below.
	if err == nil && !maxCodeSizeExceeded {
		createDataGas := uint64(len(ret)) * params.CreateDataGas
		if contract.UseGas(createDataGas) {
			evm.StateDB.SetCode(address, ret)
		} else {
			err = ErrCodeStoreOutOfGas
		}
	}

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if maxCodeSizeExceeded || (err != nil && (evm.ChainConfig().IsHomestead(evm.BlockNumber) || err != ErrCodeStoreOutOfGas)) {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	// Assign err if contract code size exceeds the max while the err is still empty.
	if maxCodeSizeExceeded && err == nil {
		err = errMaxCodeSizeExceeded
	}
	if evm.vmConfig.Debug && evm.depth == 0 {
		evm.vmConfig.Tracer.CaptureEnd(ret, gas-contract.Gas, time.Since(start), err)
	}
	return ret, address, contract.Gas, err

}

// Create creates a new contract using code as deployment code.
func (evm *EVM) Create(caller ContractRef, code []byte, gas uint64, value *big.Int, isVault bool) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error) {
	var creatorStateDb StateDB
	if evm.depth > 0 {
		creatorStateDb = evm.vaultState
	} else {
		creatorStateDb = evm.publicState
	}
	contractAddr = crypto.CreateAddress(caller.Address(), creatorStateDb.GetNonce(caller.Address()))
	return evm.create(caller, &codeAndHash{code: code}, gas, value, contractAddr, isVault)
}

// Create2 creates a new contract using code as deployment code.
//
// The different between Create2 with Create is Create2 uses sha3(0xff ++ msg.sender ++ salt ++ sha3(init_code))[12:]
// instead of the usual sender-and-nonce-hash as the address where the contract is initialized at.
func (evm *EVM) Create2(caller ContractRef, code []byte, gas uint64, endowment *big.Int, salt *big.Int, isVault bool) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error) {
	codeAndHash := &codeAndHash{code: code}
	contractAddr = crypto.CreateAddress2(caller.Address(), common.BigToHash(salt), codeAndHash.Hash().Bytes())
	return evm.create(caller, codeAndHash, gas, endowment, contractAddr, isVault)
}

// ChainConfig returns the environment's chain configuration
func (evm *EVM) ChainConfig() *params.ChainConfig { return evm.chainConfig }

func getPrivateOrPublicStateDB(env *EVM, addr common.Address) (isVault bool, thisState StateDB) {
	// priv: (a) -> (b)  (vault)
	// pub:   a  -> [b]  (vault -> public)
	// priv: (a) ->  b   (public)
	thisState = env.StateDB

	if env.VaultState().Exist(addr) {
		isVault = true
		thisState = env.VaultState()
	} else if env.PublicState().Exist(addr) {
		thisState = env.PublicState()
	}

	return isVault, thisState
}

func (env *EVM) PublicState() PublicState { return env.publicState }
func (env *EVM) VaultState() VaultState   { return env.vaultState }
func (env *EVM) Push(statedb StateDB) {
	if env.vaultState != statedb && !env.smiloReadOnly {
		env.smiloReadOnly = true
		env.readOnlyDepth = env.currentStateDepth
	}

	if castedStateDb, ok := statedb.(*state.StateDB); ok {
		env.states[env.currentStateDepth] = castedStateDb
		env.currentStateDepth++
	}

	env.StateDB = statedb
}
func (env *EVM) Pop() {
	env.currentStateDepth--
	if env.currentStateDepth == env.readOnlyDepth && env.smiloReadOnly {
		env.smiloReadOnly = false
	}
	env.StateDB = env.states[env.currentStateDepth-1]
}

func (env *EVM) Depth() int { return env.depth }

// We only need to revert the current state because when we call from vault
// public state it's read only, there wouldn't be anything to reset.
// (A)->(B)->C->(B): A failure in (B) wouldn't need to reset C, as C was flagged
// read only.
func (self *EVM) RevertToSnapshot(snapshot int) {
	self.StateDB.RevertToSnapshot(snapshot)
}
