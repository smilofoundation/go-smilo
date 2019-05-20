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

package faulty

import (
	"bytes"
	"go-smilo/src/blockchain/smilobft/consensus/sport"
	"math"
	"math/big"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"go-smilo/src/blockchain/smilobft/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
)

// New creates an sport consensus core
func New(backend sport.Backend, config *sport.Config) Engine {
	c := &core{
		config:             config,
		address:            backend.Address(),
		state:              StateAcceptRequest,
		handlerWg:          new(sync.WaitGroup),
		logger:             log.New("address", backend.Address()),
		backend:            backend,
		backlogs:           make(map[sport.Fullnode]*prque.Prque),
		backlogsMu:         new(sync.Mutex),
		pendingRequests:    prque.New(),
		pendingRequestsMu:  new(sync.Mutex),
		consensusTimestamp: time.Time{},
		//roundMeter:         metrics.NewMeter("consensus/sport/core/round"),
		//sequenceMeter:      metrics.NewMeter("consensus/sport/core/sequence"),
		//consensusTimer:     metrics.NewTimer("consensus/sport/core/consensus"),
	}
	c.validateFn = c.checkFullnodeSignature
	return c
}

// ----------------------------------------------------------------------------

type core struct {
	config  *sport.Config
	address common.Address
	state   State
	logger  log.Logger

	backend               sport.Backend
	events                *event.TypeMuxSubscription
	finalCommittedSub     *event.TypeMuxSubscription
	timeoutSub            *event.TypeMuxSubscription
	futurePreprepareTimer *time.Timer

	fullnodeSet                sport.FullnodeSet
	waitingForRoundChange bool
	validateFn            func([]byte, []byte) (common.Address, error)

	backlogs   map[sport.Fullnode]*prque.Prque
	backlogsMu *sync.Mutex

	current   *roundState
	handlerWg *sync.WaitGroup

	roundChangeSet   *roundChangeSet
	roundChangeTimer *time.Timer

	pendingRequests   *prque.Prque
	pendingRequestsMu *sync.Mutex

	consensusTimestamp time.Time
	// the meter to record the round change rate
	roundMeter metrics.Meter
	// the meter to record the sequence update rate
	sequenceMeter metrics.Meter
	// the timer to record consensus duration (from accepting a preprepare to final committed stage)
	consensusTimer metrics.Timer
}

func (c *core) finalizeMessage(msg *message) ([]byte, error) {
	var err error
	// Add sender address
	msg.Address = c.Address()

	// Add proof of consensus
	msg.CommittedSeal = []byte{}
	// Assign the CommittedSeal if it's a COMMIT message and proposal is not nil
	if msg.Code == msgCommit && c.current.Proposal() != nil {
		seal := PrepareCommittedSeal(c.current.Proposal().Hash())
		msg.CommittedSeal, err = c.backend.Sign(seal)
		if err != nil {
			return nil, err
		}
	}

	// Sign message
	data, err := msg.PayloadNoSig()
	if err != nil {
		return nil, err
	}
	msg.Signature, err = c.backend.Sign(data)
	if err != nil {
		return nil, err
	}

	if c.modifySig() {
		c.logger.Info("Modify the signature")
		str := "fake"
		copy(msg.Signature[:len(str)], []byte(str)[:])
	}
	// Convert to payload
	payload, err := msg.Payload()
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func (c *core) broadcast(msg *message) {
	logger := c.logger.New("state", c.state)

	if c.notBroadcast() {
		logger.Info("Not broadcast message", "message", msg)
		return
	}

	if c.sendWrongMsg() {
		code := uint64(rand.Intn(4))
		logger.Info("Modify the message code", "old", msg.Code, "new", code)
		msg.Code = code
	}

	payload, err := c.finalizeMessage(msg)
	if err != nil {
		logger.Error("Failed to finalize message", "msg", msg, "err", err)
		return
	}

	// Broadcast payload
	if err = c.backend.Broadcast(c.fullnodeSet, payload); err != nil {
		logger.Error("Failed to broadcast message", "msg", msg, "err", err)
		return
	}
}

func (c *core) currentView() *sport.View {
	return &sport.View{
		Sequence: new(big.Int).Set(c.current.Sequence()),
		Round:    new(big.Int).Set(c.current.Round()),
	}
}

func (c *core) isProposer() bool {
	v := c.fullnodeSet
	if v == nil {
		return false
	}
	return v.IsSpeaker(c.backend.Address())
}

func (c *core) commit() {
	c.setState(StateCommitted)

	proposal := c.current.Proposal()
	if proposal != nil {
		committedSeals := make([][]byte, c.current.Commits.Size())
		for i, v := range c.current.Commits.Values() {
			committedSeals[i] = make([]byte, types.SportExtraSeal)
			copy(committedSeals[i][:], v.CommittedSeal[:])
		}

		if err := c.backend.Commit(proposal, committedSeals); err != nil {
			c.current.UnlockHash() //Unlock block when insertion fails
			c.sendNextRoundChange()
			return
		}
	}
}

// startNewRound starts a new round. if round equals to 0, it means to starts a new sequence
func (c *core) startNewRound(round *big.Int) {
	var logger log.Logger
	if c.current == nil {
		logger = c.logger.New("old_round", -1, "old_seq", 0)
	} else {
		logger = c.logger.New("old_round", c.current.Round(), "old_seq", c.current.Sequence())
	}

	roundChange := false
	// Try to get last proposal
	lastProposal, lastProposer := c.backend.LastBlockProposal()
	if c.current == nil {
		logger.Trace("Start to the initial round")
	} else if lastProposal.Number().Cmp(c.current.Sequence()) >= 0 {
		diff := new(big.Int).Sub(lastProposal.Number(), c.current.Sequence())
		c.sequenceMeter.Mark(new(big.Int).Add(diff, common.Big1).Int64())

		if !c.consensusTimestamp.IsZero() {
			c.consensusTimer.UpdateSince(c.consensusTimestamp)
			c.consensusTimestamp = time.Time{}
		}
		logger.Trace("Catch up latest proposal", "number", lastProposal.Number().Uint64(), "hash", lastProposal.Hash())
	} else if lastProposal.Number().Cmp(big.NewInt(c.current.Sequence().Int64()-1)) == 0 {
		if round.Cmp(common.Big0) == 0 {
			// same seq and round, don't need to start new round
			return
		} else if round.Cmp(c.current.Round()) < 0 {
			logger.Warn("New round should not be smaller than current round", "seq", lastProposal.Number().Int64(), "new_round", round, "old_round", c.current.Round())
			return
		}
		roundChange = true
	} else {
		logger.Warn("New sequence should be larger than current sequence", "new_seq", lastProposal.Number().Int64())
		return
	}

	var newView *sport.View
	if roundChange {
		newView = &sport.View{
			Sequence: new(big.Int).Set(c.current.Sequence()),
			Round:    new(big.Int).Set(round),
		}
	} else {
		newView = &sport.View{
			Sequence: new(big.Int).Add(lastProposal.Number(), common.Big1),
			Round:    new(big.Int),
		}
		c.fullnodeSet = c.backend.Fullnodes(lastProposal)
	}

	// Update logger
	logger = logger.New("old_proposer", c.fullnodeSet.GetSpeaker())
	// Clear invalid ROUND CHANGE messages
	c.roundChangeSet = newRoundChangeSet(c.fullnodeSet)
	// New snapshot for new round
	c.updateRoundState(newView, c.fullnodeSet, roundChange)
	// Calculate new proposer
	c.fullnodeSet.CalcSpeaker(lastProposer, newView.Round.Uint64())
	c.waitingForRoundChange = false
	c.setState(StateAcceptRequest)
	if roundChange && c.isProposer() && c.current != nil {
		// If it is locked, propose the old proposal
		// If we have pending request, propose pending request
		if c.current.IsHashLocked() {
			r := &sport.Request{
				BlockProposal: c.current.Proposal(), //c.current.Proposal would be the locked proposal by previous proposer, see updateRoundState
			}
			c.sendPreprepare(r)
		} else if c.current.pendingRequest != nil {
			c.sendPreprepare(c.current.pendingRequest)
		}
	}
	c.newRoundChangeTimer()

	logger.Debug("New round", "new_round", newView.Round, "new_seq", newView.Sequence, "new_proposer", c.fullnodeSet.GetSpeaker(), "fullnodeSet", c.fullnodeSet.List(), "size", c.fullnodeSet.Size(), "isProposer", c.isProposer())
}

func (c *core) catchUpRound(view *sport.View) {
	logger := c.logger.New("old_round", c.current.Round(), "old_seq", c.current.Sequence(), "old_proposer", c.fullnodeSet.GetSpeaker())

	if view.Round.Cmp(c.current.Round()) > 0 {
		c.roundMeter.Mark(new(big.Int).Sub(view.Round, c.current.Round()).Int64())
	}
	c.waitingForRoundChange = true

	// Need to keep block locked for round catching up
	c.updateRoundState(view, c.fullnodeSet, true)
	c.roundChangeSet.Clear(view.Round)
	c.newRoundChangeTimer()

	logger.Trace("Catch up round", "new_round", view.Round, "new_seq", view.Sequence, "new_proposer", c.fullnodeSet)
}

// updateRoundState updates round state by checking if locking block is necessary
func (c *core) updateRoundState(view *sport.View, fullnodeSet sport.FullnodeSet, roundChange bool) {
	// Lock only if both roundChange is true and it is locked
	if roundChange && c.current != nil {
		if c.current.IsHashLocked() {
			c.current = newRoundState(view, fullnodeSet, c.current.GetLockedHash(), c.current.Preprepare, c.current.pendingRequest, c.backend.HasBadBlockProposal)
		} else {
			c.current = newRoundState(view, fullnodeSet, common.Hash{}, nil, c.current.pendingRequest, c.backend.HasBadBlockProposal)
		}
	} else {
		c.current = newRoundState(view, fullnodeSet, common.Hash{}, nil, nil, c.backend.HasBadBlockProposal)
	}
}

func (c *core) setState(state State) {
	if c.state != state {
		c.state = state
	}
	if state == StateAcceptRequest {
		c.processPendingRequests()
	}
	c.processBacklog()
}

func (c *core) Address() common.Address {
	return c.address
}

func (c *core) stopFuturePreprepareTimer() {
	if c.futurePreprepareTimer != nil {
		c.futurePreprepareTimer.Stop()
	}
}

func (c *core) stopTimer() {
	c.stopFuturePreprepareTimer()
	if c.roundChangeTimer != nil {
		c.roundChangeTimer.Stop()
	}
}

func (c *core) newRoundChangeTimer() {
	c.stopTimer()

	// set timeout based on the round number
	timeout := time.Duration(c.config.RequestTimeout) * time.Millisecond
	round := c.current.Round().Uint64()
	if round > 0 {
		timeout += time.Duration(math.Pow(2, float64(round))) * time.Second
	}

	c.roundChangeTimer = time.AfterFunc(timeout, func() {
		c.sendEvent(timeoutEvent{})
	})
}

func (c *core) checkFullnodeSignature(data []byte, sig []byte) (common.Address, error) {
	return sport.CheckFullnodeSignature(c.fullnodeSet, data, sig)
}

// PrepareCommittedSeal returns a committed seal for the given hash
func PrepareCommittedSeal(hash common.Hash) []byte {
	var buf bytes.Buffer
	buf.Write(hash.Bytes())
	buf.Write([]byte{byte(msgCommit)})
	return buf.Bytes()
}



func (c *core) IsSpeaker() bool {
	v := c.fullnodeSet
	if v == nil {
		return false
	}
	return v.IsSpeaker(c.backend.Address())
}

func (c *core) IsCurrentBlockProposal(blockHash common.Hash) bool {
	if c.current == nil || c.current.pendingRequest == nil || c.current.pendingRequest.BlockProposal == nil {
		log.Error("&*&*&*&*&* IsCurrentBlockProposal, could not evaluate complete object c.current.pendingRequest.BlockProposal, ", "c.current", c.current)
		return false
	}

	isPending := c.current.pendingRequest != nil
	isCurrentBlockProposal := c.current.pendingRequest.BlockProposal.Hash() == blockHash

	return isPending && isCurrentBlockProposal
}
