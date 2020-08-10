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

package core

import (
	"errors"
	"math/big"

	"go-smilo/src/blockchain/smilobft/cmn"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/log"

	"go-smilo/src/blockchain/smilobft/core/vm"

	"go-smilo/src/blockchain/smilobft/params"
	"go-smilo/src/blockchain/smilobft/private"
)

var (
	errInsufficientBalanceForGas = errors.New("insufficient balance to pay for gas")
)

/*
The State Transitioning Model

A state transition is a change made when a transaction is applied to the current world state
The state transitioning model does all the necessary work to work out a valid new state root.

1) Nonce handling
2) Pre pay gas
3) Create a new state object if the recipient is \0*32
4) Value transfer
== If contract creation ==
  4a) Attempt to run transaction data
  4b) If valid, use result as code for the new state object
== end ==
5) Run Script section
6) Derive new state root
*/
type StateTransition struct {
	gp         *GasPool
	msg        Message
	gas        uint64
	gasPrice   *big.Int
	initialGas uint64
	value      *big.Int
	data       []byte
	state      vm.StateDB
	evm        *vm.EVM
}

// Message represents a message sent to a contract.
type Message interface {
	From() common.Address
	//FromFrontier() (common.Address, error)
	To() *common.Address

	GasPrice() *big.Int
	Gas() uint64
	Value() *big.Int

	Nonce() uint64
	CheckNonce() bool
	Data() []byte
}

// PrivateMessage implements a private message
type PrivateMessage interface {
	Message
	IsPrivate() bool
}

// IntrinsicGas computes the 'intrinsic gas' for a message with the given data.
func IntrinsicGas(data []byte, contractCreation, isEIP155 bool, isEIP2028 bool) (uint64, error) {
	// Set the starting gas for the raw transaction
	var gas uint64
	if contractCreation && isEIP155 {
		gas = params.TxGasContractCreation
	} else {
		gas = params.TxGas
	}
	// Bump the required gas by the amount of transactional data
	if len(data) > 0 {
		// Zero and non-zero bytes are priced differently
		var nz uint64
		for _, byt := range data {
			if byt != 0 {
				nz++
			}
		}
		// Make sure we don't exceed uint64 for all data combinations
		nonZeroGas := params.TxDataNonZeroGasFrontier
		if isEIP2028 {
			nonZeroGas = params.TxDataNonZeroGasEIP2028
		}
		if (math.MaxUint64-gas)/nonZeroGas < nz {
			return 0, vm.ErrOutOfGas
		}
		gas += nz * nonZeroGas

		z := uint64(len(data)) - nz
		if (math.MaxUint64-gas)/params.TxDataZeroGas < z {
			return 0, vm.ErrOutOfGas
		}
		gas += z * params.TxDataZeroGas
	}
	return gas, nil
}

// NewStateTransition initialises and returns a new state transition object.
func NewStateTransition(evm *vm.EVM, msg Message, gp *GasPool) *StateTransition {
	return &StateTransition{
		gp:       gp,
		evm:      evm,
		msg:      msg,
		gasPrice: msg.GasPrice(),
		value:    msg.Value(),
		data:     msg.Data(),
		state:    evm.PublicState(),
	}
}

// ApplyMessage computes the new state by applying the given message
// against the old state within the environment.
//
// ApplyMessage returns the bytes returned by any EVM execution (if it took place),
// the gas used (which includes gas refunds) and an error if it failed. An error always
// indicates a core error meaning that the message would always fail for that particular
// state and would never be accepted within a block.

func ApplyMessage(evm *vm.EVM, msg Message, gp *GasPool) ([]byte, uint64, bool, error) {
	return NewStateTransition(evm, msg, gp).TransitionDb()
}

// to returns the recipient of the message.
func (st *StateTransition) to() common.Address {
	if st.msg == nil || st.msg.To() == nil /* contract creation */ {
		return common.Address{}
	}
	return *st.msg.To()
}

func (st *StateTransition) useGas(amount uint64) error {
	if st.gas < amount {
		return vm.ErrOutOfGas
	}
	st.gas -= amount

	return nil
}

//TODO: smilopay
func (st *StateTransition) buyGas() error {
	mgval := new(big.Int).Mul(new(big.Int).SetUint64(st.msg.Gas()), st.gasPrice)
	//check if balance > cost for smilopay
	if st.state.GetSmiloPay(st.msg.From(), st.evm.BlockNumber).Cmp(mgval) < 0 {
		return ErrInsufficientSmiloPay
	}
	//check if balance > cost for gas
	if st.state.GetBalance(st.msg.From()).Cmp(mgval) < 0 {
		return errInsufficientBalanceForGas
	}
	//subtract gas
	if err := st.gp.SubGas(st.msg.Gas()); err != nil {
		return err
	}
	st.gas += st.msg.Gas()
	st.initialGas = st.msg.Gas()

	//subtract smilopay
	st.state.SubSmiloPay(st.msg.From(), mgval, st.evm.BlockNumber)
	//subtract balance to pay for gas (deposit)
	st.state.SubBalance(st.msg.From(), mgval, st.evm.BlockNumber)
	return nil
}

func (st *StateTransition) preCheck() error {
	// Make sure this transaction's nonce is correct.
	if st.msg.CheckNonce() {
		nonce := st.state.GetNonce(st.msg.From())
		if nonce < st.msg.Nonce() {
			return ErrNonceTooHigh
		} else if nonce > st.msg.Nonce() {
			return ErrNonceTooLow
		}
	}
	return st.buyGas()
}

// TransitionDb will transition the state by applying the current message and
// returning the result including the used gas. It returns an error if failed.
// An error indicates a consensus issue.
//
// Quorum:
// 1. Intrinsic gas is calculated based on the encrypted payload hash
//    and NOT the actual private payload
// 2. For private transactions, we only deduct intrinsic gas from the gas pool
//    regardless the current node is party to the transaction or not
func (st *StateTransition) TransitionDb() (ret []byte, usedGas uint64, failed bool, err error) {
	if err = st.preCheck(); err != nil {
		return
	}
	msg := st.msg
	sender := vm.AccountRef(msg.From())
	homestead := st.evm.ChainConfig().IsHomestead(st.evm.BlockNumber)
	istanbul := st.evm.ChainConfig().IsIstanbul(st.evm.BlockNumber)

	contractCreation := msg.To() == nil
	isSmilo := st.evm.ChainConfig().IsSmilo
	isGas := st.evm.ChainConfig().IsGas
	isGasRefunded := st.evm.ChainConfig().IsGasRefunded

	var data []byte
	isPrivate := false
	publicState := st.state
	if msg, ok := msg.(PrivateMessage); ok && isSmilo && msg.IsPrivate() {
		if private.VaultInstance == nil {
			log.Error("&*&*&*&*& state_transition TransitionDb, Got Vault message but Vault is offline. Please report to SystemAdmin. ", "st.data", cmn.Bytes2Hex(st.data), "contractCreation", contractCreation, "isPrivate", isPrivate, "len(ret)", len(ret), "st.gasUsed", st.gasUsed(), "st.gasPrice", st.gasPrice, "sender.Address", sender.Address())
			publicState.SetNonce(sender.Address(), publicState.GetNonce(sender.Address())+1)
			return nil, 0, false, nil
		} else {
			isPrivate = true
			data, err = private.VaultInstance.Get(st.data)
			// Increment the public account nonce if:
			// 1. Tx is private and *not* a participant of the group and either call or create
			// 2. Tx is private we are part of the group and is a call
			if err != nil || !contractCreation {
				publicState.SetNonce(sender.Address(), publicState.GetNonce(sender.Address())+1)
			}

			if err != nil {
				return nil, 0, false, nil
			}
		}
	} else {
		data = st.data
	}

	// Pay intrinsic gas. For a private contract this is done using the public hash passed in,
	// not the private data retrieved above. This is because we need any (participant) validator
	// node to get the same result as a (non-participant) minter node, to avoid out-of-gas issues.
	gas, err := IntrinsicGas(st.data, contractCreation, homestead, istanbul)
	if err != nil {
		return nil, 0, false, err
	}
	if err = st.useGas(gas); err != nil {
		return nil, 0, false, err
	}

	var (
		leftoverGas uint64
		evm         = st.evm
		// vm errors do not effect consensus and are therefor
		// not assigned to err, except for insufficient balance
		// error.
		vmerr error
	)
	if contractCreation {
		ret, _, leftoverGas, vmerr = evm.Create(sender, data, st.gas, st.value, isPrivate)
	} else {
		// Increment the account nonce only if the transaction isn't private.
		// If the transaction is private it has already been incremented on
		// the public state.
		if !isPrivate {
			publicState.SetNonce(msg.From(), publicState.GetNonce(sender.Address())+1)
		}
		var to common.Address
		if isSmilo {
			to = *st.msg.To()
		} else {
			to = st.to()
		}
		//if input is empty for the smart contract call, return
		if len(data) == 0 && isPrivate {
			if isSmilo && isGas {
				st.refundGasSmiloVersion()
			} else {
				st.refundGas()
				st.state.AddBalance(st.evm.Coinbase, new(big.Int).Mul(new(big.Int).SetUint64(st.gasUsed()), st.gasPrice), st.evm.BlockNumber)
			}
			return nil, 0, false, nil
		}

		log.Debug("############### state_transition, will execute evm.Call, ", "isPrivate", isPrivate)
		ret, leftoverGas, vmerr = evm.Call(sender, to, data, st.gas, st.value, isPrivate)
	}
	if vmerr != nil {
		log.Info("VM returned with error", "err", vmerr)
		// The only possible consensus-error would be if there wasn't
		// sufficient balance to make the transfer happen. The first
		// balance transfer may never fail.
		if vmerr == vm.ErrInsufficientBalance {
			return nil, 0, false, vmerr
		}
	} else {
		log.Debug("############### state_transition, VM returned with NO error after executing evm, ", "contractCreation", contractCreation, "isPrivate", isPrivate, "leftoverGas", leftoverGas, "len(ret)", len(ret), "st.gasUsed()", st.gasUsed(), "st.gasPrice", st.gasPrice)
	}

	if st.evm.ChainConfig().AutonityContractConfig != nil && (st.evm.ChainConfig().Istanbul != nil || st.evm.ChainConfig().SportDAO != nil || st.evm.ChainConfig().Tendermint != nil) {

		st.refundGas()
		addr, innerErr := st.evm.ChainConfig().AutonityContractConfig.GetContractAddress()
		if innerErr != nil {
			return nil, 0, true, innerErr
		}
		st.state.AddBalance(addr, new(big.Int).Mul(new(big.Int).SetUint64(st.gasUsed()), st.gasPrice), st.evm.BlockNumber)

		return ret, st.gasUsed(), vmerr != nil, err
	} else {

		// Pay gas used during contract creation or execution (st.gas tracks remaining gas)
		// However, if private contract then we don't want to do this else we can get
		// a mismatch between a (non-participant) minter and (participant) validator,
		// which can cause a 'BAD BLOCK' crash.
		if !isPrivate {
			st.gas = leftoverGas
		}

		//give back gas if no err on EVM && IsSmilo=true,IsGas=true,IsGasRefund=true
		if vmerr == nil && isSmilo && isGas && isGasRefunded {
			log.Debug("############### state_transition, give back gas if no err on EVM after executing evm.Call, ", "contractCreation", contractCreation, "isPrivate", isPrivate, "leftoverGas", leftoverGas, "len(ret)", len(ret), "st.gasUsed()", st.gasUsed(), "st.gasPrice", st.gasPrice)
			st.refundGasSmiloVersion()
			// miners do not get reward in gas for transactions on smilo
			//st.state.AddBalance(st.evm.Coinbase, new(big.Int).Mul(new(big.Int).SetUint64(st.gasUsed()), st.gasPrice), st.evm.BlockNumber)
			//running normal node should do the default
		} else {
			log.Debug("############### state_transition, refund gas left if err on EVM after executing evm.Call, ", "contractCreation", contractCreation, "isPrivate", isPrivate, "leftoverGas", leftoverGas, "len(ret)", len(ret), "st.gasUsed()", st.gasUsed(), "st.gasPrice", st.gasPrice)
			st.refundGas()
			st.state.AddBalance(st.evm.Coinbase, new(big.Int).Mul(new(big.Int).SetUint64(st.gasUsed()), st.gasPrice), st.evm.BlockNumber)
		}

		if isPrivate {
			return ret, 0, vmerr != nil, err
		}
		return ret, st.gasUsed(), vmerr != nil, err
	}
}

func (st *StateTransition) refundGas() {
	// Apply refund counter, capped to half of the used gas.
	refund := st.gasUsed() / 2
	if refund > st.state.GetRefund() {
		refund = st.state.GetRefund()
	}
	st.gas += refund

	// Return ETH for remaining gas, exchanged at the original rate.
	remaining := new(big.Int).Mul(new(big.Int).SetUint64(st.gas), st.gasPrice)

	//refund gas, only what was not used for the transaction, if any
	st.state.AddBalance(st.msg.From(), remaining, st.evm.BlockNumber)

	// Also return remaining gas to the block gas counter so it is
	// available for the next transaction.
	st.gp.AddGas(st.gas)
}

//TODO: smilopay
func (st *StateTransition) refundGasSmiloVersion() {
	//refund := st.gasUsed()
	// if GetRefund > gasUsed, then refund what is on GetRefund (GetRefund is used for contract suicide params.SuicideRefundGas)
	// TODO: I'm not sure, if you do not pay smilo to deploy but get smilo back on suicide calls, you could earn smilo by doing this many times ?
	//if st.state.GetRefund() > refund {
	//	refund = st.state.GetRefund()
	//}

	// Apply refund, return whole deposit
	// Return ETH for deposited gas, exchanged at the original rate.
	deposit := new(big.Int).Mul(new(big.Int).SetUint64(st.gasUsed()), st.gasPrice)
	//refund deposit gas
	st.state.AddBalance(st.msg.From(), deposit, st.evm.BlockNumber)

	//calculate gas that was not used and return to GasPool
	refund := st.gasUsed() / 2
	if refund > st.state.GetRefund() {
		refund = st.state.GetRefund()
	}
	st.gas += refund
	// Also return remaining gas to the block gas counter so it is
	// available for the next transaction.
	st.gp.AddGas(st.gas)
}

func (st *StateTransition) refundSmiloPay() {
	// Apply refund counter, capped to half of the used gas.
	refund := st.gasUsed() / 2
	if refund > st.state.GetRefund() {
		refund = st.state.GetRefund()
	}
	st.gas += refund

	// Return ETH for remaining gas, exchanged at the original rate.
	remaining := new(big.Int).Mul(new(big.Int).SetUint64(st.gas), st.gasPrice)

	//refund smiloPay
	st.state.AddSmiloPay(st.msg.From(), remaining)

	// Also return remaining gas to the block gas counter so it is
	// available for the next transaction.
	st.gp.AddGas(st.gas)
}

// gasUsed returns the amount of gas used up by the state transition.
func (st *StateTransition) gasUsed() uint64 {
	return st.initialGas - st.gas
}
