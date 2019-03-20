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

	"github.com/ethereum/go-ethereum/common"

	"go-smilo/src/blockchain/smilobft/consensus/sport"
)

// sendNextRoundChange sends the ROUND CHANGE message with current round + 1
func (c *core) sendNextRoundChange() {
	cv := c.currentView()
	c.sendRoundChange(new(big.Int).Add(cv.Round, common.Big1))
}

// sendRoundChange sends the ROUND CHANGE message with the given round
func (c *core) sendRoundChange(round *big.Int) {
	logger := c.logger.New("state", c.state)

	cv := c.currentView()
	logger.Debug("sendRoundChange, begin ... ", "current round", cv.Round, "target round", round)
	if cv.Round.Cmp(round) >= 0 {
		logger.Error("Cannot send out the round change", "current round", cv.Round, "target round", round)
		return
	}

	c.catchUpRound(&sport.View{
		// The round number we'd like to transfer to.
		Round:    new(big.Int).Set(round),
		Sequence: new(big.Int).Set(cv.Sequence),
	})

	// Now we have the new round number and sequence number
	cv = c.currentView()
	rc := &sport.Subject{
		View:   cv,
		Digest: common.Hash{},
	}

	payload, err := Encode(rc)
	if err != nil {
		logger.Error("Failed to encode ROUND CHANGE", "rc", rc, "err", err)
		return
	}

	c.broadcast(&message{
		Code: msgRoundChange,
		Msg:  payload,
	})
}

func (c *core) handleRoundChange(msg *message, src sport.Fullnode) error {
	logger := c.logger.New("state", c.state, "from", src.Address().Hex())

	// Decode ROUND CHANGE message
	var rc *sport.Subject
	if err := msg.Decode(&rc); err != nil {
		logger.Error("Failed to decode ROUND CHANGE", "err", err)
		return errInvalidMessage
	}

	if err := c.checkMessage(msgRoundChange, rc.View); err != nil {
		return err
	}

	cv := c.currentView()
	roundView := rc.View

	// Add the ROUND CHANGE message to its message set and return how many
	// messages we've got with the same round number and sequence number.
	messageCount, err := c.roundChangeSet.Add(roundView.Round, msg)
	if err != nil {
		logger.Error("Failed to add round change message",
			"from", src,
			"msg", msg,
			"err", err,
			"F", c.fullnodeSet.MaxFaulty(),
			"E", c.fullnodeSet.E(),
		)
		return err
	}

	// Once we received f+1 ROUND CHANGE messages, those messages form a weak certificate.
	// If our round number is smaller than the certificate's round number, we would
	// try to catch up the round number.
	//2F+E
	expectedConsensus := c.fullnodeSet.MaxFaulty() + c.fullnodeSet.E()
	isRoundNumberSmallerThenCertificate := int(messageCount) == expectedConsensus

	logger.Trace("handleRoundChange, Validating variables, ",
		"isRoundNumberSmallerThenCertificate", isRoundNumberSmallerThenCertificate,
		"expectedConsensus", expectedConsensus,
		"messageCount", messageCount,
		"waitingForRoundChange", c.waitingForRoundChange,
		"MaxFaulty", c.fullnodeSet.MaxFaulty(),
		"MinApprovers", c.fullnodeSet.MinApprovers(),
		"E", c.fullnodeSet.E(),
		"messageCount==MinApprovers", int(messageCount) == c.fullnodeSet.MinApprovers(),
	)

	if c.waitingForRoundChange && isRoundNumberSmallerThenCertificate {
		//cv < roundView
		if cv.Round.Cmp(roundView.Round) < 0 {
			logger.Debug("handleRoundChange, sendRoundChange, F+1 ROUND CHANGE, Will send Round Change with current Round,",
				"Round", roundView.Round,
				"diff_rounds", cv.Round.Cmp(roundView.Round),
				"Current Round", cv.Round,
				"Round Change:", roundView.Round,
				"isRoundNumberSmallerThenCertificate", isRoundNumberSmallerThenCertificate,
				"expectedConsensus", expectedConsensus,
				"messageCount", messageCount,
				"waitingForRoundChange", c.waitingForRoundChange,
				"MaxFaulty", c.fullnodeSet.MaxFaulty(),
				"MinApprovers", c.fullnodeSet.MinApprovers(),
				"E", c.fullnodeSet.E(),
				"messageCount==MinApprovers", int(messageCount) == c.fullnodeSet.MinApprovers(),
			)
			c.sendRoundChange(roundView.Round)
		} else {
			logger.Debug("handleRoundChange, F+1 ROUND CHANGE, waitingForRoundChange && isRoundNumberSmallerThenCertificate, but Still Waiting For Round Change, (same round number and sequence number)",
				"messageCount", messageCount,
				"Round", roundView.Round,
				"diff_rounds", cv.Round.Cmp(roundView.Round),
				"Current Round", cv.Round,
				"Round Change:", roundView.Round,
				"isRoundNumberSmallerThenCertificate", isRoundNumberSmallerThenCertificate,
				"expectedConsensus", expectedConsensus,
				"messageCount", messageCount,
				"waitingForRoundChange", c.waitingForRoundChange,
				"MaxFaulty", c.fullnodeSet.MaxFaulty(),
				"MinApprovers", c.fullnodeSet.MinApprovers(),
				"E", c.fullnodeSet.E(),
				"messageCount==MinApprovers", int(messageCount) == c.fullnodeSet.MinApprovers(),
			)
		}
		return nil
		//2F+E
	} else if int(messageCount) == c.fullnodeSet.MinApprovers() && (c.waitingForRoundChange || cv.Round.Cmp(roundView.Round) < 0) {
		// We've received 2F+E ROUND CHANGE messages, start a new round immediately. handlePrepare, before
		logger.Debug("handleRoundChange, startNewRound, We've received 2F+E ROUND CHANGE messages, start a new round immediately.",
			"Round", roundView.Round,
			"diff_rounds", cv.Round.Cmp(roundView.Round),
			"Current Round", cv.Round,
			"Round Change:", roundView.Round,
			"isRoundNumberSmallerThenCertificate", isRoundNumberSmallerThenCertificate,
			"expectedConsensus", expectedConsensus,
			"messageCount", messageCount,
			"waitingForRoundChange", c.waitingForRoundChange,
			"MaxFaulty", c.fullnodeSet.MaxFaulty(),
			"MinApprovers", c.fullnodeSet.MinApprovers(),
			"E", c.fullnodeSet.E(),
			"messageCount==MinApprovers", int(messageCount) == c.fullnodeSet.MinApprovers(),
		)
		c.startNewRound(roundView.Round)
		return nil
	} else if cv.Round.Cmp(roundView.Round) < 0 {
		// Only gossip the message with current round to other fullnodes.
		logger.Debug("handleRoundChange, Only gossip the message with current round to other fullnodes.",
			"Round", roundView.Round,
			"diff_rounds", cv.Round.Cmp(roundView.Round),
			"Current Round", cv.Round,
			"Round Change:", roundView.Round,
			"isRoundNumberSmallerThenCertificate", isRoundNumberSmallerThenCertificate,
			"expectedConsensus", expectedConsensus,
			"messageCount", messageCount,
			"waitingForRoundChange", c.waitingForRoundChange,
			"MaxFaulty", c.fullnodeSet.MaxFaulty(),
			"MinApprovers", c.fullnodeSet.MinApprovers(),
			"E", c.fullnodeSet.E(),
			"messageCount==MinApprovers", int(messageCount) == c.fullnodeSet.MinApprovers(),
		)
		return errIgnored
	}
	return nil
}

// ----------------------------------------------------------------------------

// Add adds the round and message into round change set
func (rcs *roundChangeSet) Add(r *big.Int, msg *message) (int, error) {
	rcs.mu.Lock()
	defer rcs.mu.Unlock()

	round := r.Uint64()
	if rcs.roundChanges[round] == nil {
		rcs.roundChanges[round] = newMessageSet(rcs.fullnodeSet)
	}
	err := rcs.roundChanges[round].Add(msg)
	if err != nil {
		return 0, err
	}
	return rcs.roundChanges[round].Size(), nil
}

// Clear deletes the messages with smaller round
func (rcs *roundChangeSet) Clear(round *big.Int) {
	rcs.mu.Lock()
	defer rcs.mu.Unlock()

	for k, rms := range rcs.roundChanges {
		if len(rms.Values()) == 0 || k < round.Uint64() {
			delete(rcs.roundChanges, k)
		}
	}
}

// MaxRound returns the max round which the number of messages is equal or larger than num
func (rcs *roundChangeSet) MaxRound(num int) *big.Int {
	rcs.mu.Lock()
	defer rcs.mu.Unlock()

	var maxRound *big.Int
	for k, rms := range rcs.roundChanges {
		if rms.Size() < num {
			continue
		}
		r := big.NewInt(int64(k))
		if maxRound == nil || maxRound.Cmp(r) < 0 {
			maxRound = r
		}
	}
	return maxRound
}
