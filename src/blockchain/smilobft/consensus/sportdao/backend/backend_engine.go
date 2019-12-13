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
	"errors"
	"fmt"
	"go-smilo/src/blockchain/smilobft/consensus/sportdao/fullnode"
	"go-smilo/src/blockchain/smilobft/core"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	lru "github.com/hashicorp/golang-lru"

	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"go-smilo/src/blockchain/smilobft/rpc"

	"github.com/orinocopay/go-etherutils"

	"go-smilo/src/blockchain/smilobft/consensus"
	"go-smilo/src/blockchain/smilobft/consensus/sportdao"
	"go-smilo/src/blockchain/smilobft/core/state"
	"go-smilo/src/blockchain/smilobft/core/types"
)

const (
	checkpointInterval = 1024 // Number of blocks after which to save the vote snapshot to the database
	inmemorySnapshots  = 128  // Number of recent vote snapshots to keep in memory
	inmemoryPeers      = 40
	inmemoryMessages   = 1024
)

var (
	// errInvalidBlockProposal is returned when a proposal is malformed.
	errInvalidBlockProposal = errors.New("invalid block proposal")
	// errInvalidSignature is returned when given signature is not signed by given
	// address.
	errInvalidSignature = errors.New("invalid signature")
	// errUnknownBlock is returned when the list of fullnodes is requested for a block
	// that is not part of the local blockchain.
	errUnknownBlock = errors.New("unknown block")
	// errUnauthorized is returned if a header is signed by a non authorized entity.
	errUnauthorized = errors.New("unauthorized")
	// errInvalidDifficulty is returned if the difficulty of a block is not 1
	errInvalidDifficulty = errors.New("invalid difficulty")
	// errInvalidExtraDataFormat is returned when the extra data format is incorrect
	errInvalidExtraDataFormat = errors.New("invalid extra data format")
	// errInvalidMixDigest is returned if a block's mix digest is not Sport digest.
	errInvalidMixDigest = errors.New("invalid Sport mix digest")
	// errInvalidNonce is returned if a block's nonce is invalid
	errInvalidNonce = errors.New("invalid nonce")
	// errInvalidUncleHash is returned if a block contains an non-empty uncle list.
	errInvalidUncleHash = errors.New("non empty uncle hash")
	// errInconsistentValidatorSet is returned if the validator set is inconsistent
	errInconsistentValidatorSet = errors.New("inconsistent validator set")
	// errInvalidTimestamp is returned if the timestamp of a block is lower than the previous block's timestamp + the minimum block period.
	errInvalidTimestamp = errors.New("invalid timestamp")
	// errInvalidVotingChain is returned if an authorization list is attempted to
	// be modified via out-of-range or non-contiguous headers.
	errInvalidVotingChain = errors.New("invalid voting chain")
	// errInvalidVote is returned if a nonce value is something else that the two
	// allowed constants of 0x00..0 or 0xff..f.
	errInvalidVote = errors.New("vote nonce not 0x00..0 or 0xff..f")
	// errInvalidCommittedSeals is returned if the committed seal is not signed by any of parent fullnodes.
	errInvalidCommittedSeals = errors.New("invalid committed seals")
	// errEmptyCommittedSeals is returned if the field of committed seals is zero.
	errEmptyCommittedSeals = errors.New("zero committed seals")
	// errMismatchTxhashes is returned if the TxHash in header is mismatch.

	errMismatchTxhashes = errors.New("mismatch transaction hashes")

	// errWaitTransactions is returned if an empty block is attempted to be sealed
	// on an instant chain (0 second period). It's important to refuse these as the
	// block reward is zero, so an empty block just bloats the chain... fast.
	errWaitTransactions = errors.New("waiting for transactions")
)
var (
	defaultDifficulty = big.NewInt(1)
	nilUncleHash      = types.CalcUncleHash(nil) // Always Keccak256(RLP([])) as uncles are meaningless outside of PoW.
	emptyNonce        = types.BlockNonce{}
	now               = time.Now

	nonceAuthVote = hexutil.MustDecode("0xffffffffffffffff") // Magic nonce number to vote on adding a new fullnode
	nonceDropVote = hexutil.MustDecode("0x0000000000000000") // Magic nonce number to vote on removing a fullnode.

	inmemoryAddresses  = 20 // Number of recent addresses from ecrecover
	recentAddresses, _ = lru.NewARC(inmemoryAddresses)

	// smiloTokenMetricsTable is the Block reward in wei to a fullnode when successfully mining a block
	smiloTokenMetricsTable = map[*big.Int]*big.Int{
		big.NewInt(20000000):   getSmiloValue("4000000000 gwei"), // 4 smilo
		big.NewInt(40000000):   getSmiloValue("2000000000 gwei"), // 2 smilo
		big.NewInt(60000000):   getSmiloValue("1750000000 gwei"), // 1.75 smilo
		big.NewInt(80000000):   getSmiloValue("1500000000 gwei"), // 1.5 smilo
		big.NewInt(100000000):  getSmiloValue("1250000000 gwei"), // 1.25 smilo
		big.NewInt(120000000):  getSmiloValue("1000000000 gwei"), // 1 smilo
		big.NewInt(140000000):  getSmiloValue("800000000 gwei"),  // 0.8 smilo
		big.NewInt(160000000):  getSmiloValue("600000000 gwei"),  // 0.6 smilo
		big.NewInt(180000000):  getSmiloValue("400000000 gwei"),  // 0.4 smilo
		big.NewInt(200000000):  getSmiloValue("200000000 gwei"),  // 0.2 smilo
		big.NewInt(400000000):  getSmiloValue("100000000 gwei"),  // 0.1 smilo
		big.NewInt(800000000):  getSmiloValue("50000000 gwei"),   // 0.05 smilo
		big.NewInt(1600000000): getSmiloValue("25000000 gwei"),   // 0.025 smilo
	}
)

func getSmiloValue(value string) *big.Int {
	v, _ := etherutils.StringToWei(value)
	return v
}

// Author (clique override) retrieves the Ethereum address of the account that minted the given
// block, which may be different from the header's coinbase if a consensus
// engine is based on signatures.
func (sb *Backend) Author(header *types.Header) (common.Address, error) {
	return types.Ecrecover(header)
}

// VerifyHeader (clique override) checks whether a header conforms to the consensus rules of a
// given engine. Verifying the seal may be done optionally here, or explicitly
// via the VerifySeal method.
func (sb *Backend) VerifyHeader(chain consensus.ChainReader, header *types.Header, seal bool) error {
	return sb.verifyHeader(chain, header, nil)
}

// verifyHeader (clique override) checks whether a header conforms to the consensus rules.The
// caller may optionally pass in a batch of parents (ascending order) to avoid
// looking those up from the database. This is useful for concurrently verifying
// a batch of new headers.
func (sb *Backend) verifyHeader(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	if header.Number == nil {
		return errUnknownBlock
	}

	// Don't waste time checking blocks from the future
	if big.NewInt(int64(header.Time)).Cmp(big.NewInt(now().Unix())) > 0 {
		return consensus.ErrFutureBlock
	}

	// Ensure that the extra data format is satisfied
	if _, err := types.ExtractSportExtra(header); err != nil {
		return errInvalidExtraDataFormat
	}

	// Ensure that the coinbase is valid
	if header.Nonce != (emptyNonce) && !bytes.Equal(header.Nonce[:], nonceAuthVote) && !bytes.Equal(header.Nonce[:], nonceDropVote) {
		return errInvalidNonce
	}
	// Ensure that the mix digest is zero as we don't have fork protection currently
	if header.MixDigest != types.SportDigest {
		return errInvalidMixDigest
	}
	// Ensure that the block doesn't contain any uncles which are meaningless in Sport
	if header.UncleHash != nilUncleHash {
		return errInvalidUncleHash
	}
	// Ensure that the block's difficulty is meaningful (may not be correct at this point)
	if header.Difficulty == nil || header.Difficulty.Cmp(defaultDifficulty) != 0 {
		return errInvalidDifficulty
	}

	return sb.verifyCascadingFields(chain, header, parents)
}

// verifyCascadingFields (clique override) verifies all the header fields that are not standalone,
// rather depend on a batch of previous headers. The caller may optionally pass
// in a batch of parents (ascending order) to avoid looking those up from the
// database. This is useful for concurrently verifying a batch of new headers.
func (sb *Backend) verifyCascadingFields(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	// The genesis block is the always valid dead-end
	number := header.Number.Uint64()
	if number == 0 {
		return nil
	}
	// Ensure that the block's timestamp isn't too close to it's parent
	var parent *types.Header
	if len(parents) > 0 {
		parent = parents[len(parents)-1]
	} else {
		parent = chain.GetHeader(header.ParentHash, number-1)
	}
	if parent == nil || parent.Number.Uint64() != number-1 || parent.Hash() != header.ParentHash {
		return consensus.ErrUnknownAncestor
	}
	if parent.Time+sb.config.BlockPeriod > header.Time {
		return errInvalidTimestamp
	}

	if err := sb.verifySigner(chain, header, parents); err != nil {
		return err
	}

	return sb.verifyCommittedSeals(chain, header, parents)
}

// VerifyHeaders (clique override) is similar to VerifyHeader, but verifies a batch of headers
// concurrently. The method returns a quit channel to abort the operations and
// a results channel to retrieve the async verifications (the order is that of
// the input slice).
func (sb *Backend) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(headers))
	go func() {
		for i, header := range headers {
			err := sb.verifyHeader(chain, header, headers[:i])

			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	return abort, results
}

// VerifyUncles (clique override) verifies that the given block's uncles conform to the consensus
// rules of a given engine.
func (sb *Backend) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) > 0 {
		return errInvalidUncleHash
	}
	return nil
}


// VerifySeal (clique override) checks whether the crypto seal on a header is valid according to
// the consensus rules of the given engine.
func (sb *Backend) VerifySeal(chain consensus.ChainReader, header *types.Header) error {
	// get parent header and ensure the signer is in parent's fullnode set
	number := header.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}

	// ensure that the difficulty equals to defaultDifficulty
	if header.Difficulty.Cmp(defaultDifficulty) != 0 {
		return errInvalidDifficulty
	}
	return sb.verifySigner(chain, header, nil)
}

// Prepare (clique override) initializes the consensus fields of a block header according to the
// rules of a particular engine. The changes are executed inline.
func (sb *Backend) Prepare(chain consensus.ChainReader, header *types.Header) error {
	// unused fields, force to set to empty
	header.Coinbase = sb.address
	header.Nonce = emptyNonce
	header.MixDigest = types.SportDigest

	// copy the parent extra data as the header extra data
	number := header.Number.Uint64()
	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	// use the same difficulty for all blocks
	header.Difficulty = defaultDifficulty

	var parents []*types.Header
	parents = append(parents, parent)
	fullnodeAddresses, err := sb.retrieveValidators(header, parents, chain)
	if err != nil {
		log.Error("Could not assemble the voting snapshot from retrieveValidators", "err", err)
		return err
	}
	fullnodeSet := fullnode.NewSet(fullnodeAddresses, sb.config.GetProposerPolicy())

	fullnodes := make([]common.Address, 0, fullnodeSet.Size())
	for _, fullnode := range fullnodeSet.List() {
		fullnodes = append(fullnodes, fullnode.Address())
	}
	for i := 0; i < len(fullnodes); i++ {
		for j := i + 1; j < len(fullnodes); j++ {
			if bytes.Compare(fullnodes[i][:], fullnodes[j][:]) > 0 {
				fullnodes[i], fullnodes[j] = fullnodes[j], fullnodes[i]
			}
		}
	}

	// add fullnodes in snapshot to extraData's fullnodes section
	extra, err := prepareExtra(header, fullnodes)
	if err != nil {
		log.Error("Could not add fullnodes in snapshot to extraData's fullnodes section.", "err", err)
		return err
	}
	header.Extra = extra

	// set header's timestamp
	header.Time = parent.Time + sb.config.BlockPeriod
	if int64(header.Time) < time.Now().Unix() {
		header.Time = uint64(time.Now().Unix())
	}
	return nil
}

// Finalize (clique override) runs any post-transaction state modifications (e.g. block rewards)
// and assembles the final block.
//
// Note, the block header and state database might be updated to reflect any
// consensus rules that happen at finalization (e.g. block rewards).
func (sb *Backend) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction,
	uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {

	if sb.blockchain == nil {
		sb.blockchain = chain.(*core.BlockChain) // in the case of Finalize() called before the engine start()
	}

	validators, err := sb.getValidators(header, chain, state)
	if err != nil {
		fmt.Println("consensus/istanbul/backend/engine.go:337 getValidators err", err)
		return nil, err
	}

	// add validators to extraData's validators section
	if header.Extra, err = types.PrepareExtra(header.Extra, validators); err != nil {
		return nil, err
	}

	// warn for empty blocks
	number := header.Number.Int64()

	//Will generate rewards for every block until block 40000000
	//From this point on, ddd block rewards in Sport only if there is transactions in it
	if header.Number.Cmp(big.NewInt(1)) > 0 && len(txs) > 0 || number < 40000000 {
		AccumulateRewards(sb.config.CommunityAddress, state, header)
	}

	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))

	// drop uncles
	header.UncleHash = nilUncleHash

	// Assemble and return the final block for sealing
	return types.NewBlock(header, txs, nil, receipts), nil
}

// Seal (clique override) generates a new block for the given input block with the local miner's seal place on top.
func (sb *Backend) Seal(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {
	// update the block header timestamp and signature and propose the block to core engine
	header := block.Header()
	number := header.Number.Uint64()

	// Bail out if we're unauthorized to sign a block
	if _, v := sb.Fullnodes(number).GetByAddress(sb.address); v == nil {
		sb.logger.Error("Seal, Bail out if we're unauthorized to sign a block", "addr", sb.address.String())
		return nil, errUnauthorized
	}

	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		log.Error("Seal, Bail out ErrUnknownAncestor")
		return nil, consensus.ErrUnknownAncestor
	}
	block, err := sb.updateBlock(block)
	if err != nil {
		log.Error("Seal, Bail out updateBlock", "err", err)
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

	log.Debug("will crash ?? ", "sb.config.MinBlocksEmptyMining", sb.config.MinBlocksEmptyMining)
	log.Debug("will crash ?? ", "block.Number()", block.Number())
	log.Trace("If we're mining, but nothing is being processed, wake on new transactions ? ", "MinBlocksEmptyMining", sb.config.MinBlocksEmptyMining, "BlockNum", block.Number(), "BlockNum Cmp MinBlocksMining", block.Number().Cmp(sb.config.MinBlocksEmptyMining))
	if len(block.Transactions()) == 0 && block.Number().Cmp(sb.config.MinBlocksEmptyMining) >= 0 {
		log.Debug("Seal, Bail out errWaitTransactions")
		return nil, errWaitTransactions
	}

	// get the proposed block hash and clear it if the seal() is completed.
	sb.sealMu.Lock()
	sb.proposedBlockHash = block.Hash()
	sb.logger.Debug("get the proposed block hash and clear it if the seal() is completed ", "hash", sb.proposedBlockHash)
	clear := func() {
		sb.proposedBlockHash = common.Hash{}
		sb.sealMu.Unlock()
	}
	defer clear()

	// post block into Sport engine
	go func() {
		requestEvent := sportdao.RequestEvent{
			BlockProposal: block,
		}
		err := sb.EventMux().Post(requestEvent)
		if err != nil {
			log.Error("Seal, Could not send sportdao.RequestEvent message with block proposal, ", "RequestEvent", requestEvent)
		}
	}()

	for {
		select {
		case result := <-sb.commitChBlock:
			sb.logger.Debug("Seal, got back the committed block from commitChBlock")
			// if the block hash and the hash from channel are the same,
			// return the result. Otherwise, keep waiting the next hash.
			if result != nil && block.Hash() == result.Hash() {
				log.Debug("Seal, lock hash and the hash from channel are the same. return result", "block.Hash", block.Hash(), "result.Hash", result.Hash(), "result", result)
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

// APIs (clique override) returns the RPC APIs this consensus engine provides.
func (sb *Backend) APIs(chain consensus.ChainReader) []rpc.API {
	return []rpc.API{{
		Namespace: "smilobftdao",
		Version:   "1.0",
		Service:   &API{chain: chain, smilo: sb},
		Public:    true,
	}}
}


// SealHash returns the hash of a block prior to it being sealed.
func (sb *Backend) SealHash(header *types.Header) common.Hash {
	return types.SigHash(header)
}
