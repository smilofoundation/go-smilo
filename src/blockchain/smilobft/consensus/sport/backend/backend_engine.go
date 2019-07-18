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
	"go-smilo/src/blockchain/smilobft/rpc"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	lru "github.com/hashicorp/golang-lru"

	"bytes"
	"math/rand"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/orinocopay/go-etherutils"

	"golang.org/x/crypto/sha3"

	"go-smilo/src/blockchain/smilobft/consensus"
	"go-smilo/src/blockchain/smilobft/consensus/sport"
	"go-smilo/src/blockchain/smilobft/consensus/sport/fullnode"
	"go-smilo/src/blockchain/smilobft/core"
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
func (sb *backend) Author(header *types.Header) (common.Address, error) {
	return ecrecover(header)
}

// VerifyHeader (clique override) checks whether a header conforms to the consensus rules of a
// given engine. Verifying the seal may be done optionally here, or explicitly
// via the VerifySeal method.
func (sb *backend) VerifyHeader(chain consensus.ChainReader, header *types.Header, seal bool) error {
	return sb.verifyHeader(chain, header, nil)
}

// verifyHeader (clique override) checks whether a header conforms to the consensus rules.The
// caller may optionally pass in a batch of parents (ascending order) to avoid
// looking those up from the database. This is useful for concurrently verifying
// a batch of new headers.
func (sb *backend) verifyHeader(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	if header.Number == nil {
		return errUnknownBlock
	}

	// Don't waste time checking blocks from the future
	if header.Time.Cmp(big.NewInt(now().Unix())) > 0 {
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
func (sb *backend) verifyCascadingFields(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
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
	if parent.Time.Uint64()+sb.config.BlockPeriod > header.Time.Uint64() {
		return errInvalidTimestamp
	}
	// Verify fullnodes in extraData. Fullnodes in snapshot and extraData should be the same.
	snap, err := sb.snapshot(chain, number-1, header.ParentHash, parents)
	if err != nil {
		return err
	}
	fullnodes := make([]byte, len(snap.fullnodes())*common.AddressLength)
	for i, f := range snap.fullnodes() {
		copy(fullnodes[i*common.AddressLength:], f[:])
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
func (sb *backend) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
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
func (sb *backend) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) > 0 {
		return errInvalidUncleHash
	}
	return nil
}

// VerifySeal (clique override) checks whether the crypto seal on a header is valid according to
// the consensus rules of the given engine.
func (sb *backend) VerifySeal(chain consensus.ChainReader, header *types.Header) error {
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
func (sb *backend) Prepare(chain consensus.ChainReader, header *types.Header) error {
	// unused fields, force to set to empty
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

	// Assemble the voting snapshot
	snap, err := sb.snapshot(chain, number-1, header.ParentHash, nil)
	if err != nil {
		log.Error("Could not assemble the voting snapshot.", "err", err)
		return err
	}

	// get valid candidate list
	sb.candidatesLock.RLock()
	var addresses []common.Address
	var authorizes []bool

	if sb.coreStarted {
		statedb, _, err := sb.chain.State()
		if err != nil {
			log.Error("Could not Prepare candidates, got error with statedb", "error", err)
			sb.candidatesLock.RUnlock()
			return err
		}
		for address, authorize := range sb.candidates {
			if snap.checkVote(address, authorize) {

				//check that the address have 10k smilos in it
				if statedb.GetBalance(address).Cmp(big.NewInt(sb.config.MinFunds)) < 0 {
					log.Error("Could not Prepare candidate, failed to check that the address have enough smilos in it",
						"error", core.ErrInsufficientFunds.Error(), "address", address)
				} else {
					log.Debug("Prepared candidate with MinFunds ok",
						"error", core.ErrInsufficientFunds.Error(), "address", address)
					addresses = append(addresses, address)
					authorizes = append(authorizes, authorize)
				}
			}
		}
	} else {
		log.Debug("****** sb core is not yet started, will not check that the candidates have 10k smilos in it ******")
		for address, authorize := range sb.candidates {
			if snap.checkVote(address, authorize) {
				addresses = append(addresses, address)
				authorizes = append(authorizes, authorize)
			}
		}
	}
	sb.candidatesLock.RUnlock()

	// pick one of the candidates randomly
	if len(addresses) > 0 {
		index := rand.Intn(len(addresses))
		// add fullnode voting in coinbase
		header.Coinbase = addresses[index]
		if authorizes[index] {
			copy(header.Nonce[:], nonceAuthVote)
		} else {
			copy(header.Nonce[:], nonceDropVote)
		}
	}

	// add fullnodes in snapshot to extraData's fullnodes section
	extra, err := prepareExtra(header, snap.fullnodes())
	if err != nil {
		log.Error("Could not add fullnodes in snapshot to extraData's fullnodes section.", "err", err)
		return err
	}
	header.Extra = extra

	// set header's timestamp
	header.Time = new(big.Int).Add(parent.Time, new(big.Int).SetUint64(sb.config.BlockPeriod))
	if header.Time.Int64() < time.Now().Unix() {
		header.Time = big.NewInt(time.Now().Unix())
	}
	return nil
}

// Finalize (clique override) implements consensus.Engine, ensuring no uncles are set, nor block
// rewards given.
func (sb *backend) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header) {
	// No block rewards in PoA, so the state remains as is and uncles are dropped
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
	header.UncleHash = types.CalcUncleHash(nil)
}

// FinalizeAndAssemble (clique override) runs any post-transaction state modifications (e.g. block rewards)
// and assembles the final block.
//
// Note, the block header and state database might be updated to reflect any
// consensus rules that happen at finalization (e.g. block rewards).
func (sb *backend) FinalizeAndAssemble(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction,
	uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {

	// warn for empty blocks
	//number := header.Number.Int64()

	//Will generate rewards for every block until block 40000000
	//From this point on, ddd block rewards in Sport only if there is transactions in it
	//if header.Number.Cmp(big.NewInt(1)) > 0 && len(txs) > 0 || number < 40000000 {
	//	AccumulateRewards(sb.config.CommunityAddress, state, header)
	//}

	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))

	// drop uncles
	header.UncleHash = nilUncleHash

	// Assemble and return the final block for sealing
	return types.NewBlock(header, txs, nil, receipts), nil
}

// Seal generates a new block for the given input block with the local miner's
// seal place on top.
func (sb *backend) Seal(chain consensus.ChainReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {

	// update the block header timestamp and signature and propose the block to core engine
	header := block.Header()
	number := header.Number.Uint64()

	// Bail out if we're unauthorized to sign a block
	snap, err := sb.snapshot(chain, number-1, header.ParentHash, nil)
	if err != nil {
		return err
	}
	if _, v := snap.FullnodeSet.GetByAddress(sb.address); v == nil {
		return errUnauthorized
	}

	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	block, err = sb.updateBlock(parent, block)
	if err != nil {
		return err
	}

	// wait for the timestamp of header, use this to adjust the block period
	delay := time.Unix(block.Header().Time.Int64(), 0).Sub(now())
	select {
	case <-time.After(delay):
	case <-stop:
		results <- nil
		return nil
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

	// post block into Istanbul engine
	go sb.EventMux().Post(sport.RequestEvent{
		BlockProposal: block,
	})
	for {
		select {
		case result := <-sb.commitChBlock:
			sb.logger.Debug("Seal got back the committed block from commitChBlock")
			// if the block hash and the hash from channel are the same,
			// return the result. Otherwise, keep waiting the next hash.
			if result != nil && block.Hash() == result.Hash() {
				results <- result
				return nil
			}
		case <-stop:
			sb.logger.Debug("Seal got stop inside commitChBlock loop, WTF ?")
			results <- nil
			return nil
		}
	}
}

// APIs (clique override) returns the RPC APIs this consensus engine provides.
func (sb *backend) APIs(chain consensus.ChainReader) []rpc.API {
	return []rpc.API{{
		Namespace: "smilobft",
		Version:   "1.0",
		Service:   &API{chain: chain, smilo: sb},
		Public:    true,
	}}
}

// snapshot (clique override) retrieves the authorization snapshot at a given point in time.
func (sb *backend) snapshot(chain consensus.ChainReader, number uint64, hash common.Hash, parents []*types.Header) (*Snapshot, error) {
	// Search for a snapshot in memory or on disk for checkpoints
	var (
		headers []*types.Header
		snap    *Snapshot
	)
	for snap == nil {
		// If an in-memory snapshot was found, use that
		if s, ok := sb.recents.Get(hash); ok {
			snap = s.(*Snapshot)
			break
		}
		// If an on-disk checkpoint snapshot can be found, use that
		if number%checkpointInterval == 0 {
			if s, err := loadSnapshot(sb.config.Epoch, sb.db, hash); err == nil {
				log.Info("Loaded voting snapshot from disk", "number", number, "hash", hash, "fullnodes", s.fullnodes())
				snap = s
				break
			}
		}
		// If we're at block zero, make a snapshot
		if number == 0 {
			genesis := chain.GetHeaderByNumber(0)
			if err := sb.VerifyHeader(chain, genesis, false); err != nil {
				return nil, err
			}
			sportExtra, err := types.ExtractSportExtra(genesis)
			if err != nil {
				return nil, err
			}
			snap = newSnapshot(sb.config.Epoch, 0, genesis.Hash(), fullnode.NewFullnodeSet(sportExtra.Fullnodes, sb.config.SpeakerPolicy))
			if err := snap.store(sb.db); err != nil {
				log.Error("Could not store the genesis snapshot to disk!! ")
				return nil, err
			}
			log.Info("Stored genesis voting snapshot to disk")
			break
		}
		// No snapshot for this header, gather the header and move backward
		var header *types.Header
		if len(parents) > 0 {
			// If we have explicit parents, pick from there (enforced)
			header = parents[len(parents)-1]
			if header.Hash() != hash || header.Number.Uint64() != number {
				return nil, consensus.ErrUnknownAncestor
			}
			parents = parents[:len(parents)-1]
		} else {
			// No explicit parents (or no more left), reach out to the database
			header = chain.GetHeader(hash, number)
			if header == nil {
				return nil, consensus.ErrUnknownAncestor
			}
		}
		headers = append(headers, header)
		number, hash = number-1, header.ParentHash
	}
	// Previous snapshot found, apply any pending headers on top of it
	for i := 0; i < len(headers)/2; i++ {
		headers[i], headers[len(headers)-1-i] = headers[len(headers)-1-i], headers[i]
	}
	snap, err := snap.apply(headers)
	if err != nil {
		return nil, err
	}
	sb.recents.Add(snap.Hash, snap)

	// If we've generated a new checkpoint snapshot, save to disk
	if snap.Number%checkpointInterval == 0 && len(headers) > 0 {
		if err = snap.store(sb.db); err != nil {
			return nil, err
		}
		log.Trace("Stored voting snapshot to disk", "number", snap.Number, "hash", snap.Hash)
	}
	return snap, err
}

// sigHash (clique override) returns the hash which is used as input for the Sport
// signing. It is the hash of the entire header apart from the 65 byte signature
// contained at the end of the extra data.
//
// Note, the method requires the extra data to be at least 65 bytes, otherwise it
// panics. This is done to avoid accidentally using both forms (signature present
// or not), which could be abused to produce different hashes for the same header.
// FIXME: Need to update this for Sport
func sigHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.NewLegacyKeccak256()

	// Clean seal is required for calculating speaker seal.
	rlp.Encode(hasher, types.SportFilteredHeader(header, false))
	hasher.Sum(hash[:0])
	return hash
}

// SealHash returns the hash of a block prior to it being sealed.
func (sb *backend) SealHash(header *types.Header) common.Hash {
	return sigHash(header)
}
