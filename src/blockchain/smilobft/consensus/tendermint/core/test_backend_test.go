package core

import (
	"crypto/ecdsa"
	"go-smilo/src/blockchain/smilobft/core/types"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"go-smilo/src/blockchain/smilobft/consensus/tendermint/committee"
	"go-smilo/src/blockchain/smilobft/consensus/tendermint/config"
)

type addressKeyMap map[common.Address]*ecdsa.PrivateKey

func generateCommittee(n int) (types.Committee, addressKeyMap) {
	vals := make(types.Committee, 0)
	keymap := make(addressKeyMap)
	for i := 0; i < n; i++ {
		privateKey, _ := crypto.GenerateKey()
		committeeMember := types.CommitteeMember{
			Address:     crypto.PubkeyToAddress(privateKey.PublicKey),
			VotingPower: new(big.Int).SetUint64(1),
		}
		vals = append(vals, committeeMember)
		keymap[committeeMember.Address] = privateKey
	}
	sort.Sort(vals)
	return vals, keymap
}

func newTestCommitteeSet(n int) committee.Set {

	validators, _ := generateCommittee(n)
	set, _ := committee.NewSet(validators, config.RoundRobin, validators[0].Address)
	return set
}

func newTestCommitteeSetWithKeys(n int) (committee.Set, addressKeyMap) {
	validators, keyMap := generateCommittee(n)
	set, _ := committee.NewSet(validators, config.RoundRobin, validators[0].Address)
	return set, keyMap
}

func generatePrivateKey() (*ecdsa.PrivateKey, error) {
	key := "bb047e5940b6d83354d9432db7c449ac8fca2248008aaa7271369880f9f11cc1"
	return crypto.HexToECDSA(key)
}

func getAddress() common.Address {
	return common.HexToAddress("0x70524d664ffe731100208a0154e556f9bb679ae6")
}
