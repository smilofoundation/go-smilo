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

package types

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"fmt"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var k0v, _ = new(big.Int).SetString("25807260602402504536675820444142779248993100028628438487502323668296269534891", 10)

var k1v, _ = new(big.Int).SetString("10148397294747000913768625849546502595195728826990639993137198410557736548965", 10)

func createKey(c elliptic.Curve, k *big.Int) (*ecdsa.PrivateKey, error) {
	sk := new(ecdsa.PrivateKey)
	sk.PublicKey.Curve = c
	sk.D = k
	sk.PublicKey.X, sk.PublicKey.Y = c.ScalarBaseMult(k.Bytes())
	return sk, nil
}

func signTx(key *ecdsa.PrivateKey, signer Signer) (*Transaction, common.Address, error) {
	addr := crypto.PubkeyToAddress(key.PublicKey)
	tx := NewTransaction(0, addr, new(big.Int), 0, new(big.Int), nil)
	signedTx, err := SignTx(tx, signer, key)
	return signedTx, addr, err
}

func TestSignSmiloHomesteadPublic(t *testing.T) {


	k0, _ := createKey(crypto.S256(), k0v)
	k1, _ := createKey(crypto.S256(), k1v)

	homeSinger := HomesteadSigner{}

	signedTx, addr, _ := signTx(k1, homeSinger)

	require.True(t, signedTx.data.V.Cmp(big.NewInt(27)) == 0, fmt.Sprintf("v wasn't 27 it was [%v]", signedTx.data.V))

	from, _ := Sender(homeSinger, signedTx)
	require.True(t, from == addr, fmt.Sprintf("Expected from and address to be equal. Got %x want %x", from, addr))

	signedTx, addr, _ = signTx(k0, homeSinger)
	require.True(t, signedTx.data.V.Cmp(big.NewInt(28)) == 0, fmt.Sprintf("v wasn't 28 it was [%v]\n", signedTx.data.V))

	from, _ = Sender(homeSinger, signedTx)
	require.True(t, from == addr, fmt.Sprintf("Expected from and address to be equal. Got %x want %x", from, addr))

}

func TestSignSmiloEIP155Public(t *testing.T) {


	k0, _ := createKey(crypto.S256(), k0v)
	k1, _ := createKey(crypto.S256(), k1v)

	var chainId int64
	chainId = 2

	v0 := chainId*2 + 35
	v1 := chainId*2 + 36

	EIPsigner := NewEIP155Signer(big.NewInt(chainId))

	signedTx, addr, _ := signTx(k0, EIPsigner)

	require.True(t, signedTx.data.V.Cmp(big.NewInt(v0)) == 0, fmt.Sprintf("v wasn't [%v] it was [%v]\n", v0, signedTx.data.V))
	from, _ := Sender(EIPsigner, signedTx)

	require.True(t, from == addr, fmt.Sprintf("Expected from and address to be equal. Got %x want %x", from, addr))

	require.False(t, signedTx.IsVault(), fmt.Sprintf("Public transaction is set to a private transation v == [%v]", signedTx.data.V))

	signedTx, addr, _ = signTx(k1, EIPsigner)

	require.True(t, signedTx.data.V.Cmp(big.NewInt(v1)) == 0, fmt.Sprintf("v wasn't [%v], it was [%v]\n", v1, signedTx.data.V))
	from, _ = Sender(EIPsigner, signedTx)

	require.True(t, from == addr, fmt.Sprintf("Expected from and address to be equal. Got %x want %x", from, addr))

}

func TestSignSmiloEIP155FailPublicChain1(t *testing.T) {


	k0, _ := createKey(crypto.S256(), k0v)
	k1, _ := createKey(crypto.S256(), k1v)

	var chainId int64
	chainId = 1

	v0 := chainId*2 + 35
	v1 := chainId*2 + 36

	EIPsigner := NewEIP155Signer(big.NewInt(chainId))

	signedTx, addr, _ := signTx(k0, EIPsigner)

	require.True(t, signedTx.data.V.Cmp(big.NewInt(v0)) == 0, fmt.Sprintf("v wasn't [%v] it was "+
		"[%v]\n", v0, signedTx.data.V))

	require.True(t, signedTx.IsVault(), "A public transaction with EIP155 and chainID 1 is expected to be "+
		"considered vault, as its v param conflict with a vault transaction. signedTx.IsVault() == [%v]", signedTx.IsVault())
	from, _ := Sender(EIPsigner, signedTx)

	require.False(t, from == addr, fmt.Sprintf("Expected the sender of a public TX from chainId 1, \n "+
		"should not be recoverable from [%x] addr [%v] ", from, addr))

	signedTx, addr, _ = signTx(k1, EIPsigner)

	require.True(t, signedTx.data.V.Cmp(big.NewInt(v1)) == 0,
		fmt.Sprintf("v wasn't [%v] it was [%v]", v1, signedTx.data.V))

	require.True(t, signedTx.IsVault(), "A public transaction with EIP155 and chainID 1 is expected to "+
		"to be considered vault, as its v param conflict with a vault transaction. "+
		"signedTx.IsVault() == [%v]", signedTx.IsVault())
	from, _ = Sender(EIPsigner, signedTx)

	require.False(t, from == addr, fmt.Sprintf("Expected the sender of a public TX from chainId 1, "+
		"should not be recoverable from [%x] addr [%v] ", from, addr))

}


func TestSignSmiloHomesteadEIP155SigningVaultSmilo(t *testing.T) {


	keys := []*big.Int{k0v, k1v}

	homeSinger := HomesteadSigner{}
	recoverySigner := NewEIP155Signer(big.NewInt(18))

	for i := 0; i < len(keys); i++ {
		key, _ := createKey(crypto.S256(), keys[i])
		signedTx, addr, err := signTx(key, homeSinger)

		require.Nil(t, err, err)

		signedTx.SetVault()

		require.True(t, signedTx.IsVault(), fmt.Sprintf("Expected the transaction to be private [%v]", signedTx.IsVault()))

		from, err := Sender(recoverySigner, signedTx)

		require.Nil(t, err, err)
		require.True(t, from == addr, fmt.Sprintf("Expected from and address to be equal. Got %x want %x", from, addr))
	}

}


func TestSignSmiloHomesteadOnlyVaultSmilo(t *testing.T) {


	keys := []*big.Int{k0v, k1v}

	homeSinger := HomesteadSigner{}
	recoverySigner := HomesteadSigner{}

	for i := 0; i < len(keys); i++ {
		key, _ := createKey(crypto.S256(), keys[i])
		signedTx, addr, err := signTx(key, homeSinger)

		require.Nil(t, err, err)

		signedTx.SetVault()
		require.True(t, signedTx.IsVault(), fmt.Sprintf("Expected the transaction to be private [%v]", signedTx.IsVault()))

		from, err := Sender(recoverySigner, signedTx)

		require.Nil(t, err, err)
		require.True(t, from == addr, fmt.Sprintf("Expected from and address to be equal. Got %x want %x", from, addr))
	}

}

func TestSignSmiloEIP155OnlyVaultSmilo(t *testing.T) {

	keys := []*big.Int{k0v, k1v}

	EIP155Signer := NewEIP155Signer(big.NewInt(0))

	for i := 0; i < len(keys); i++ {
		key, _ := createKey(crypto.S256(), keys[i])
		signedTx, addr, err := signTx(key, EIP155Signer)

		require.Nil(t, err, err)

		signedTx.SetVault()

		require.True(t, signedTx.IsVault(), fmt.Sprintf("Expected the transaction to be private [%v]", signedTx.IsVault()))

		from, err := Sender(EIP155Signer, signedTx)

		require.Nil(t, err, err)
		require.False(t, from == addr, fmt.Sprintf("Expected recovery to fail. from [%x] should not equal addr [%x]", from, addr))
	}

}
