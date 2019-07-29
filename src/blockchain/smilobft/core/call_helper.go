package core

import (
	"crypto/ecdsa"
	"math/big"

	"go-smilo/src/blockchain/smilobft/core/rawdb"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"go-smilo/src/blockchain/smilobft/consensus/ethash"
	"go-smilo/src/blockchain/smilobft/core/state"
	"go-smilo/src/blockchain/smilobft/core/types"
	"go-smilo/src/blockchain/smilobft/core/vm"
	"go-smilo/src/blockchain/smilobft/ethdb"
	"go-smilo/src/blockchain/smilobft/params"
)

// callHelper makes it easier to do proper calls and use the state transition object.
// It also manages the nonces of the caller and keeps vault and public state, which
// can be freely modified outside of the helper.
type callHelper struct {
	db ethdb.Database

	nonces map[common.Address]uint64
	header types.Header
	gp     *GasPool

	VaultState, PublicState *state.StateDB
}

// TxNonce returns the pending nonce
func (cg *callHelper) TxNonce(addr common.Address) uint64 {
	return cg.nonces[addr]
}

// MakeCall makes does a call to the recipient using the given input. It can switch between vault and public
// by setting the vault boolean flag. It returns an error if the call failed.
func (cg *callHelper) MakeCall(vault bool, key *ecdsa.PrivateKey, to common.Address, input []byte) error {
	var (
		from = crypto.PubkeyToAddress(key.PublicKey)
		err  error
	)

	cg.header.Number = new(big.Int)
	cg.header.Time = 43
	cg.header.Difficulty = new(big.Int).SetUint64(1000488)
	cg.header.GasLimit = 4700000

	signer := types.MakeSigner(params.SmiloTestChainConfig, cg.header.Number)
	tx, err := types.SignTx(types.NewTransaction(cg.TxNonce(from), to, new(big.Int), 1000000, new(big.Int), input), signer, key)
	if err != nil {
		return err
	}
	defer func() { cg.nonces[from]++ }()
	msg, err := tx.AsMessage(signer)
	if err != nil {
		return err
	}

	publicState, vaultState := cg.PublicState, cg.VaultState
	if !vault {
		vaultState = publicState
	} else {
		tx.SetVault()
	}

	bc, _ := NewBlockChain(cg.db, nil, params.SmiloTestChainConfig, ethash.NewFaker(), vm.Config{}, nil)
	context := NewEVMContext(msg, &cg.header, bc, &from)
	vmenv := vm.NewEVM(context, publicState, vaultState, params.SmiloTestChainConfig, vm.Config{})
	_, _, _, err = ApplyMessage(vmenv, msg, cg.gp)
	return err
}

// MakeCallHelper returns a new callHelper
func MakeCallHelper() *callHelper {
	memdb := rawdb.NewMemoryDatabase()
	db := state.NewDatabase(memdb)

	publicState, err := state.New(common.Hash{}, db)
	if err != nil {
		panic(err)
	}
	vaultState, err := state.New(common.Hash{}, db)
	if err != nil {
		panic(err)
	}
	cg := &callHelper{
		db:          memdb,
		nonces:      make(map[common.Address]uint64),
		gp:          new(GasPool).AddGas(5000000),
		PublicState: publicState,
		VaultState:  vaultState,
	}
	return cg
}
