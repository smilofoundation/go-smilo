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
	"bytes"
	"context"
	"errors"
	"fmt"
	"go-smilo/src/blockchain/smilobft/consensus/tendermint/bft"
	"math/big"
	"time"

	"go-smilo/src/blockchain/smilobft/consensus"
	tendermintCore "go-smilo/src/blockchain/smilobft/consensus/tendermint/core"
	"go-smilo/src/blockchain/smilobft/consensus/tendermint/events"
	"go-smilo/src/blockchain/smilobft/core"
	"go-smilo/src/blockchain/smilobft/core/state"
	"go-smilo/src/blockchain/smilobft/core/types"
	"go-smilo/src/blockchain/smilobft/rpc"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

const (
	inmemorySnapshots = 128 // Number of recent vote snapshots to keep in memory
	inmemoryPeers     = 40
	inmemoryMessages  = 1024
)

// ErrStartedEngine is returned if the engine is already started
var ErrStartedEngine = errors.New("started engine")

var (
	// errInvalidProposal is returned when a prposal is malformed.
	//errInvalidProposal = errors.New("invalid proposal")
	// errUnknownBlock is returned when the list of committee is requested for a block
	// that is not part of the local blockchain.
	errUnknownBlock = errors.New("unknown block")
	// errUnauthorized is returned if a header is signed by a non authorized entity.
	errUnauthorized = errors.New("unauthorized")
	// errInvalidCoindbase is returned if the signer is not the coinbase address,
	errInvalidCoinbase = errors.New("invalid coinbase")
	// errInvalidDifficulty is returned if the difficulty of a block is not 1
	errInvalidDifficulty = errors.New("invalid difficulty")
	// errInvalidMixDigest is returned if a block's mix digest is not BFT digest.
	errInvalidMixDigest = errors.New("invalid BFT mix digest")
	// errInvalidNonce is returned if a block's nonce is invalid
	errInvalidNonce = errors.New("invalid nonce")
	// errInvalidUncleHash is returned if a block contains an non-empty uncle list.
	errInvalidUncleHash = errors.New("non empty uncle hash")
	// errInvalidTimestamp is returned if the timestamp of a block is lower than the previous block's timestamp + the minimum block period.
	errInvalidTimestamp = errors.New("invalid timestamp")
	// errInvalidRound is returned if the round exceed maximum round number.
	errInvalidRound = errors.New("invalid round")
)
var (
	defaultDifficulty = big.NewInt(1)
	nilUncleHash      = types.CalcUncleHash(nil) // Always Keccak256(RLP([])) as uncles are meaningless outside of PoW.
	emptyNonce        = types.BlockNonce{}
	now               = time.Now

	nonceAuthVote = hexutil.MustDecode("0xffffffffffffffff") // Magic nonce number to vote on adding a new validator
	nonceDropVote = hexutil.MustDecode("0x0000000000000000") // Magic nonce number to vote on removing a validator.
)

// Author retrieves the Ethereum address of the account that minted the given
// block, which may be different from the header's coinbase if a consensus
// engine is based on signatures.
func (sb *Backend) Author(header *types.Header) (common.Address, error) {
	return types.Ecrecover(header)
}

// VerifyHeader checks whether a header conforms to the consensus rules of a
// given engine. Verifying the seal may be done optionally here, or explicitly
// via the VerifySeal method.
func (sb *Backend) VerifyHeader(chain consensus.ChainReader, header *types.Header, _ bool) error {
	return sb.verifyHeader(header, chain.GetHeaderByHash(header.ParentHash))
}

// verifyHeader checks whether a header conforms to the consensus rules. It
// expects the parent header to be provided unless header is the genesis
// header.
func (sb *Backend) verifyHeader(header, parent *types.Header) error {
	if header.Number == nil {
		return errUnknownBlock
	}
	if header.Round > tendermintCore.MaxRound {
		return errInvalidRound
	}
	// Don't waste time checking blocks from the future
	if big.NewInt(int64(header.Time)).Cmp(big.NewInt(now().Unix())) > 0 {
		return consensus.ErrFutureBlock
	}

	// Ensure that the coinbase is valid
	if header.Nonce != (emptyNonce) && !bytes.Equal(header.Nonce[:], nonceAuthVote) && !bytes.Equal(header.Nonce[:], nonceDropVote) {
		return errInvalidNonce
	}
	// Ensure that the mix digest is zero as we don't have fork protection currently
	if header.MixDigest != types.TendermintDigest {
		return errInvalidMixDigest
	}
	// Ensure that the block doesn't contain any uncles which are meaningless in BFT
	if header.UncleHash != nilUncleHash {
		return errInvalidUncleHash
	}
	// Ensure that the block's difficulty is meaningful (may not be correct at this point)
	if header.Difficulty == nil || header.Difficulty.Cmp(defaultDifficulty) != 0 {
		return errInvalidDifficulty
	}

	// If this is the genesis block there is no further verification to be
	// done.
	if header.IsGenesis() {
		return nil
	}
	// We expect the parent to be non nil when header is not the genesis header.
	if parent == nil {
		return errUnknownBlock
	}
	return sb.verifyHeaderAgainstParent(header, parent)
}

// verifyHeaderAgainstParent verifies that the given header is valid with respect to its parent.
func (sb *Backend) verifyHeaderAgainstParent(header, parent *types.Header) error {
	if parent.Number.Uint64() != header.Number.Uint64()-1 || parent.Hash() != header.ParentHash {
		return consensus.ErrUnknownAncestor
	}
	// Ensure that the block's timestamp isn't too close to it's parent
	if parent.Time+sb.config.BlockPeriod > header.Time {
		return errInvalidTimestamp
	}
	if err := sb.verifySigner(header, parent); err != nil {
		return err
	}

	return sb.verifyCommittedSeals(header, parent)
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers
// concurrently. The method returns a quit channel to abort the operations and
// a results channel to retrieve the async verifications (the order is that of
// the input slice).
func (sb *Backend) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{}, 1)
	results := make(chan error, len(headers))
	go func() {
		for i, header := range headers {
			var parent *types.Header
			switch {
			case i > 0:
				parent = headers[i-1]
			case i == 0:
				parent = chain.GetHeaderByHash(header.ParentHash)
			}
			err := sb.verifyHeader(header, parent)
			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	return abort, results
}

// VerifyUncles verifies that the given block's uncles conform to the consensus
// rules of a given engine.
func (sb *Backend) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) > 0 {
		return errInvalidUncleHash
	}
	return nil
}

// verifySigner checks that the signer is part of the committee.
func (sb *Backend) verifySigner(header, parent *types.Header) error {
	// resolve the authorization key and check against signers
	signer, err := types.Ecrecover(header)
	if err != nil {
		return err
	}

	if header.Coinbase != signer {
		return errInvalidCoinbase
	}

	// Signer should be in the validator set of previous block's extraData.
	if parent.CommitteeMember(signer) != nil {
		return nil
	}

	return errUnauthorized
}

// verifyCommittedSeals validates that the committed seals for header come from
// committee members and that the voting power of the committed seals constitutes
// a quorum.
func (sb *Backend) verifyCommittedSeals(header, parent *types.Header) error {
	// The length of Committed seals should be larger than 0
	if len(header.CommittedSeals) == 0 {
		return types.ErrEmptyCommittedSeals
	}

	// Setup map to track votes made by committee members
	votes := make(map[common.Address]int, len(parent.Committee))

	// Calculate total voting power
	var committeeVotingPower uint64
	for _, member := range parent.Committee {
		committeeVotingPower += member.VotingPower.Uint64()
	}

	// Total Voting power for this block
	var power uint64
	// The data that was sined over for this block
	headerSeal := tendermintCore.PrepareCommittedSeal(header.Hash(), int64(header.Round), header.Number)

	// 1. Get committed seals from current header
	for _, signedSeal := range header.CommittedSeals {
		// 2. Get the address from signature
		addr, err := types.GetSignatureAddress(headerSeal, signedSeal)
		if err != nil {
			sb.logger.Error("not a valid address", "err", err)
			return types.ErrInvalidSignature
		}

		member := parent.CommitteeMember(addr)
		if member == nil {
			sb.logger.Error(fmt.Sprintf("block had seal from non committee member %q", addr))
			return types.ErrInvalidCommittedSeals
		}

		votes[member.Address]++
		if votes[member.Address] > 1 {
			sb.logger.Error(fmt.Sprintf("committee member %q had multiple seals on block", addr))
			return types.ErrInvalidCommittedSeals
		}
		power += member.VotingPower.Uint64()
	}

	// We need at least a quorum for the block to be considered valid
	if power < bft.Quorum(committeeVotingPower) {
		return types.ErrInvalidCommittedSeals
	}

	return nil
}

// VerifySeal checks whether the crypto seal on a header is valid according to
// the consensus rules of the given engine.
func (sb *Backend) VerifySeal(chain consensus.ChainReader, header *types.Header) error {
	// Ensure the signer is part of the committee

	// The genesis block is not signed.
	if header.IsGenesis() {
		return errUnknownBlock
	}

	// ensure that the difficulty equals to defaultDifficulty
	if header.Difficulty.Cmp(defaultDifficulty) != 0 {
		return errInvalidDifficulty
	}

	parent := chain.GetHeaderByHash(header.ParentHash)
	if parent == nil {
		// TODO make this ErrUnknownAncestor
		return errUnknownBlock
	}
	return sb.verifySigner(header, parent)
}

// Prepare initializes the consensus fields of a block header according to the
// rules of a particular engine. The changes are executed inline.
func (sb *Backend) Prepare(chain consensus.ChainReader, header *types.Header) error {
	// unused fields, force to set to empty
	header.Coinbase = sb.Address()
	header.Nonce = emptyNonce
	header.MixDigest = types.TendermintDigest

	// copy the parent extra data as the header extra data
	number := header.Number.Uint64()
	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	// use the same difficulty for all blocks
	header.Difficulty = defaultDifficulty

	// set header's timestamp
	header.Time = new(big.Int).Add(big.NewInt(int64(parent.Time)), new(big.Int).SetUint64(sb.config.BlockPeriod)).Uint64()
	if int64(header.Time) < time.Now().Unix() {
		header.Time = uint64(time.Now().Unix())
	}
	return nil
}

// Finalize runs any post-transaction state modifications (e.g. block rewards)
// and assembles the final block.
//
// Note, the block header and state database might be updated to reflect any
// consensus rules that happen at finalization (e.g. block rewards).
func (sb *Backend) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	if sb.blockchain == nil {
		sb.blockchain = chain.(*core.BlockChain) // in the case of Finalize() called before the engine start()
	}

	committeeSet, _, err := sb.AutonityContractFinalize(header, chain, state, txs, receipts)
	if err != nil {
		sb.logger.Error("AutonityContractFinalize", "err", err.Error())
		return nil, err
	}

	//receipts = append(receipts, receipt)

	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))

	// drop uncles
	header.UncleHash = nilUncleHash

	// add committee to extraData's committee section
	header.Committee = committeeSet
	// Assemble and return the final block for sealing
	return types.NewBlock(header, txs, nil, receipts), nil
}

// AutonityContractFinalize is called to deploy the Autonity Contract at block #1. it returns as well the
// committee field containaining the list of committee members allowed to participate in consensus for the next block.
func (sb *Backend) AutonityContractFinalize(header *types.Header, chain consensus.ChainReader, state *state.StateDB,
	txs []*types.Transaction, receipts []*types.Receipt) (types.Committee, *types.Receipt, error) {
	sb.contractsMu.Lock()
	defer sb.contractsMu.Unlock()

	committeeSet, receipt, err := sb.blockchain.GetAutonityContractTendermint().FinalizeAndGetCommittee(txs, receipts, header, state)
	if err != nil {
		sb.logger.Error("Autonity Contract finalize returns err", "err", err)
		return nil, nil, err
	}
	return committeeSet, receipt, nil
}

// Seal generates a new block for the given input block with the local miner's
// seal place on top.
func (sb *Backend) Seal(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {
	sb.coreMu.RLock()
	isStarted := sb.coreStarted
	sb.coreMu.RUnlock()
	if !isStarted {
		return nil, ErrStoppedEngine
	}

	// update the block header and signature and propose the block to core engine
	header := block.Header()

	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		sb.logger.Error("Error ancestor")
		return nil, consensus.ErrUnknownAncestor
	}
	nodeAddress := sb.Address()
	if parent.CommitteeMember(nodeAddress) == nil {
		sb.logger.Error("error validator errUnauthorized", "addr", sb.address)
		return nil, errUnauthorized
	}

	block, err := sb.AddSeal(block)
	if err != nil {
		sb.logger.Error("seal error updateBlock", "err", err.Error())
		return nil, err
	}

	// wait for the timestamp of header, use this to adjust the block period
	delay := time.Unix(int64(block.Header().Time), 0).Sub(now())
	select {
	case <-time.After(delay):
	case <-stop:
		log.Error("Seal, Bail out <-stop")
		return nil, nil
	}

	//log.Debug("will crash ?? ", "sb.config.MinBlocksEmptyMining", sb.config.MinBlocksEmptyMining)
	//log.Debug("will crash ?? ", "block.Number()", block.Number())
	//log.Trace("If we're mining, but nothing is being processed, wake on new transactions ? ", "MinBlocksEmptyMining", sb.config.MinBlocksEmptyMining, "BlockNum", block.Number(), "BlockNum Cmp MinBlocksMining", block.Number().Cmp(sb.config.MinBlocksEmptyMining))
	//if len(block.Transactions()) == 0 && block.Number().Cmp(sb.config.MinBlocksEmptyMining) >= 0 {
	//	log.Debug("Seal, Bail out errWaitTransactions")
	//	return nil, errWaitTransactions
	//}

	// get the proposed block hash and clear it if the seal() is completed.
	sb.sealMu.Lock()
	sb.proposedBlockHash = block.Hash()
	sb.logger.Debug("get the proposed block hash and clear it if the seal() is completed ", "hash", sb.proposedBlockHash)
	clear := func() {
		sb.proposedBlockHash = common.Hash{}
		sb.sealMu.Unlock()
	}
	defer clear()

	// post block into BFT engine
	go sb.eventMux.Post(events.NewUnminedBlockEvent{ //RequestEvent
		NewUnminedBlock: *block,
	})

	for {
		select {
		case result := <-sb.commitCh:
			sb.logger.Debug("Seal, got back the committed block from commitChBlock")
			// if the block hash and the hash from channel are the same,
			// return the result. Otherwise, keep waiting the next hash.
			if result != nil && block.Hash() == result.Hash() {
				log.Debug("Seal, lock hash and the hash from channel are the same. return result", "block.Hash", block.Hash(), "result.Hash", result.Hash(), "result", result.String())
				return result, nil
			} else {
				log.Error("Seal, lock hash and the hash from channel NOT the same. Keep waiting the next hash.", "block.Hash", block.Hash(), "result.Hash", result.Hash())
			}
		case <-stop:
			log.Error("Seal, Bail out for, select, <-stop")
			return nil, nil
		}
	}

}

//func (sb *Backend) setResultChan(results chan *types.Block) {
//	sb.coreMu.Lock()
//	defer sb.coreMu.Unlock()
//
//	sb.commitCh = results
//}

func (sb *Backend) sendResultChan(block *types.Block) {
	sb.coreMu.Lock()
	defer sb.coreMu.Unlock()

	sb.commitCh <- block
}

func (sb *Backend) isResultChanNil() bool {
	sb.coreMu.RLock()
	defer sb.coreMu.RUnlock()

	return sb.commitCh == nil
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns the difficulty
// that a new block should have based on the previous blocks in the blockchain and the
// current signer.
func (sb *Backend) CalcDifficulty(chain consensus.ChainReader, time uint64, parent *types.Header) *big.Int {
	return defaultDifficulty
}

func (sb *Backend) SetProposedBlockHash(hash common.Hash) {
	sb.proposedBlockHash = hash
}

// update timestamp and signature of the block based on its number of transactions
func (sb *Backend) AddSeal(block *types.Block) (*types.Block, error) {
	header := block.Header()
	// sign the hash
	seal, err := sb.Sign(types.SigHash(header).Bytes())
	if err != nil {
		return nil, err
	}

	err = types.WriteSeal(header, seal)
	if err != nil {
		return nil, err
	}

	return block.WithSeal(header), nil
}

// APIs returns the RPC APIs this consensus engine provides.
func (sb *Backend) APIs(chain consensus.ChainReader) []rpc.API {
	return []rpc.API{{
		Namespace: "tendermint",
		Version:   "1.0",
		Service:   &API{chain: chain, tendermint: sb, getCommittee: getCommittee},
		Public:    true,
	}}
}

// getCommittee retrieves the committee for the given header.
func getCommittee(header *types.Header, chain consensus.ChainReader) (types.Committee, error) {
	parent := chain.GetHeaderByHash(header.ParentHash)
	if parent == nil {
		return nil, errUnknownBlock
	}
	return parent.Committee, nil
}

// Start implements consensus.tendermint.Start
func (sb *Backend) Start(_ context.Context, chain consensus.ChainReader, currentBlock func() *types.Block, hasBadBlock func(hash common.Hash) bool) error {
	// the mutex along with coreStarted should prevent double start
	sb.coreMu.Lock()
	defer sb.coreMu.Unlock()
	if sb.coreStarted {
		return ErrStartedEngine
	}

	sb.stopped = make(chan struct{})

	// clear previous data
	sb.proposedBlockHash = common.Hash{}
	if sb.commitCh != nil {
		close(sb.commitCh)
	}
	sb.commitCh = make(chan *types.Block, 1)

	sb.blockchain = chain.(*core.BlockChain) // in the case of Finalize() called before the engine start()
	sb.currentBlock = currentBlock
	sb.hasBadBlock = hasBadBlock

	//TODO: is this required or not ?
	//if err := sb.core.Start(context.Background(), chain, currentBlock, hasBadBlock); err != nil {
	//	return err
	//}

	sb.coreStarted = true

	return nil
}

// Stop implements consensus.tendermint.Stop
func (sb *Backend) Stop() error {
	// the mutex along with coreStarted should prevent double stop
	sb.coreMu.Lock()
	defer sb.coreMu.Unlock()

	if !sb.coreStarted {
		return ErrStoppedEngine
	}

	// Stop Tendermint
	sb.core.Stop()

	sb.coreStarted = false

	close(sb.stopped)

	return nil
}

func (sb *Backend) SealHash(header *types.Header) common.Hash {
	return types.SigHash(header)
}

func (sb *Backend) SetBlockchain(bc *core.BlockChain) {
	sb.blockchain = bc

	sb.currentBlock = bc.CurrentBlock
	sb.hasBadBlock = bc.HasBadBlock
}


// Protocol implements consensus.Engine.Protocol
func (sb *Backend) ProtocolOld() consensus.Protocol {
	return consensus.TendermintProtocol
}

