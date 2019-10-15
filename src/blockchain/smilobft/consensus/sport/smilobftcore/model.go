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
	"go-smilo/src/blockchain/smilobft/cmn"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"

	"go-smilo/src/blockchain/smilobft/consensus/sport"
)

// ----------------------------------------------------------------------------

type core struct {
	config  *sport.Config
	address common.Address
	state   State
	logger  log.Logger

	backend               sport.Backend
	events                *cmn.TypeMuxSubscription
	finalCommittedSub     *cmn.TypeMuxSubscription
	timeoutSub            *cmn.TypeMuxSubscription
	futurePreprepareTimer *time.Timer

	fullnodeSet           sport.FullnodeSet
	waitingForRoundChange bool
	validateFn            func([]byte, []byte) (common.Address, error)

	backlogs   map[sport.Fullnode]*prque.Prque
	backlogsMu *sync.Mutex

	current   *roundState
	handlerStopCh chan struct{}

	roundChangeSet   *roundChangeSet
	roundChangeTimer *time.Timer
	roundChangeTimerMu sync.RWMutex

	pendingRequests   *prque.Prque
	pendingRequestsMu *sync.Mutex

	consensusTimestamp time.Time
	// the meter to record the round change rate
	roundMeter metrics.Meter
	// the meter to record the sequence update rate
	sequenceMeter metrics.Meter
	// the timer to record consensus duration (from accepting a preprepare to final committed stage)
	consensusTimer metrics.Timer

	sentPreprepare bool
}

// New creates an smilobft consensus core
func New(backend sport.Backend, config *sport.Config) Engine {
	c := &core{
		config:             config,
		address:            backend.Address(),
		state:              StateAcceptRequest,
		handlerStopCh:      make(chan struct{}),
		logger:             log.New("address", backend.Address()),
		backend:            backend,
		backlogs:           make(map[sport.Fullnode]*prque.Prque),
		backlogsMu:         new(sync.Mutex),
		pendingRequests:    prque.New(),
		pendingRequestsMu:  new(sync.Mutex),
		consensusTimestamp: time.Time{},
		roundMeter:         metrics.NewRegisteredMeter("consensus/sport/smilobftcore/round", nil),
		sequenceMeter:      metrics.NewRegisteredMeter("consensus/sport/smilobftcore/sequence", nil),
		consensusTimer:     metrics.NewRegisteredTimer("consensus/sport/smilobftcore/consensus", nil),
	}

	c.validateFn = c.checkFullnodeSignature
	return c
}

// ----------------------------------------------------------------------------

// backlogEvent struct for sport.Fullnode and message
type backlogEvent struct {
	src sport.Fullnode
	msg *message
}

// timeoutEvent struct
type timeoutEvent struct{}

// ----------------------------------------------------------------------------

// messageSet struct for mutex, sport.FullnodeSet and messages for sport.View
type messageSet struct {
	view        *sport.View
	fullnodeSet sport.FullnodeSet
	messagesMu  *sync.Mutex
	messages    map[common.Address]*message
}

// newMessageSet Construct a new message set to accumulate messages for given sequence/view number.
func newMessageSet(fullnodeSet sport.FullnodeSet) *messageSet {
	return &messageSet{
		view: &sport.View{
			Round:    new(big.Int),
			Sequence: new(big.Int),
		},
		messagesMu:  new(sync.Mutex),
		messages:    make(map[common.Address]*message),
		fullnodeSet: fullnodeSet,
	}
}

// ----------------------------------------------------------------------------

// roundChangeSet struct for mutex, sport.FullnodeSet and roundChanges messageSet
type roundChangeSet struct {
	fullnodeSet  sport.FullnodeSet
	roundChanges map[uint64]*messageSet
	mu           *sync.Mutex
}

// newRoundChangeSet create new roundChangeSet based on @fullnodeSet
func newRoundChangeSet(fullnodeSet sport.FullnodeSet) *roundChangeSet {
	return &roundChangeSet{
		fullnodeSet:  fullnodeSet,
		roundChanges: make(map[uint64]*messageSet),
		mu:           new(sync.Mutex),
	}
}

// ----------------------------------------------------------------------------

// roundState stores the consensus state
type roundState struct {
	round          *big.Int
	sequence       *big.Int
	Preprepare     *sport.Preprepare
	Prepares       *messageSet
	Commits        *messageSet
	lockedHash     common.Hash
	pendingRequest *sport.Request

	mu                  *sync.RWMutex
	hasBadBlockProposal func(hash common.Hash) bool
}

// newRoundState creates a new roundState instance with the given view and FullnodeSet
// lockedHash and preprepare are for round change when lock exists,
// we need to keep a reference of preprepare in order to propose locked proposal when there is a lock and itself is the speaker
func newRoundState(view *sport.View, fullnodeSet sport.FullnodeSet, lockedHash common.Hash, preprepare *sport.Preprepare, pendingRequest *sport.Request, hasBadProposal func(hash common.Hash) bool) *roundState {
	return &roundState{
		round:               view.Round,
		sequence:            view.Sequence,
		Preprepare:          preprepare,
		Prepares:            newMessageSet(fullnodeSet),
		Commits:             newMessageSet(fullnodeSet),
		lockedHash:          lockedHash,
		mu:                  new(sync.RWMutex),
		pendingRequest:      pendingRequest,
		hasBadBlockProposal: hasBadProposal,
	}
}

// ----------------------------------------------------------------------------

// Engine consensus methods
type Engine interface {
	Start() error
	Stop() error

}

// ----------------------------------------------------------------------------

// message fields for message
type message struct {
	Code          uint64
	Msg           []byte
	Address       common.Address
	Signature     []byte
	CommittedSeal []byte
}
