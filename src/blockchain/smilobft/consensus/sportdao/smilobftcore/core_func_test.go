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

package smilobftcore

import (
	"crypto/ecdsa"
	"math/big"
	"time"

	"go-smilo/src/blockchain/smilobft/core/rawdb"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	elog "github.com/ethereum/go-ethereum/log"

	"go-smilo/src/blockchain/smilobft/cmn"
	"go-smilo/src/blockchain/smilobft/consensus/sportdao"
	"go-smilo/src/blockchain/smilobft/consensus/sportdao/fullnode"
	"go-smilo/src/blockchain/smilobft/core/types"
	"go-smilo/src/blockchain/smilobft/ethdb"
)

var testLogger = elog.New()

type testSystemBackend struct {
	id  int
	sys *testSystem

	engine Engine
	peers  sportdao.FullnodeSet
	events *cmn.TypeMux

	committedMsgs []testCommittedMsgs
	sentMsgs      [][]byte // store the message when Send is called by core

	address common.Address
	db      ethdb.Database
}

type testCommittedMsgs struct {
	commitBlockProposal sportdao.BlockProposal
	committedSeals      [][]byte
}

// ==============================================
//
// define the functions that needs to be provided for sportdao.

func (b *testSystemBackend) Address() common.Address {
	return b.address
}

// Peers returns all connected peers
func (b *testSystemBackend) Fullnodes(number uint64) sportdao.FullnodeSet {
	return b.peers
}

func (b *testSystemBackend) EventMux() *cmn.TypeMux {
	return b.events
}

func (b *testSystemBackend) Send(message []byte, target common.Address) error {
	testLogger.Info("enqueuing a message...", "address", b.Address())
	b.sentMsgs = append(b.sentMsgs, message)
	b.sys.queuedMessage <- sportdao.MessageEvent{
		Payload: message,
	}
	return nil
}

func (b *testSystemBackend) Broadcast(fullnodeSet sportdao.FullnodeSet, message []byte) error {
	testLogger.Info("enqueuing a message...", "address", b.Address())
	b.sentMsgs = append(b.sentMsgs, message)
	b.sys.queuedMessage <- sportdao.MessageEvent{
		Payload: message,
	}
	return nil
}

func (b *testSystemBackend) Gossip(fullnodeSet sportdao.FullnodeSet, message []byte) error {
	testLogger.Warn("not sign any data")
	return nil
}

func (b *testSystemBackend) Commit(proposal sportdao.BlockProposal, seals [][]byte) error {
	testLogger.Info("commit message", "address", b.Address())
	b.committedMsgs = append(b.committedMsgs, testCommittedMsgs{
		commitBlockProposal: proposal,
		committedSeals:      seals,
	})

	// fake new head events
	go b.events.Post(sportdao.FinalCommittedEvent{})
	return nil
}

func (b *testSystemBackend) Verify(proposal sportdao.BlockProposal) (time.Duration, error) {
	return 0, nil
}

func (b *testSystemBackend) Sign(data []byte) ([]byte, error) {
	testLogger.Warn("not sign any data")
	return data, nil
}

func (b *testSystemBackend) CheckSignature([]byte, common.Address, []byte) error {
	return nil
}

func (b *testSystemBackend) CheckFullnodeSignature(data []byte, sig []byte) (common.Address, error) {
	return common.Address{}, nil
}

func (b *testSystemBackend) Hash(i interface{}) common.Hash {
	return cmn.StringToHash("Test")
}

func (b *testSystemBackend) NewRequest(request sportdao.BlockProposal) {
	go b.events.Post(sportdao.RequestEvent{
		BlockProposal: request,
	})
}

func (b *testSystemBackend) HasBadBlockProposal(hash common.Hash) bool {
	return false
}

func (b *testSystemBackend) LastBlockProposal() (sportdao.BlockProposal, common.Address) {
	l := len(b.committedMsgs)
	if l > 0 {
		return b.committedMsgs[l-1].commitBlockProposal, common.Address{}
	}
	return makeBlock(0), common.Address{}
}

// Only block height 5 will return true
func (b *testSystemBackend) HasBlockProposal(hash common.Hash, number *big.Int) bool {
	return number.Cmp(big.NewInt(5)) == 0
}

func (b *testSystemBackend) GetSpeaker(number uint64) common.Address {
	return common.Address{}
}

func (b *testSystemBackend) SetProposedBlockHash(hash common.Hash) {
	return
}

func (b *testSystemBackend) ParentFullnodes(proposal sportdao.BlockProposal) sportdao.FullnodeSet {
	return b.peers
}

func (b *testSystemBackend) Close() error {
	return nil
}

// ==============================================
//
// define the struct that need to be provided for integration tests.

type testSystem struct {
	backends []*testSystemBackend

	queuedMessage chan sportdao.MessageEvent
	quit          chan struct{}
}

func newTestSystem(availableNodes int) *testSystem {
	testLogger.SetHandler(elog.StdoutHandler)
	return &testSystem{
		backends: make([]*testSystemBackend, availableNodes),

		queuedMessage: make(chan sportdao.MessageEvent),
		quit:          make(chan struct{}),
	}
}

func generateFullnodes(availableNodes int) []common.Address {
	vals := make([]common.Address, 0)
	for i := 0; i < availableNodes; i++ {
		privateKey, _ := crypto.GenerateKey()
		vals = append(vals, crypto.PubkeyToAddress(privateKey.PublicKey))
	}
	return vals
}

func newTestFullnodeSet(n int) sportdao.FullnodeSet {
	return fullnode.NewFullnodeSet(generateFullnodes(n), sportdao.RoundRobin)
}

func NewTestSystemWithBackend(availableNodes int) *testSystem {
	testLogger.SetHandler(elog.StdoutHandler)

	addrs := generateFullnodes(availableNodes)
	sys := newTestSystem(availableNodes)
	config := sportdao.DefaultConfig

	for i := 0; i < availableNodes; i++ {
		vset := fullnode.NewFullnodeSet(addrs, sportdao.RoundRobin)
		backend := sys.NewBackend(i)
		backend.peers = vset
		backend.address = vset.GetByIndex(uint64(i)).Address()

		core := New(backend, config).(*core)
		core.state = StateAcceptRequest
		core.current = newRoundState(&sportdao.View{
			Round:    big.NewInt(0),
			Sequence: big.NewInt(1),
		}, vset, common.Hash{}, nil, nil, func(hash common.Hash) bool {
			return false
		})
		core.fullnodeSet = vset
		core.logger = testLogger
		core.validateFn = backend.CheckFullnodeSignature

		backend.engine = core
	}

	return sys
}

// listen will consume messages from queue and deliver a message to core
func (t *testSystem) listen() {
	for {
		select {
		case <-t.quit:
			return
		case queuedMessage := <-t.queuedMessage:
			testLogger.Info("consuming a queue message...")
			for _, backend := range t.backends {
				go backend.EventMux().Post(queuedMessage)
			}
		}
	}
}

// Run will start system components based on given flag, and returns a closer
// function that caller can control lifecycle
//
// Given a true for core if you want to initialize core engine.
func (t *testSystem) Run(core bool) func() {
	for _, b := range t.backends {
		if core {
			b.engine.Start() // start smilobft core
		}
	}

	go t.listen()
	closer := func() { t.stop(core) }
	return closer
}

func (t *testSystem) stop(core bool) {
	close(t.quit)

	for _, b := range t.backends {
		if core {
			b.engine.Stop()
		}
	}
}

func (t *testSystem) NewBackend(id int) *testSystemBackend {
	// assume always success
	ethDB := rawdb.NewMemoryDatabase()
	backend := &testSystemBackend{
		id:     id,
		sys:    t,
		events: new(cmn.TypeMux),
		db:     ethDB,
	}

	t.backends[id] = backend
	return backend
}

// ==============================================
//
// helper functions.

func getPublicKeyAddress(privateKey *ecdsa.PrivateKey) common.Address {
	return crypto.PubkeyToAddress(privateKey.PublicKey)
}

func makeBlock(number int64) *types.Block {
	header := &types.Header{
		Difficulty: big.NewInt(0),
		Number:     big.NewInt(number),
		GasLimit:   0,
		GasUsed:    0,
		Time:       0,
	}
	block := &types.Block{}
	return block.WithSeal(header)
}

func newTestBlockProposal() sportdao.BlockProposal {
	return makeBlock(1)
}
