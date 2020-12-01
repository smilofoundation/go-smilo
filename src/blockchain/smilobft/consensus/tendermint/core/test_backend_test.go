package core

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"go-smilo/src/blockchain/smilobft/consensus/tendermint/config"
	"go-smilo/src/blockchain/smilobft/consensus/tendermint/committee"
)

type addressKeyMap map[common.Address]*ecdsa.PrivateKey

func generateValidators(n int) ([]common.Address, addressKeyMap) {
	vals := make([]common.Address, 0)
	keymap := make(addressKeyMap)
	for i := 0; i < n; i++ {
		privateKey, _ := crypto.GenerateKey()
		addr := crypto.PubkeyToAddress(privateKey.PublicKey)
		vals = append(vals, addr)
		keymap[addr] = privateKey
	}
	return vals, keymap
}

func newTestValidatorSet(n int) committee.Set {
	validators, _ := generateValidators(n)
	return committee.NewSet(validators, config.RoundRobin)
}

func newTestValidatorSetWithKeys(n int) (committee.Set, addressKeyMap) {
	validators, keyMap := generateValidators(n)
	return committee.NewSet(validators, config.RoundRobin), keyMap
}

func generatePrivateKey() (*ecdsa.PrivateKey, error) {
	key := "bb047e5940b6d83354d9432db7c449ac8fca2248008aaa7271369880f9f11cc1"
	return crypto.HexToECDSA(key)
}

func getAddress() common.Address {
	return common.HexToAddress("0x70524d664ffe731100208a0154e556f9bb679ae6")
}
