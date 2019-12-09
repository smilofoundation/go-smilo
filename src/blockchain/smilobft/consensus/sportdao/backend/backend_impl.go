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
	"go-smilo/src/blockchain/smilobft/cmn"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	lru "github.com/hashicorp/golang-lru"

	"go-smilo/src/blockchain/smilobft/consensus"
	"go-smilo/src/blockchain/smilobft/consensus/sportdao"
	"go-smilo/src/blockchain/smilobft/consensus/sportdao/fullnode"
	"go-smilo/src/blockchain/smilobft/core"
	"go-smilo/src/blockchain/smilobft/core/types"
)

// Fullnodes implements sportdao.Backend.Fullnodes
func (sb *backend) Fullnodes(number uint64) sportdao.FullnodeSet {
	validators, err := sb.retrieveSavedValidators(number, sb.chain)
	proposerPolicy := sb.config.GetProposerPolicy()
	if err != nil {
		return fullnode.NewSet(nil, proposerPolicy)
	}
	return fullnode.NewSet(validators, proposerPolicy)
}

// Broadcast implements sportdao.Backend.Broadcast
func (sb *backend) Broadcast(fullnodeSet sportdao.FullnodeSet, payload []byte) error {
	// send to others
	err := sb.Gossip(fullnodeSet, payload)
	if err != nil {
		sb.logger.Error("Failed to boadcast Gossip", "err", err)
	}
	// send to self
	msg := sportdao.MessageEvent{
		Payload: payload,
	}
	go func() {
		err = sb.smilobftEventMux.Post(msg)
		if err != nil {
			sb.logger.Error("Failed to boadcast smilobftEventMux.Post(msg)", "err", err)
		} else {
			log.Debug("Broadcast, smilobftMsg message, OK!!!")
		}
	}()
	return nil
}

// Gossip implements sportdao.Backend.Gossip
func (sb *backend) Gossip(fullnodeSet sportdao.FullnodeSet, payload []byte) error {
	hash := types.RLPHash(payload)
	sb.knownMessages.Add(hash, true)

	targets := make(map[common.Address]struct{})
	for _, val := range fullnodeSet.List() {
		if val.Address() != sb.Address() {
			targets[val.Address()] = struct{}{}
		}
	}

	if sb.broadcaster != nil && len(targets) > 0 {
		ps := sb.broadcaster.FindPeers(targets)
		if len(ps) == 0 {
			log.Warn("backend/backend_impl.go, Gossip() FindPeers returned zero peers ....")
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

			go func() {
				err := p.Send(smilobftMsg, payload)

				if err != nil {
					log.Error("Gossip, smilobftMsg message, FAIL!!!", "payload hash", hash.Hex(), "peer", p.String(), "err", err)
				} else {
					log.Debug("eth/backend_impl.go, Gossip(), smilobftMsg, Send message OK!!!", "payload hash", hash.Hex(), "peer", p.String())
				}
			}()

		}
	}
	return nil
}

// Commit implements sportdao.Backend.Commit
func (sb *backend) Commit(proposal sportdao.BlockProposal, seals [][]byte) error {
	// Check if the proposal is a valid block
	block := &types.Block{}
	block, ok := proposal.(*types.Block)
	if !ok {
		sb.logger.Error("Invalid proposal, %v", proposal)
		return errInvalidBlockProposal
	}

	h := block.Header()
	// Append seals into extra-data
	err := types.WriteCommittedSeals(h, seals)
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
		sb.logger.Debug("SUCCESS to compare proposedBlockHash with actual sealed block hash", "proposedBlockHash", sb.proposedBlockHash, "block.Hash", block.Hash())
		sb.commitChBlock <- block
		return nil
	} else {
		sb.logger.Error("ERROR WHEN comparing proposedBlockHash with actual sealed block hash", "proposedBlockHash", sb.proposedBlockHash, "block.Hash", block.Hash())
	}

	if sb.broadcaster != nil {
		sb.logger.Debug("broadcaster Enqueue fetcherID block")
		sb.broadcaster.Enqueue(fetcherID, block)
	} else {
		sb.logger.Debug("Failed broadcast Enqueue fetcherID block, wtf ? ", "proposedBlockHash", sb.proposedBlockHash, "block.Hash", block.Hash())
	}
	return nil
}

// EventMux implements sportdao.Backend.EventMux
func (sb *backend) EventMux() *cmn.TypeMux {
	return sb.smilobftEventMux
}

// Verify implements sportdao.Backend.Verify
func (sb *backend) Verify(proposal sportdao.BlockProposal) (time.Duration, error) {
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
	err := sb.VerifyHeader(sb.blockchain, block.Header(), false)
	// ignore errEmptyCommittedSeals error because we don't have the committed seals yet
	if err == nil || err == types.ErrEmptyCommittedSeals {
		// the current blockchain state is synchronised with Istanbul's state
		// and we know that the proposed block was mined by a valid validator
		header := block.Header()
		//We need at this point to process all the transactions in the block
		//in order to extract the list of the next validators and validate the extradata field
		var validators []common.Address
		var err error
		if header.Number.Uint64() > 1 {

			state, vaultstate, _ := sb.blockchain.State()
			state = state.Copy() // copy the state, we don't want to save modifications
			gp := new(core.GasPool).AddGas(block.GasLimit())
			usedGas := new(uint64)
			// blockchain.Processor().Process() would have been a better choice but it calls back Finalize()
			for i, tx := range block.Transactions() {
				state.Prepare(tx.Hash(), block.Hash(), i)
				// Might be vulnerable to DoS Attack depending on gaslimit
				// Todo : Double check
				_, _, _, err := core.ApplyTransaction(sb.blockchain.Config(), sb.blockchain, nil,
					gp, state, vaultstate, header, tx, usedGas, *sb.vmConfig)

				if err != nil {
					return 0, err
				}
			}

			validators, err = sb.blockchain.GetAutonityContract().ContractGetValidators(sb.blockchain, header, state)
			if err != nil {
				return 0, err
			}
		} else {
			validators, err = sb.retrieveSavedValidators(1, sb.blockchain) //genesis block and block #1 have the same validators
		}
		istanbulExtra, _ := types.ExtractSportExtra(header)

		//Perform the actual comparison
		if len(istanbulExtra.Fullnodes) != len(validators) {
			return 0, errInconsistentValidatorSet
		}

		for i := range validators {
			if istanbulExtra.Fullnodes[i] != validators[i] {
				return 0, errInconsistentValidatorSet
			}
		}
		// At this stage extradata field is consistent with the validator list returned by Soma-contract

		return 0, nil
	} else if err == consensus.ErrFutureBlock {
		sb.logger.Error("Invalid proposal, consensus.ErrFutureBlock ", "proposal", proposal, "err", err)
		return time.Unix(int64(block.Header().Time), 0).Sub(now()), consensus.ErrFutureBlock
	} else {
		sb.logger.Error("Invalid proposal", "proposal", proposal, "err", err)
	}
	return 0, err
}

// Sign implements sportdao.Backend.Sign
func (sb *backend) Sign(data []byte) ([]byte, error) {
	hashData := crypto.Keccak256(data)
	return crypto.Sign(hashData, sb.privateKey)
}

// CheckSignature implements sportdao.Backend.CheckSignature
func (sb *backend) CheckSignature(data []byte, address common.Address, sig []byte) error {
	signer, err := types.GetSignatureAddress(data, sig)
	if err != nil {
		log.Error("CheckSignature, Failed to GetSignatureAddress, ", "err", err)
		return err
	}
	// Compare derived addresses
	if signer != address {
		return types.ErrInvalidSignature
	}
	return nil
}

// HasBlockProposal implements sportdao.Backend.HashBlock
func (sb *backend) HasBlockProposal(hash common.Hash, number *big.Int) bool {
	return sb.blockchain.GetHeader(hash, number.Uint64()) != nil
}

// GetSpeaker implements sportdao.Backend.GetSpeaker
func (sb *backend) GetSpeaker(number uint64) common.Address {
	if h := sb.blockchain.GetHeaderByNumber(number); h != nil {
		a, _ := sb.Author(h)
		return a
	}
	return common.Address{}
}

// ParentFullnodes implements sportdao.Backend.GetParentFullnodes
func (sb *backend) ParentFullnodes(proposal sportdao.BlockProposal) sportdao.FullnodeSet {
	if block, ok := proposal.(*types.Block); ok {
		return sb.getFullnodes(block.Number().Uint64()-1, block.ParentHash())
	}
	return fullnode.NewFullnodeSet(nil, sb.config.SpeakerPolicy)
}

func (sb *backend) getFullnodes(number uint64, hash common.Hash) sportdao.FullnodeSet {
	//snap, err := sb.snapshot(sb.chain, number, hash, nil)
	//if err != nil {
	//	sb.logger.Error("Failed to getFullnodes from snapshot", "err", err)
	//	return fullnode.NewFullnodeSet(nil, sb.config.SpeakerPolicy)
	//}
	//return snap.FullnodeSet
	return nil
}

// LastBlockProposal returns the last block header and speaker
func (sb *backend) LastBlockProposal() (sportdao.BlockProposal, common.Address) {
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

// Whitelist for the current block
func (sb *backend) WhiteList() []string {
	state, vaultstate, err := sb.blockchain.State()
	if err != nil {
		sb.logger.Error("Failed to get block white list", "err", err)
		return nil
	}

	enodes, err := sb.blockchain.GetAutonityContract().GetWhitelist(sb.blockchain.CurrentBlock(), state, vaultstate)
	if err != nil {
		sb.logger.Error("Failed to get block white list", "err", err)
		return nil
	}

	return enodes.StrList
}
