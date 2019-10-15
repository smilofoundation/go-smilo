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
	"crypto/ecdsa"
	"go-smilo/src/blockchain/smilobft/cmn"
	"go-smilo/src/blockchain/smilobft/core/vm"
	"go-smilo/src/blockchain/smilobft/params"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	lru "github.com/hashicorp/golang-lru"
	"go-smilo/src/blockchain/smilobft/consensus"
	"go-smilo/src/blockchain/smilobft/consensus/istanbul"
	istanbulCore "go-smilo/src/blockchain/smilobft/consensus/istanbul/core"
	"go-smilo/src/blockchain/smilobft/consensus/istanbul/validator"
	"go-smilo/src/blockchain/smilobft/core"
	"go-smilo/src/blockchain/smilobft/core/types"
	"go-smilo/src/blockchain/smilobft/ethdb"
)

const (
	// fetcherID is the ID indicates the block is from Istanbul engine
	fetcherID = "istanbul"
)

// New creates an Ethereum Backend for Istanbul core engine.
func New(config *istanbul.Config, privateKey *ecdsa.PrivateKey, db ethdb.Database, chainConfig *params.ChainConfig, vmConfig *vm.Config) *backend {
	if chainConfig.Istanbul.Epoch != 0 {
		config.Epoch = chainConfig.Istanbul.Epoch
	}

	if chainConfig.Istanbul.RequestTimeout != 0 {
		config.RequestTimeout = chainConfig.Istanbul.RequestTimeout
	}
	if chainConfig.Istanbul.BlockPeriod != 0 {
		config.BlockPeriod = chainConfig.Istanbul.BlockPeriod
	}
	config.SetProposerPolicy(istanbul.ProposerPolicy(chainConfig.Istanbul.ProposerPolicy))

	recents, _ := lru.NewARC(inmemorySnapshots)
	recentMessages, _ := lru.NewARC(inmemoryPeers)
	knownMessages, _ := lru.NewARC(inmemoryMessages)
	backend := &backend{
		config:           config,
		istanbulEventMux: new(cmn.TypeMux),
		privateKey:       privateKey,
		address:          crypto.PubkeyToAddress(privateKey.PublicKey),
		logger:           log.New(),
		db:               db,
		commitCh:         make(chan *types.Block, 1),
		recents:          recents,
		candidates:       make(map[common.Address]bool),
		coreStarted:      false,
		recentMessages:   recentMessages,
		knownMessages:    knownMessages,
		vmConfig:         vmConfig,
	}
	backend.core = istanbulCore.New(backend, backend.config)
	return backend
}

// ----------------------------------------------------------------------------

type backend struct {
	config           *istanbul.Config
	istanbulEventMux *cmn.TypeMux
	privateKey       *ecdsa.PrivateKey
	address          common.Address
	core             istanbulCore.Engine
	logger           log.Logger
	db               ethdb.Database
	blockchain       *core.BlockChain
	chain            consensus.ChainReader
	currentBlock     func() *types.Block
	hasBadBlock      func(hash common.Hash) bool

	// the channels for istanbul engine notifications
	commitCh          chan *types.Block
	proposedBlockHash common.Hash
	sealMu            sync.Mutex
	coreStarted       bool
	coreMu            sync.RWMutex

	// Current list of candidates we are pushing
	candidates map[common.Address]bool
	// Protects the signer fields
	candidatesLock sync.RWMutex
	// Snapshots for recent block to speed up reorgs
	recents *lru.ARCCache

	// event subscription for ChainHeadEvent event
	broadcaster consensus.Broadcaster

	recentMessages *lru.ARCCache // the cache of peer's messages
	knownMessages  *lru.ARCCache // the cache of self messages

	autonityContractAddress common.Address // Ethereum address of the autonity contract
	vmConfig                *vm.Config
}


// Address implements istanbul.Backend.Address
func (sb *backend) Address() common.Address {
	return sb.address
}

func (sb *backend) Validators(number uint64) istanbul.ValidatorSet {
	validators, err := sb.retrieveSavedValidators(number, sb.blockchain)
	proposerPolicy := sb.config.GetProposerPolicy()
	if err != nil {
		return validator.NewSet(nil, proposerPolicy)
	}
	return validator.NewSet(validators, proposerPolicy)
}

// Broadcast implements istanbul.Backend.Broadcast
func (sb *backend) Broadcast(valSet istanbul.ValidatorSet, payload []byte) error {
	// send to others
	_ = sb.Gossip(valSet, payload)
	// send to self
	msg := istanbul.MessageEvent{
		Payload: payload,
	}
	go func() {
		_ = sb.istanbulEventMux.Post(msg)
	}()
	return nil
}

// Broadcast implements istanbul.Backend.Gossip
func (sb *backend) Gossip(valSet istanbul.ValidatorSet, payload []byte) error {
	hash := types.RLPHash(payload)
	sb.knownMessages.Add(hash, true)

	targets := make(map[common.Address]struct{})
	for _, val := range valSet.List() {
		if val.Address() != sb.Address() {
			targets[val.Address()] = struct{}{}
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

			go func() {
				err := p.Send(istanbulMsg, payload)
				if err != nil {
					log.Error("Gossip, istanbulMsg message, FAIL!!!", "payload hash", hash.Hex(), "peer", p.String(), "err", err)
				} else {
					//log.Debug("Gossip, istanbulMsg message, OK!!!", "payload hash", hash.Hex(), "peer", p.String())
				}
			}()
		}
	}
	return nil
}

// Commit implements istanbul.Backend.Commit
func (sb *backend) Commit(proposal istanbul.Proposal, seals [][]byte) error {
	// Check if the proposal is a valid block
	block := &types.Block{}
	block, ok := proposal.(*types.Block)
	if !ok {
		sb.logger.Error("Invalid proposal, %v", proposal)
		return errInvalidProposal
	}

	h := block.Header()
	// Append seals into extra-data
	err := types.WriteCommittedSeals(h, seals)
	if err != nil {
		return err
	}
	// update block's header
	block = block.WithSeal(h)

	sb.logger.Info("Committed", "address", sb.Address(), "hash", proposal.Hash(), "number", proposal.Number().Uint64())
	// - if the proposed and committed blocks are the same, send the proposed hash
	//   to commit channel, which is being watched inside the engine.Seal() function.
	// - otherwise, we try to insert the block.
	// -- if success, the ChainHeadEvent event will be broadcasted, try to build
	//    the next block and the previous Seal() will be stopped.
	// -- otherwise, a error will be returned and a round change event will be fired.
	if sb.proposedBlockHash == block.Hash() {
		// feed block hash to Seal() and wait the Seal() result
		sb.logger.Debug("SUCCESS to compare proposedBlockHash with actual sealed block hash", "proposedBlockHash", sb.proposedBlockHash, "block.Hash", block.Hash())
		sb.commitCh <- block
		return nil
	}

	if sb.broadcaster != nil {
		sb.logger.Debug("broadcaster Enqueue fetcherID block")
		sb.broadcaster.Enqueue(fetcherID, block)
	} else {
		sb.logger.Debug("Failed broadcast Enqueue fetcherID block, wtf ? ", "proposedBlockHash", sb.proposedBlockHash, "block.Hash", block.Hash())
	}
	return nil
}

// EventMux implements istanbul.Backend.EventMux
func (sb *backend) EventMux() *cmn.TypeMux {
	return sb.istanbulEventMux
}

// Verify implements istanbul.Backend.Verify
func (sb *backend) Verify(proposal istanbul.Proposal) (time.Duration, error) {
	// Check if the proposal is a valid block
	block := &types.Block{}
	block, ok := proposal.(*types.Block)
	if !ok {
		sb.logger.Error("Invalid proposal, %v", proposal)
		return 0, errInvalidProposal
	}

	// check bad block
	if sb.HasBadProposal(block.Hash()) {
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

			state,vaultstate, _ := sb.blockchain.State()
			state = state.Copy() // copy the state, we don't want to save modifications
			gp := new(core.GasPool).AddGas(block.GasLimit())
			usedGas := new(uint64)
			// blockchain.Processor().Process() would have been a better choice but it calls back Finalize()
			for i, tx := range block.Transactions() {
				state.Prepare(tx.Hash(), block.Hash(), i)
				// Might be vulnerable to DoS Attack depending on gaslimit
				// Todo : Double check
				_, _,_, err := core.ApplyTransaction(sb.blockchain.Config(), sb.blockchain, nil,
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
		istanbulExtra, _ := types.ExtractBFTHeaderExtra(header)

		//Perform the actual comparison
		if len(istanbulExtra.Validators) != len(validators) {
			return 0, errInconsistentValidatorSet
		}

		for i := range validators {
			if istanbulExtra.Validators[i] != validators[i] {
				return 0, errInconsistentValidatorSet
			}
		}
		// At this stage extradata field is consistent with the validator list returned by Soma-contract

		return 0, nil
	} else if err == consensus.ErrFutureBlock {
		return time.Unix(int64(block.Header().Time), 0).Sub(now()), consensus.ErrFutureBlock
	}
	return 0, err
}

// Sign implements istanbul.Backend.Sign
func (sb *backend) Sign(data []byte) ([]byte, error) {
	hashData := crypto.Keccak256(data)
	return crypto.Sign(hashData, sb.privateKey)
}

// CheckSignature implements istanbul.Backend.CheckSignature
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

// HasPropsal implements istanbul.Backend.HashBlock
func (sb *backend) HasPropsal(hash common.Hash, number *big.Int) bool {
	return sb.blockchain.GetHeader(hash, number.Uint64()) != nil
}

// GetProposer implements istanbul.Backend.GetProposer
func (sb *backend) GetProposer(number uint64) common.Address {
	if h := sb.blockchain.GetHeaderByNumber(number); h != nil {
		a, _ := sb.Author(h)
		return a
	}
	return common.Address{}
}

func (sb *backend) LastProposal() (istanbul.Proposal, common.Address) {
	block := sb.currentBlock()

	var proposer common.Address
	if block.Number().Cmp(common.Big0) > 0 {
		var err error
		proposer, err = sb.Author(block.Header())
		if err != nil {
			sb.logger.Error("Failed to get block proposer", "err", err)
			return nil, common.Address{}
		}
	}

	// Return header only block here since we don't need block body
	return block, proposer
}

func (sb *backend) HasBadProposal(hash common.Hash) bool {
	if sb.hasBadBlock == nil {
		return false
	}
	return sb.hasBadBlock(hash)
}

func (sb *backend) Close() error {
	return nil
}

// Whitelist for the current block
func (sb *backend) WhiteList() []string {
	state,vaultstate, err := sb.blockchain.State()
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

