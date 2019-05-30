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
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"math"

	"go-smilo/src/blockchain/smilobft/consensus/sport"
	"go-smilo/src/blockchain/smilobft/core/types"
)

// ----------------------------------------------------------------------------

func (c *core) finalizeMessage(msg *message) ([]byte, error) {
	var err error
	// Add sender address
	msg.Address = c.Address()

	// Add proof of consensus
	msg.CommittedSeal = []byte{}
	// Assign the CommittedSeal if it's a COMMIT message and proposal is not nil
	if msg.Code == msgCommit && c.current.BlockProposal() != nil {
		seal := PrepareCommittedSeal(c.current.BlockProposal().Hash())
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

	// Convert to payload
	payload, err := msg.Payload()
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func (c *core) broadcast(msg *message) {
	logger := c.logger.New("state", c.state)

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

func (c *core) commit() {
	c.setState(StateCommitted)

	proposal := c.current.BlockProposal()
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
	lastBlockProposal, lastSpeaker := c.backend.LastBlockProposal()
	if c.current == nil {
		logger.Debug("Start to the initial round")
	} else if lastBlockProposal.Number().Cmp(c.current.Sequence()) >= 0 {
		diff := new(big.Int).Sub(lastBlockProposal.Number(), c.current.Sequence())
		c.sequenceMeter.Mark(new(big.Int).Add(diff, common.Big1).Int64())

		if !c.consensusTimestamp.IsZero() {
			c.consensusTimer.UpdateSince(c.consensusTimestamp)
			c.consensusTimestamp = time.Time{}
		}
		logger.Trace("Catch up latest proposal", "number", lastBlockProposal.Number().Uint64(), "hash", lastBlockProposal.Hash())
	} else if lastBlockProposal.Number().Cmp(big.NewInt(c.current.Sequence().Int64()-1)) == 0 {
		if round.Cmp(common.Big0) == 0 {
			// same seq and round, don't need to start new round
			return
		} else if round.Cmp(c.current.Round()) < 0 {
			logger.Warn("New round should not be smaller than current round", "seq", lastBlockProposal.Number().Int64(), "new_round", round, "old_round", c.current.Round())
			return
		}
		roundChange = true
	} else {
		logger.Warn("New sequence should be larger than current sequence", "new_seq", lastBlockProposal.Number().Int64())
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
			Sequence: new(big.Int).Add(lastBlockProposal.Number(), common.Big1),
			Round:    new(big.Int),
		}
		c.fullnodeSet = c.backend.Fullnodes(lastBlockProposal)
	}

	// Update logger
	logger = logger.New("old_speaker", c.fullnodeSet.GetSpeaker())
	// Clear invalid ROUND CHANGE messages
	c.roundChangeSet = newRoundChangeSet(c.fullnodeSet)
	// New snapshot for new round
	c.updateRoundState(newView, c.fullnodeSet, roundChange)
	// Calculate new speaker

	currentBlockHash := lastBlockProposal.Hash().Hex()
	if c.current.Preprepare != nil {
		currentBlockHash = c.current.Preprepare.BlockProposal.Hash().Hex()
	}

	c.fullnodeSet.CalcSpeaker(lastSpeaker, newView.Round.Uint64(), c.backend.GetPrivateKey(), currentBlockHash)
	c.waitingForRoundChange = false
	c.setState(StateAcceptRequest)
	if roundChange && c.IsSpeaker() && c.current != nil {
		// If it is locked, propose the old proposal
		// If we have pending request, propose pending request
		if c.current.IsHashLocked() {
			r := &sport.Request{
				BlockProposal: c.current.BlockProposal(), //c.current.BlockProposal would be the locked proposal by previous speaker, see updateRoundState
			}
			c.sendPreprepare(r)
		} else if c.current.pendingRequest != nil {
			c.sendPreprepare(c.current.pendingRequest)
		}
	}
	c.newRoundChangeTimer()

	logger.Debug("New round", "new_round", newView.Round, "new_seq", newView.Sequence, "speaker", c.fullnodeSet.GetSpeaker(), "fullnodeSet", c.fullnodeSet.List(), "size", c.fullnodeSet.Size(), "IsSpeaker", c.IsSpeaker())
}

func (c *core) catchUpRound(view *sport.View) {
	logger := c.logger.New("old_round", c.current.Round(), "old_seq", c.current.Sequence(), "speaker", c.fullnodeSet.GetSpeaker())

	if view.Round.Cmp(c.current.Round()) > 0 {
		c.roundMeter.Mark(new(big.Int).Sub(view.Round, c.current.Round()).Int64())
	}
	c.waitingForRoundChange = true

	// Need to keep block locked for round catching up
	c.updateRoundState(view, c.fullnodeSet, true)
	c.roundChangeSet.Clear(view.Round)
	c.newRoundChangeTimer()

	logger.Debug("Catch up round", "new_round", view.Round, "new_seq", view.Sequence, "fullnodeSet", c.fullnodeSet)
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
		thisTimeout := time.Duration(c.config.MaxTimeout) * time.Second
		if timeout > thisTimeout {
			timeout = thisTimeout
			c.logger.Debug("************* Round Timeout increased to maxTimeout", "maxTimeout", c.config.MaxTimeout, "timeout", timeout, "timeoutOriginal", time.Duration(c.config.RequestTimeout)*time.Millisecond)
		} else {
			c.logger.Debug("************* Round Timeout increased", "increase", time.Duration(math.Pow(2, float64(round)))*time.Second, "timeout", timeout, "timeoutOriginal", time.Duration(c.config.RequestTimeout)*time.Millisecond)
		}
	}

	c.roundChangeTimer = time.AfterFunc(timeout, func() {
		c.logger.Debug("newRoundChangeTimer, Timeout for round !", "round", round, "timeout", timeout, "timeoutOriginal", time.Duration(c.config.RequestTimeout)*time.Millisecond)
		c.sendEvent(timeoutEvent{})
	})
}

func (c *core) checkFullnodeSignature(data []byte, sig []byte) (common.Address, error) {
	return sport.CheckFullnodeSignature(c.fullnodeSet, data, sig)
}
