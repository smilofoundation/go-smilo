// Copyright 2019 The go-smilo Authors
// Copyright 2017 The go-ethereum Authors
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

package fullnode

import (
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/stretchr/testify/require"

	"go-smilo/src/blockchain/smilobft/cmn"
	"go-smilo/src/blockchain/smilobft/consensus/sportdao"
)

var (
	testAddress  = "70524d664ffe731100208a0154e556f9bb679ae6"
	testAddress2 = "b37866a925bccd69cfa98d43b510f1d23d78a851"
)

func TestNewFullnodeSet(t *testing.T) {
	var fullnodes []sportdao.Fullnode
	const ValCnt = 1000

	// Create 1000 fullnodes with random addresses
	extraData := []byte{}
	for i := 0; i < ValCnt; i++ {
		key, _ := crypto.GenerateKey()
		addr := crypto.PubkeyToAddress(key.PublicKey)
		val := NewFullNode(addr)
		fullnodes = append(fullnodes, val)
		extraData = append(extraData, val.Address().Bytes()...)
	}

	require.True(t, ValidateExtraData(extraData))

	// Create FullnodeSet

	fullnodeSet := NewFullnodeSet(ExtractFullnodes(extraData), sportdao.RoundRobin)
	if fullnodeSet == nil {
		t.Errorf("the fullnode byte array cannot be parsed")
		t.FailNow()
	}

	require.True(t, fullnodeSet.Size() == ValCnt)

	// Check fullnodes sorting: should be in ascending order
	for i := 0; i < ValCnt-1; i++ {
		val := fullnodeSet.GetByIndex(uint64(i))
		nextVal := fullnodeSet.GetByIndex(uint64(i + 1))
		if strings.Compare(val.String(), nextVal.String()) >= 0 {
			t.Errorf("fullnode set is not sorted in ascending order")
		}
	}
}

func TestNormalFullnodeSet(t *testing.T) {
	b1 := cmn.Hex2Bytes(testAddress)
	b2 := cmn.Hex2Bytes(testAddress2)
	addr1 := common.BytesToAddress(b1)
	addr2 := common.BytesToAddress(b2)
	val1 := NewFullNode(addr1)
	val2 := NewFullNode(addr2)

	fullnodeSet := newFullnodeSet([]common.Address{addr1, addr2}, sportdao.RoundRobin)
	if fullnodeSet == nil {
		t.Errorf("the format of fullnode set is invalid")
		t.FailNow()
	}

	// check size
	if size := fullnodeSet.Size(); size != 2 {
		t.Errorf("the size of fullnode set is wrong: have %v, want 2", size)
	}
	// test get by index
	if val := fullnodeSet.GetByIndex(uint64(0)); !reflect.DeepEqual(val, val1) {
		t.Errorf("fullnode mismatch: have %v, want %v", val, val1)
	}
	// test get by invalid index
	if val := fullnodeSet.GetByIndex(uint64(2)); val != nil {
		t.Errorf("fullnode mismatch: have %v, want nil", val)
	}
	// test get by address
	if _, val := fullnodeSet.GetByAddress(addr2); !reflect.DeepEqual(val, val2) {
		t.Errorf("fullnode mismatch: have %v, want %v", val, val2)
	}
	// test get by invalid address
	invalidAddr := cmn.HexToAddress("0x9535b2e7faaba5288511d89341d94a38063a349b")
	if _, val := fullnodeSet.GetByAddress(invalidAddr); val != nil {
		t.Errorf("fullnode mismatch: have %v, want nil", val)
	}
	// test get speaker
	if val := fullnodeSet.GetSpeaker(); !reflect.DeepEqual(val, val1) {
		t.Errorf("speaker mismatch: have %v, want %v", val, val1)
	}
	// test calculate speaker
	lastSpeaker := addr1
	fullnodeSet.CalcSpeaker(lastSpeaker, uint64(0))
	if val := fullnodeSet.GetSpeaker(); !reflect.DeepEqual(val, val2) {
		t.Errorf("speaker mismatch: have %v, want %v", val, val2)
	}
	fullnodeSet.CalcSpeaker(lastSpeaker, uint64(3))
	if val := fullnodeSet.GetSpeaker(); !reflect.DeepEqual(val, val1) {
		t.Errorf("speaker mismatch: have %v, want %v", val, val1)
	}
	// test empty last speaker
	lastSpeaker = common.Address{}
	fullnodeSet.CalcSpeaker(lastSpeaker, uint64(3))
	if val := fullnodeSet.GetSpeaker(); !reflect.DeepEqual(val, val2) {
		t.Errorf("speaker mismatch: have %v, want %v", val, val2)
	}
}

func TestEmptyFullnodeSet(t *testing.T) {
	extraData := []byte{}
	require.True(t, ValidateExtraData(extraData))

	fullnodeSet := NewFullnodeSet(ExtractFullnodes(extraData), sportdao.RoundRobin)
	if fullnodeSet == nil {
		t.Errorf("fullnode set should not be nil")
	}
}

func TestAddAndRemoveFullnode(t *testing.T) {
	extraData := []byte{}
	require.True(t, ValidateExtraData(extraData), "Extra data should be valid")

	fullnodeSet := NewFullnodeSet(ExtractFullnodes(extraData), sportdao.RoundRobin)
	require.Len(t, fullnodeSet.List(), 0, "Fullnodeset should start empty")

	require.True(t, fullnodeSet.AddFullnode(cmn.StringToAddress(string(2))), "the fullnode 2 should be added")

	require.True(t, !fullnodeSet.AddFullnode(cmn.StringToAddress(string(2))), "the existing fullnode 2 should not be added")

	require.True(t, fullnodeSet.AddFullnode(cmn.StringToAddress(string(1))), "the fullnode 1 should be added")

	require.True(t, fullnodeSet.AddFullnode(cmn.StringToAddress(string(0))), "the fullnode 0 should be added")

	require.Len(t, fullnodeSet.List(), 3, "the size of fullnode set should be 3")

	for i, v := range fullnodeSet.List() {
		expected := cmn.StringToAddress(string(i))
		require.Equal(t, expected, v.Address(), "Full node found is not correct.")
	}

	require.True(t, fullnodeSet.RemoveFullnode(cmn.StringToAddress(string(2))), "the fullnode should be removed")

	require.True(t, !fullnodeSet.RemoveFullnode(cmn.StringToAddress(string(2))), "the non-existing fullnode should not be removed")

	require.Len(t, fullnodeSet.List(), 2, "the size of fullnode set should be 2")

	require.True(t, fullnodeSet.RemoveFullnode(cmn.StringToAddress(string(1))), "the fullnode should be removed")

	require.Len(t, fullnodeSet.List(), 1, "the size of fullnode set should be 1")

	require.True(t, fullnodeSet.RemoveFullnode(cmn.StringToAddress(string(0))), "the fullnode should be removed")

	require.Len(t, fullnodeSet.List(), 0, "the size of fullnode set should be 0")

}
