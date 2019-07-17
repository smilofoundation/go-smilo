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
	"go-smilo/src/blockchain/smilobft/core/rawdb"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	elog "github.com/ethereum/go-ethereum/log"

	"go-smilo/src/blockchain/smilobft/cmn"
	"go-smilo/src/blockchain/smilobft/consensus/sport"
	"go-smilo/src/blockchain/smilobft/consensus/sport/fullnode"
	"go-smilo/src/blockchain/smilobft/core/types"
	"go-smilo/src/blockchain/smilobft/ethdb"
)

var testLogger = elog.New()

type testSystemBackend struct {
	id  int
	sys *testSystem

	engine Engine
	peers  sport.FullnodeSet
	events *event.TypeMux

	committedMsgs []testCommittedMsgs
	sentMsgs      [][]byte // store the message when Send is called by core

	address common.Address
	db      ethdb.Database
}

type testCommittedMsgs struct {
	commitBlockProposal sport.BlockProposal
	committedSeals      [][]byte
}

// ==============================================
//
// define the functions that needs to be provided for sport.

func (self *testSystemBackend) Address() common.Address {
	return self.address
}

// Peers returns all connected peers
func (self *testSystemBackend) Fullnodes(proposal sport.BlockProposal) sport.FullnodeSet {
	return self.peers
}

func (self *testSystemBackend) EventMux() *event.TypeMux {
	return self.events
}

func (self *testSystemBackend) Send(message []byte, target common.Address) error {
	testLogger.Info("enqueuing a message...", "address", self.Address())
	self.sentMsgs = append(self.sentMsgs, message)
	self.sys.queuedMessage <- sport.MessageEvent{
		Payload: message,
	}
	return nil
}

func (self *testSystemBackend) Broadcast(fullnodeSet sport.FullnodeSet, message []byte) error {
	testLogger.Info("enqueuing a message...", "address", self.Address())
	self.sentMsgs = append(self.sentMsgs, message)
	self.sys.queuedMessage <- sport.MessageEvent{
		Payload: message,
	}
	return nil
}

func (self *testSystemBackend) Gossip(fullnodeSet sport.FullnodeSet, message []byte) error {
	testLogger.Warn("not sign any data")
	return nil
}

func (self *testSystemBackend) Commit(proposal sport.BlockProposal, seals [][]byte) error {
	testLogger.Info("commit message", "address", self.Address())
	self.committedMsgs = append(self.committedMsgs, testCommittedMsgs{
		commitBlockProposal: proposal,
		committedSeals:      seals,
	})

	// fake new head events
	go self.events.Post(sport.FinalCommittedEvent{})
	return nil
}

func (self *testSystemBackend) Verify(proposal sport.BlockProposal) (time.Duration, error) {
	return 0, nil
}

func (self *testSystemBackend) Sign(data []byte) ([]byte, error) {
	testLogger.Warn("not sign any data")
	return data, nil
}

func (self *testSystemBackend) CheckSignature([]byte, common.Address, []byte) error {
	return nil
}

func (self *testSystemBackend) CheckFullnodeSignature(data []byte, sig []byte) (common.Address, error) {
	return common.Address{}, nil
}

func (self *testSystemBackend) Hash(b interface{}) common.Hash {
	return cmn.StringToHash("Test")
}

func (self *testSystemBackend) NewRequest(request sport.BlockProposal) {
	go self.events.Post(sport.RequestEvent{
		BlockProposal: request,
	})
}

func (self *testSystemBackend) HasBadBlockProposal(hash common.Hash) bool {
	return false
}

func (self *testSystemBackend) LastBlockProposal() (sport.BlockProposal, common.Address) {
	l := len(self.committedMsgs)
	if l > 0 {
		return self.committedMsgs[l-1].commitBlockProposal, common.Address{}
	}
	return makeBlock(0), common.Address{}
}

// Only block height 5 will return true
func (self *testSystemBackend) HasBlockProposal(hash common.Hash, number *big.Int) bool {
	return number.Cmp(big.NewInt(5)) == 0
}

func (self *testSystemBackend) GetSpeaker(number uint64) common.Address {
	return common.Address{}
}

func (self *testSystemBackend) ParentFullnodes(proposal sport.BlockProposal) sport.FullnodeSet {
	return self.peers
}

func (sb *testSystemBackend) Close() error {
	return nil
}

// ==============================================
//
// define the struct that need to be provided for integration tests.

type testSystem struct {
	backends []*testSystemBackend

	queuedMessage chan sport.MessageEvent
	quit          chan struct{}
}

func newTestSystem(availableNodes int) *testSystem {
	testLogger.SetHandler(elog.StdoutHandler)
	return &testSystem{
		backends: make([]*testSystemBackend, availableNodes),

		queuedMessage: make(chan sport.MessageEvent),
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

func newTestFullnodeSet(n int) sport.FullnodeSet {
	return fullnode.NewFullnodeSet(generateFullnodes(n), sport.RoundRobin)
}

func NewTestSystemWithBackend(availableNodes int) *testSystem {
	testLogger.SetHandler(elog.StdoutHandler)

	addrs := generateFullnodes(availableNodes)
	sys := newTestSystem(availableNodes)
	config := sport.DefaultConfig

	for i := 0; i < availableNodes; i++ {
		vset := fullnode.NewFullnodeSet(addrs, sport.RoundRobin)
		backend := sys.NewBackend(i)
		backend.peers = vset
		backend.address = vset.GetByIndex(uint64(i)).Address()

		core := New(backend, config).(*core)
		core.state = StateAcceptRequest
		core.current = newRoundState(&sport.View{
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
		events: new(event.TypeMux),
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
		Time:       big.NewInt(0).Uint64(),
	}
	block := &types.Block{}
	return block.WithSeal(header)
}

func newTestBlockProposal() sport.BlockProposal {
	return makeBlock(1)
}
