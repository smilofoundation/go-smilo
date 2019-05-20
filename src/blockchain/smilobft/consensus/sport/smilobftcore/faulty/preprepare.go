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
	"time"

	"github.com/ethereum/go-ethereum/consensus"
	"go-smilo/src/blockchain/smilobft/consensus/sport"
)

func (c *core) sendPreprepare(request *sport.Request) {
	logger := c.logger.New("state", c.state)

	// If I'm the proposer and I have the same sequence with the proposal
	if c.current.Sequence().Cmp(request.BlockProposal.Number()) != 0 || c.isProposer() {
		if !c.alwaysPropose() {
			return
		}
		logger.Info("Always propose a BlockProposal", "request", request)
	}
	curView := c.currentView()
	preprepare, err := Encode(&sport.Preprepare{
		View:     curView,
		BlockProposal: request.BlockProposal,
	})
	if err != nil {
		logger.Error("Failed to encode", "view", curView)
		return
	}

	c.broadcast(&message{
		Code: msgPreprepare,
		Msg:  preprepare,
	})

	logger.Trace("sendPreprepare")
	c.broadcast(&message{
		Code: msgPreprepare,
		Msg:  preprepare,
	})
}

func (c *core) handlePreprepare(msg *message, src sport.Fullnode) error {
	logger := c.logger.New("from", src, "state", c.state)

	// Decode PRE-PREPARE
	var preprepare *sport.Preprepare
	err := msg.Decode(&preprepare)
	if err != nil {
		return errFailedDecodePreprepare
	}

	// Ensure we have the same view with the PRE-PREPARE message
	// If it is old message, see if we need to broadcast COMMIT
	if err := c.checkMessage(msgPreprepare, preprepare.View); err != nil {
		if err == errOldMessage {
			// Get Fullnode set for the given proposal
			fullnodeSet := c.backend.ParentFullnodes(preprepare.BlockProposal).Copy()
			previousProposer := c.backend.GetSpeaker(preprepare.BlockProposal.Number().Uint64() - 1)
			fullnodeSet.CalcSpeaker(previousProposer, preprepare.View.Round.Uint64())
			// Broadcast COMMIT if it is an existing block
			// 1. The proposer needs to be a proposer matches the given (Sequence + Round)
			// 2. The given block must exist
			if fullnodeSet.IsSpeaker(src.Address()) && c.backend.HasBlockProposal(preprepare.BlockProposal.Hash(), preprepare.BlockProposal.Number()) {
				c.sendCommitForOldBlock(preprepare.View, preprepare.BlockProposal.Hash())
				return nil
			}
		}
		return err
	}

	// Check if the message comes from current proposer
	if !c.fullnodeSet.IsSpeaker(src.Address()) {
		logger.Warn("Ignore preprepare messages from non-proposer")
		return errNotFromProposer
	}

	// Verify the proposal we received
	if duration, err := c.backend.Verify(preprepare.BlockProposal); err != nil {
		logger.Warn("Failed to verify proposal", "err", err, "duration", duration)
		// if it's a future block, we will handle it again after the duration
		if err == consensus.ErrFutureBlock {
			c.stopFuturePreprepareTimer()
			c.futurePreprepareTimer = time.AfterFunc(duration, func() {
				c.sendEvent(backlogEvent{
					src: src,
					msg: msg,
				})
			})
		} else {
			c.sendNextRoundChange()
		}
		return err
	}

	// Here is about to accept the PRE-PREPARE
	if c.state == StateAcceptRequest {
		// Send ROUND CHANGE if the locked proposal and the received proposal are different
		if c.current.IsHashLocked() {
			if preprepare.BlockProposal.Hash() == c.current.GetLockedHash() {
				// Broadcast COMMIT and enters Prepared state directly
				c.acceptPreprepare(preprepare)
				c.setState(StatePrepared)
				c.sendCommit()
			} else {
				// Send round change
				c.sendNextRoundChange()
			}
		} else {
			// Either
			//   1. the locked proposal and the received proposal match
			//   2. we have no locked proposal
			c.acceptPreprepare(preprepare)
			c.setState(StatePreprepared)
			c.sendPrepare()
		}
	}

	return nil
}

func (c *core) acceptPreprepare(preprepare *sport.Preprepare) {
	c.consensusTimestamp = time.Now()
	c.current.SetPreprepare(preprepare)
}
