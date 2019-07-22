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

package backend

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	lru "github.com/hashicorp/golang-lru"

	"go-smilo/src/blockchain/smilobft/consensus"
	"go-smilo/src/blockchain/smilobft/consensus/sport"
	"go-smilo/src/blockchain/smilobft/consensus/sport/fullnode"
	"go-smilo/src/blockchain/smilobft/core"
	"go-smilo/src/blockchain/smilobft/core/types"
)

// Fullnodes implements sport.Backend.Fullnodes
func (sb *backend) Fullnodes(proposal sport.BlockProposal) sport.FullnodeSet {
	return sb.getFullnodes(proposal.Number().Uint64(), proposal.Hash())
}

// Broadcast implements sport.Backend.Broadcast
func (sb *backend) Broadcast(fullnodeSet sport.FullnodeSet, payload []byte) error {
	// send to others
	sb.Gossip(fullnodeSet, payload)
	// send to self
	msg := sport.MessageEvent{
		Payload: payload,
	}
	go sb.smilobftEventMux.Post(msg)
	return nil
}

// Gossip implements sport.Backend.Gossip
func (sb *backend) Gossip(fullnodeSet sport.FullnodeSet, payload []byte) error {
	hash := sport.RLPHash(payload)
	sb.knownMessages.Add(hash, true)

	targets := make(map[common.Address]bool)
	for _, val := range fullnodeSet.List() {
		if val.Address() != sb.Address() {
			targets[val.Address()] = true
		}
	}

	if sb.broadcaster != nil && len(targets) > 0 {
		ps := sb.broadcaster.FindPeers(targets)
		if len(ps) == 0 {
			log.Warn("Gossip FindPeers returned zero peers ....")
		}
		for addr, p := range ps {
			ms, ok := sb.recentMessages.Get(addr)
			var m *lru.ARCCache
			if ok {
				m, _ = ms.(*lru.ARCCache)
				if _, k := m.Get(hash); k {
					// This peer had this event, skip it
					continue
				}
			} else {
				m, _ = lru.NewARC(inmemoryMessages)
			}

			m.Add(hash, true)
			sb.recentMessages.Add(addr, m)

				err := p.Send(smilobftMsg, payload)
				if err != nil {
					log.Error("Gossip, smilobftMsg message, FAIL!!!", "payload hash", hash.Hex(), "peer", p.String(), "err", err)
				} else {
					//log.Debug("Gossip, smilobftMsg message, OK!!!", "payload hash", hash.Hex(), "peer", p.String())
				}

		}
	}
	return nil
}

// Commit implements sport.Backend.Commit
func (sb *backend) Commit(proposal sport.BlockProposal, seals [][]byte) error {
	// Check if the proposal is a valid block
	block := &types.Block{}
	block, ok := proposal.(*types.Block)
	if !ok {
		sb.logger.Error("Invalid proposal, %v", proposal)
		return errInvalidBlockProposal
	}

	h := block.Header()
	// Append seals into extra-data
	err := writeCommittedSeals(h, seals)
	if err != nil {
		return err
	}
	// update block's header
	block = block.WithSeal(h)

	sb.logger.Info("Commit, Committed", "address", sb.Address(), "hash", proposal.Hash(), "number", proposal.Number().Uint64())
	// - if the proposed and committed blocks are the same, send the proposed hash
	//   to commit channel, which is being watched inside the engine.Seal() function.
	// - otherwise, we try to insert the block.
	// -- if success, the ChainHeadEvent event will be broadcasted, try to build
	//    the next block and the previous Seal() will be stopped.
	// -- otherwise, a error will be returned and a round change event will be fired.
	if sb.proposedBlockHash == block.Hash() {
		sb.logger.Debug("feed block hash to Seal() and wait the Seal() result")
		sb.commitChBlock <- block
		return nil
	} else {
		sb.logger.Debug("Failed to compare proposedBlockHash with actual sealed block hash", "proposedBlockHash", sb.proposedBlockHash, "block.Hash", block.Hash())
	}

	if sb.broadcaster != nil {
		sb.logger.Debug("broadcaster Enqueue fetcherID block")
		sb.broadcaster.Enqueue(fetcherID, block)
	} else {
		sb.logger.Debug("Failed broadcast Enqueue fetcherID block, wtf ? ", "proposedBlockHash", sb.proposedBlockHash, "block.Hash", block.Hash())
	}
	return nil
}

// EventMux implements sport.Backend.EventMux
func (sb *backend) EventMux() *event.TypeMux {
	return sb.smilobftEventMux
}

// Verify implements sport.Backend.Verify
func (sb *backend) Verify(proposal sport.BlockProposal) (time.Duration, error) {
	// Check if the proposal is a valid block
	block := &types.Block{}
	block, ok := proposal.(*types.Block)
	if !ok {
		sb.logger.Error("Invalid proposal, %v", proposal)
		return 0, errInvalidBlockProposal
	}

	// check bad block
	if sb.HasBadBlockProposal(block.Hash()) {
		sb.logger.Error("Invalid proposal, core.ErrBlacklistedHash %v", proposal)
		return 0, core.ErrBlacklistedHash
	}

	// check block body
	txnHash := types.DeriveSha(block.Transactions())
	uncleHash := types.CalcUncleHash(block.Uncles())
	if txnHash != block.Header().TxHash {
		sb.logger.Error("Invalid proposal, errMismatchTxhashes %v", proposal)
		return 0, errMismatchTxhashes
	}
	if uncleHash != nilUncleHash {
		sb.logger.Error("Invalid proposal, errInvalidUncleHash %v", proposal)
		return 0, errInvalidUncleHash
	}

	// verify the header of proposed block
	err := sb.VerifyHeader(sb.chain, block.Header(), false)
	// ignore errEmptyCommittedSeals error because we don't have the committed seals yet
	if err == nil || err == errEmptyCommittedSeals {
		return 0, nil
	} else if err == consensus.ErrFutureBlock {
		sb.logger.Error("Invalid proposal, consensus.ErrFutureBlock %v", proposal)
		return time.Unix(block.Header().Time.Int64(), 0).Sub(now()), consensus.ErrFutureBlock
	}
	return 0, err
}

// Sign implements sport.Backend.Sign
func (sb *backend) Sign(data []byte) ([]byte, error) {
	hashData := crypto.Keccak256(data)
	return crypto.Sign(hashData, sb.privateKey)
}

// CheckSignature implements sport.Backend.CheckSignature
func (sb *backend) CheckSignature(data []byte, address common.Address, sig []byte) error {
	signer, err := sport.GetSignatureAddress(data, sig)
	if err != nil {
		log.Error("CheckSignature, Failed to GetSignatureAddress, ", "err", err)
		return err
	}
	// Compare derived addresses
	if signer != address {
		return errInvalidSignature
	}
	return nil
}

// HasBlockProposal implements sport.Backend.HashBlock
func (sb *backend) HasBlockProposal(hash common.Hash, number *big.Int) bool {
	return sb.chain.GetHeader(hash, number.Uint64()) != nil
}

// GetSpeaker implements sport.Backend.GetSpeaker
func (sb *backend) GetSpeaker(number uint64) common.Address {
	if h := sb.chain.GetHeaderByNumber(number); h != nil {
		a, _ := sb.Author(h)
		return a
	}
	return common.Address{}
}

// ParentFullnodes implements sport.Backend.GetParentFullnodes
func (sb *backend) ParentFullnodes(proposal sport.BlockProposal) sport.FullnodeSet {
	if block, ok := proposal.(*types.Block); ok {
		return sb.getFullnodes(block.Number().Uint64()-1, block.ParentHash())
	}
	return fullnode.NewFullnodeSet(nil, sb.config.SpeakerPolicy)
}

func (sb *backend) getFullnodes(number uint64, hash common.Hash) sport.FullnodeSet {
	snap, err := sb.snapshot(sb.chain, number, hash, nil)
	if err != nil {
		sb.logger.Error("Failed to getFullnodes from snapshot", "err", err)
		return fullnode.NewFullnodeSet(nil, sb.config.SpeakerPolicy)
	}
	return snap.FullnodeSet
}

// LastBlockProposal returns the last block header and speaker
func (sb *backend) LastBlockProposal() (sport.BlockProposal, common.Address) {
	block := sb.currentBlock()

	var speaker common.Address
	if block.Number().Cmp(common.Big0) > 0 {
		var err error
		speaker, err = sb.Author(block.Header())
		if err != nil {
			sb.logger.Error("Failed to get block speaker", "err", err)
			return nil, common.Address{}
		}
	}

	// Return header only block here since we don't need block body
	return block, speaker
}

// HasBadBlockProposal check if the hash has a bad block associated to it
func (sb *backend) HasBadBlockProposal(hash common.Hash) bool {
	if sb.hasBadBlock == nil {
		return false
	}
	return sb.hasBadBlock(hash)
}
