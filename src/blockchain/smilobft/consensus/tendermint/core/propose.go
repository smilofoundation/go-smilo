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

package core

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"go-smilo/src/blockchain/smilobft/consensus"
	"go-smilo/src/blockchain/smilobft/core/types"
)

func (c *core) sendProposal(ctx context.Context, p *types.Block) {
	logger := c.logger.New("step", c.currentRoundState.Step())

	// If I'm the proposer and I have the same height with the proposal
	if c.currentRoundState.Height().Int64() == p.Number().Int64() && c.isProposer() && !c.sentProposal {
		logger.Debug("I'm the proposer and I have the same height with the proposal", "c.currentRoundState.Height().Int64()", c.currentRoundState.Height().Int64(),
			"p.Number().Int64()", p.Number().Int64(), "c.sentProposal", c.sentProposal)

		proposalBlock := NewProposal(c.currentRoundState.Round(), c.currentRoundState.Height(), c.validRound, p, c.logger)
		proposal, err := Encode(proposalBlock)
		if err != nil {
			logger.Error("sendProposal, Failed to encode", "Round", proposalBlock.Round, "Height", proposalBlock.Height, "ValidRound", c.validRound)
			return
		}

		if proposalBlock == nil {
			logger.Error("send nil proposed block",
				"Round", c.currentRoundState.round.String(), "Height",
				c.currentRoundState.height.String(), "ValidRound", c.validRound)

			return
		}

		c.sentProposal = true
		c.backend.SetProposedBlockHash(p.Hash())

		msg := &Message{
			Code:          msgProposal,
			Msg:           proposal,
			Address:       c.address,
			CommittedSeal: []byte{},
		}

		c.logProposalMessageEvent("MessageEvent(Proposal): Sent", *proposalBlock, c.address.String(), "broadcast")

		logger.Debug("I'm the proposer and broadcast", "msg", msg)

		c.broadcast(ctx, msg)
	}
}

func (c *core) handleProposal(ctx context.Context, msg *Message) error {
	var proposal Proposal
	err := msg.Decode(&proposal)
	if err != nil {
		return errFailedDecodeProposal
	}

	// Ensure we have the same view with the Proposal message
	if err := c.checkMessage(proposal.Round, proposal.Height, propose); err != nil {
		// We don't care about old proposals so they are ignored
		c.logger.Warn("We don't care about old proposals so they are ignored", "msg", msg)
		return err
	}

	// Check if the message comes from currentRoundState proposer
	if !c.valSet.IsProposer(msg.Address) {
		c.logger.Warn("Ignore proposal messages from non-proposer", "msg", msg)
		return errNotFromProposer
	}

	// Verify the proposal we received
	if duration, err := c.backend.VerifyProposal(*proposal.ProposalBlock); err != nil {
		c.logger.Warn("Verify the proposal we received", "msg", msg, "duration", duration, "proposal.ProposalBlock", proposal.ProposalBlock)

		if timeoutErr := c.proposeTimeout.stopTimer(); timeoutErr != nil {
			c.logger.Warn("Verify the proposal timeoutErr", "msg", msg, "duration", duration,
				"proposal.ProposalBlock", proposal.ProposalBlock, "timeoutErr", timeoutErr)
			return timeoutErr
		}

		c.logger.Debug("Stopped Scheduled Proposal Timeout, sendPrevote nil", "duration", duration, "proposal.ProposalBlock", proposal.ProposalBlock)
		c.sendPrevote(ctx, true)
		// do not to accept another proposal in current round
		c.setStep(prevote)

		c.logger.Warn("Failed to verify proposal", "err", err, "duration", duration)
		// if it's a future block, we will handle it again after the duration
		// TIME FIELD OF HEADER CHECKED HERE - NOT HEIGHT
		// TODO: implement wiggle time / median time
		if err == consensus.ErrFutureBlock {
			c.logger.Warn("if it's a future block, we will handle it again after the duration", "err", err, "duration", duration)

			c.stopFutureProposalTimer()
			c.futureProposalTimer = time.AfterFunc(duration, func() {
				_, sender := c.valSet.GetByAddress(msg.Address)
				toSend := backlogEvent{
					src: sender,
					msg: msg,
				}

				c.logger.Warn("if it's a future block, time.AfterFunc, ", "err", err, "duration", duration, "toSend", toSend)

				c.sendEvent(toSend)
			})
		}
		return err
	}

	// Here is about to accept the Proposal
	if c.currentRoundState.Step() == propose {
		c.logger.Warn("Here is about to accept the Proposal ", "c.currentRoundState.Step()", c.currentRoundState.Step())

		if err := c.proposeTimeout.stopTimer(); err != nil {
			c.logger.Error("propose, Here is about to accept the Proposal, proposeTimeout err ", "c.currentRoundState.Step()", c.currentRoundState.Step(), "err", err)
			return err
		}
		c.logger.Debug("propose, Stopped Scheduled Proposal Timeout")

		// Set the proposal for the current round
		c.currentRoundState.SetProposal(&proposal, msg)

		c.logProposalMessageEvent("propose, MessageEvent(Proposal): Received", proposal, msg.Address.String(), c.address.String())

		vr := proposal.ValidRound.Int64()
		h := proposal.ProposalBlock.Hash()
		curR := c.currentRoundState.Round().Int64()

		c.currentHeightOldRoundsStatesMu.RLock()
		defer c.currentHeightOldRoundsStatesMu.RUnlock()

		// Line 22 in Algorithm 1 of The latest gossip on BFT consensus
		if vr == -1 {
			var voteForProposal = false
			if c.lockedValue != nil {
				voteForProposal = c.lockedRound.Int64() == -1 || h == c.lockedValue.Hash()

			}
			c.logger.Debug("prevote, Line 22 in Algorithm 1 of The latest gossip on BFT consensus", "voteForProposal", voteForProposal)
			c.sendPrevote(ctx, voteForProposal)
			c.setStep(prevote)
			return nil
		}

		rs, ok := c.currentHeightOldRoundsStates[vr]
		if !ok {
			c.logger.Error("handleProposal. unknown old round",
				"proposalHeight", h,
				"proposalRound", vr,
				"currentHeight", c.currentRoundState.height.Uint64(),
				"currentRound", c.currentRoundState.round,
			)
		}

		// Line 28 in Algorithm 1 of The latest gossip on BFT consensus
		if ok && vr < curR && c.Quorum(rs.Prevotes.VotesSize(h)) {
			var voteForProposal = false
			if c.lockedValue != nil {
				voteForProposal = c.lockedRound.Int64() <= vr || h == c.lockedValue.Hash()
			}

			c.logger.Debug("prevote, Line 28 in Algorithm 1 of The latest gossip on BFT consensus", "ok", ok, "vr", vr, "curR", curR, "rs.Prevotes.VotesSize(h)", rs.Prevotes.VotesSize(h),
				"voteForProposal", voteForProposal)

			c.sendPrevote(ctx, voteForProposal)
			c.setStep(prevote)
		}
	}

	return nil
}

func (c *core) logProposalMessageEvent(message string, proposal Proposal, from, to string) {
	c.logger.Debug(message,
		"type", "Proposal",
		"from", from,
		"to", to,
		"currentHeight", c.currentRoundState.Height(),
		"msgHeight", proposal.Height,
		"currentRound", c.currentRoundState.Round(),
		"msgRound", proposal.Round,
		"currentStep", c.currentRoundState.Step(),
		"isProposer", c.isProposer(),
		"currentProposer", c.valSet.GetProposer(),
		"isNilMsg", proposal.ProposalBlock.Hash() == common.Hash{},
		"hash", proposal.ProposalBlock.Hash(),
	)
}
