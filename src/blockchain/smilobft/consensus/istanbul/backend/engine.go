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
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/orinocopay/go-etherutils"

	"go-smilo/src/blockchain/smilobft/core"
	"go-smilo/src/blockchain/smilobft/rpc"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"go-smilo/src/blockchain/smilobft/consensus"
	"go-smilo/src/blockchain/smilobft/consensus/istanbul"
	istanbulCore "go-smilo/src/blockchain/smilobft/consensus/istanbul/core"
	"go-smilo/src/blockchain/smilobft/consensus/istanbul/validator"
	"go-smilo/src/blockchain/smilobft/core/state"
	"go-smilo/src/blockchain/smilobft/core/types"
)

const (
	inmemorySnapshots = 128 // Number of recent vote snapshots to keep in memory
	inmemoryPeers     = 40
	inmemoryMessages  = 1024
)

var (
	// errInvalidProposal is returned when a prposal is malformed.
	errInvalidProposal = errors.New("invalid proposal")
	// errUnknownBlock is returned when the list of validators is requested for a block
	// that is not part of the local blockchain.
	errUnknownBlock = errors.New("unknown block")
	// errUnauthorized is returned if a header is signed by a non authorized entity.
	errUnauthorized = errors.New("unauthorized")
	// errInvalidDifficulty is returned if the difficulty of a block is not 1
	errInvalidDifficulty = errors.New("invalid difficulty")
	// errInvalidExtraDataFormat is returned when the extra data format is incorrect
	errInvalidExtraDataFormat = errors.New("invalid extra data format")
	// errInvalidMixDigest is returned if a block's mix digest is not Istanbul digest.
	errInvalidMixDigest = errors.New("invalid Istanbul mix digest")
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
	//errInvalidVotingChain = errors.New("invalid voting blockchain")
	// errInvalidVote is returned if a nonce value is something else that the two
	// allowed constants of 0x00..0 or 0xff..f.
	//errInvalidVote = errors.New("vote nonce not 0x00..0 or 0xff..f")
	// errMismatchTxhashes is returned if the TxHash in header is mismatch.
	errMismatchTxhashes = errors.New("mismatch transactions hashes")
	errWaitTransactions = errors.New("waiting for transactions")
)
var (
	defaultDifficulty = big.NewInt(1)
	nilUncleHash      = types.CalcUncleHash(nil) // Always Keccak256(RLP([])) as uncles are meaningless outside of PoW.
	emptyNonce        = types.BlockNonce{}
	now               = time.Now

	nonceAuthVote = hexutil.MustDecode("0xffffffffffffffff") // Magic nonce number to vote on adding a new validator
	nonceDropVote = hexutil.MustDecode("0x0000000000000000") // Magic nonce number to vote on removing a validator.

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

// Author retrieves the Ethereum address of the account that minted the given
// block, which may be different from the header's coinbase if a consensus
// engine is based on signatures.
func (sb *Backend) Author(header *types.Header) (common.Address, error) {
	return types.Ecrecover(header)
}

// VerifyHeader checks whether a header conforms to the consensus rules of a
// given engine. Verifying the seal may be done optionally here, or explicitly
// via the VerifySeal method.
func (sb *Backend) VerifyHeader(chain consensus.ChainReader, header *types.Header, seal bool) error {
	return sb.verifyHeader(chain, header, nil)
}

// verifyHeader checks whether a header conforms to the consensus rules.The
// caller may optionally pass in a batch of parents (ascending order) to avoid
// looking those up from the database. This is useful for concurrently verifying
// a batch of new headers.
func (sb *Backend) verifyHeader(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	if header.Number == nil {
		return errUnknownBlock
	}

	// Don't waste time checking blocks from the future
	if big.NewInt(int64(header.Time)).Cmp(big.NewInt(now().Unix())) > 0 {
		log.Warn("$$$ Don't waste time checking blocks from the future, ", "header.Time", header.Time, "now().Unix()", now().Unix())
		return consensus.ErrFutureBlock
	}

	// Ensure that the extra data format is satisfied
	if _, err := types.ExtractBFTHeaderExtra(header); err != nil {
		return errInvalidExtraDataFormat
	}

	// Ensure that the coinbase is valid
	if header.Nonce != (emptyNonce) && !bytes.Equal(header.Nonce[:], nonceAuthVote) && !bytes.Equal(header.Nonce[:], nonceDropVote) {
		return errInvalidNonce
	}
	// Ensure that the mix digest is zero as we don't have fork protection currently
	if header.MixDigest != types.BFTDigest {
		return errInvalidMixDigest
	}
	// Ensure that the block doesn't contain any uncles which are meaningless in Istanbul
	if header.UncleHash != nilUncleHash {
		return errInvalidUncleHash
	}
	// Ensure that the block's difficulty is meaningful (may not be correct at this point)
	if header.Difficulty == nil || header.Difficulty.Cmp(defaultDifficulty) != 0 {
		return errInvalidDifficulty
	}

	return sb.verifyCascadingFields(chain, header, parents)
}

// verifyCascadingFields verifies all the header fields that are not standalone,
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

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers
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

// VerifyUncles verifies that the given block's uncles conform to the consensus
// rules of a given engine.
func (sb *Backend) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) > 0 {
		return errInvalidUncleHash
	}
	return nil
}

// verifySigner checks whether the signer is in parent's validator set
func (sb *Backend) verifySigner(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	// Verifying the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}

	validators, err := sb.retrieveValidators(header, parents, chain)

	if err != nil {
		return err
	}

	// resolve the authorization key and check against signers
	signer, err := types.Ecrecover(header)
	if err != nil {
		return err
	}

	// Signer should be in the validator set of previous block's extraData.

	for i := range validators {
		if validators[i] == signer {
			return nil
		}
	}
	return errUnauthorized
}

// verifyCommittedSeals checks whether every committed seal is signed by one of the parent's validators
func (sb *Backend) verifyCommittedSeals(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	number := header.Number.Uint64()
	// We don't need to verify committed seals in the genesis block
	if number == 0 {
		return nil
	}

	validatorAddresses, err := sb.retrieveValidators(header, parents, chain)
	if err != nil {
		return err
	}
	validators := validator.NewSet(validatorAddresses, sb.config.GetProposerPolicy())

	extra, err := types.ExtractBFTHeaderExtra(header)
	if err != nil {
		return err
	}
	// The length of Committed seals should be larger than 0
	if len(extra.CommittedSeal) == 0 {
		return types.ErrEmptyCommittedSeals
	}

	// Check whether the committed seals are generated by parent's validators
	validSeal := 0
	proposalSeal := istanbulCore.PrepareCommittedSeal(header.Hash())
	// 1. Get committed seals from current header
	for _, seal := range extra.CommittedSeal {
		// 2. Get the original address by seal and parent block hash
		addr, err := types.GetSignatureAddress(proposalSeal, seal)
		if err != nil {
			sb.logger.Error("not a valid address", "err", err)
			return types.ErrInvalidSignature
		}
		// Every validator can have only one seal. If more than one seals are signed by a
		// validator, the validator cannot be found and errInvalidCommittedSeals is returned.
		if validators.RemoveValidator(addr) {
			validSeal += 1
		} else {
			return types.ErrInvalidCommittedSeals
		}
	}

	// The length of validSeal should be larger than number of faulty node + 1
	if validSeal <= 2*validators.F() {
		return types.ErrInvalidCommittedSeals
	}

	return nil
}

// VerifySeal checks whether the crypto seal on a header is valid according to
// the consensus rules of the given engine.
func (sb *Backend) VerifySeal(chain consensus.ChainReader, header *types.Header) error {
	// get parent header and ensure the signer is in parent's validator set
	number := header.Number.Uint64()
	//fmt.Println("number VerifySeal", number)

	if number == 0 {
		return errUnknownBlock
	}

	// ensure that the difficulty equals to defaultDifficulty
	if header.Difficulty.Cmp(defaultDifficulty) != 0 {
		return errInvalidDifficulty
	}

	err := sb.verifySigner(chain, header, nil)

	return err
}

// Prepare initializes the consensus fields of a block header according to the
// rules of a particular engine. The changes are executed inline.
func (sb *Backend) Prepare(chain consensus.ChainReader, header *types.Header) error {
	// unused fields, force to set to empty
	header.Coinbase = sb.address
	header.Nonce = emptyNonce
	header.MixDigest = types.BFTDigest

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
	fullnodeSet := validator.NewSet(fullnodeAddresses, sb.config.GetProposerPolicy())

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
	extra, err := types.PrepareExtra(header.Extra, fullnodes)
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

func getSmiloBlockReward(blockNum *big.Int) (blockReward *big.Int) {
	blockReward = new(big.Int)
	for maxBlockRange, reward := range smiloTokenMetricsTable {
		if blockNum.Cmp(maxBlockRange) == -1 {
			if reward.Cmp(blockReward) > 0 {
				blockReward = reward
			}
		}
	}
	return blockReward
}

// AccumulateRewards (override from ethash) credits the coinbase of the given block with the mining reward.
// The total reward consists of the static block reward and rewards for  the community.
func AccumulateRewards(communityAddress string, state *state.StateDB, header *types.Header) {
	// add reward based on chain progression
	blockReward := getSmiloBlockReward(header.Number)

	// Accumulate the rewards
	reward := new(big.Int).Set(blockReward)
	emptryAddress := common.Address{}
	if header.Coinbase != emptryAddress {

		log.Info("$$$$$$$$$$$$$$$$$$$$$ AccumulateRewards, block: ", "blockNum", header.Number.Int64(), "BlockReward", blockReward.Int64(), "Coinbase", header.Coinbase.Hex())

		// Accumulate the rewards to community
		if communityAddress != "" {
			rewardForCommunity := new(big.Int).Div(blockReward, big.NewInt(4))
			state.AddBalance(common.HexToAddress(communityAddress), rewardForCommunity, header.Number)
			log.Info("$$$$$$$$$$$$$$$$$$$$$ AccumulateRewards, adding reward to community ", "rewardForCommunity", rewardForCommunity, "communityAddress", communityAddress)
		}
		state.AddBalance(header.Coinbase, reward, header.Number)
	}
}

// Finalize runs any post-transaction state modifications (e.g. block rewards)
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
		log.Error("finalize. after getValidators", "err", err.Error())
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

func (sb *Backend) getValidators(header *types.Header, chain consensus.ChainReader, state *state.StateDB) ([]common.Address, error) {
	var validators []common.Address

	if header.Number.Int64() == 1 {
		log.Info("Autonity Contract Deployer", "Address", chain.Config().AutonityContractConfig.Deployer)

		sb.blockchain.GetAutonityContract().SavedValidatorsRetriever = func(i uint64) (addresses []common.Address, e error) {
			chain := chain
			return sb.retrieveSavedValidators(i, chain)
		}
		contractAddress, err := sb.blockchain.GetAutonityContract().DeployAutonityContract(chain, header, state)
		if err != nil {
			log.Error("Deploy autonity contract error", "error", err)
			return nil, err
		}
		sb.autonityContractAddress = contractAddress
		validators, err = sb.retrieveSavedValidators(1, chain)
		if err != nil {
			return nil, err
		}
	} else {
		if sb.autonityContractAddress == common.HexToAddress("0000000000000000000000000000000000000000") {
			sb.autonityContractAddress = crypto.CreateAddress(chain.Config().AutonityContractConfig.Deployer, 0)
		}
		var err error
		validators, err = sb.blockchain.GetAutonityContract().ContractGetValidators(chain, header, state)
		if err != nil {
			log.Error("ContractGetValidators error", "error", err)
			return nil, err
		}
	}
	return validators, nil
}

// Seal generates a new block for the given input block with the local miner's
// seal place on top.
func (sb *Backend) Seal(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {

	// update the block header timestamp and signature and propose the block to core engine
	header := block.Header()
	number := header.Number.Uint64()

	// Bail out if we're unauthorized to sign a block
	if _, v := sb.Validators(number).GetByAddress(sb.address); v == nil {
		sb.logger.Error("Seal, Bail out if we're unauthorized to sign a block", "addr", sb.address.String())
		return nil, errUnauthorized
	}

	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		sb.logger.Error("Seal, Bail out ErrUnknownAncestor")
		return nil, consensus.ErrUnknownAncestor
	}
	block, err := sb.updateBlock(block)
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

	// post block into Istanbul engine
	go sb.EventMux().Post(istanbul.RequestEvent{
		Proposal: block,
	})

	for {
		select {
		case result := <-sb.commitCh:
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
func (sb *Backend) updateBlock(block *types.Block) (*types.Block, error) {
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
		Namespace: "istanbul",
		Version:   "1.0",
		Service:   &API{chain: chain, istanbul: sb},
		Public:    true,
	}}
}

// Start implements consensus.BFT.Start
func (sb *Backend) Start(_ context.Context, chain consensus.ChainReader, currentBlock func() *types.Block, hasBadBlock func(hash common.Hash) bool) error {
	sb.coreMu.Lock()
	defer sb.coreMu.Unlock()
	if sb.coreStarted {
		return istanbul.ErrStartedEngine
	}

	// clear previous data
	sb.proposedBlockHash = common.Hash{}
	if sb.commitCh != nil {
		close(sb.commitCh)
	}
	sb.commitCh = make(chan *types.Block, 1)

	sb.blockchain = chain.(*core.BlockChain)
	sb.currentBlock = currentBlock
	sb.hasBadBlock = hasBadBlock

	if err := sb.core.Start(); err != nil {
		return err
	}

	sb.coreStarted = true
	return nil
}

// Stop implements consensus.Istanbul.Stop
func (sb *Backend) Stop() error {
	sb.coreMu.Lock()
	defer sb.coreMu.Unlock()
	if !sb.coreStarted {
		return istanbul.ErrStoppedEngine
	}
	if err := sb.core.Stop(); err != nil {
		return err
	}
	sb.coreStarted = false
	return nil
}

// retrieve list of validators for the block at height passed as parameter
func (sb *Backend) retrieveSavedValidators(number uint64, chain consensus.ChainReader) ([]common.Address, error) {

	if number == 0 {
		number = 1
	}

	header := chain.GetHeaderByNumber(number - 1)
	if header == nil {
		sb.logger.Error("Error when chain.GetHeaderByNumber, ", "errUnknownBlock", errUnknownBlock)
		return nil, errUnknownBlock
	}

	istanbulExtra, err := types.ExtractBFTHeaderExtra(header)
	if err != nil {
		sb.logger.Error("Error when ExtractBFTHeaderExtra , ", "errUnknownBlock", errUnknownBlock)
		return nil, err
	}

	return istanbulExtra.Validators, nil

}

// retrieve list of validators for the block header passed as parameter
func (sb *Backend) retrieveValidators(header *types.Header, parents []*types.Header, chain consensus.ChainReader) ([]common.Address, error) {

	var validators []common.Address
	var err error

	/*
		We can't use retrieveSavedValidators if parents are being passed :
		those blocks are not yet saved in the blockchain.
		autonity will stop processing the received blockchain from the moment an error appears.
		See insertChain in blockchain.go
	*/

	if len(parents) > 0 {
		parent := parents[len(parents)-1]
		var istanbulExtra *types.BFTExtra
		istanbulExtra, err = types.ExtractBFTHeaderExtra(parent)
		if err == nil {
			validators = istanbulExtra.Validators
		}
	} else {
		validators, err = sb.retrieveSavedValidators(header.Number.Uint64(), chain)
	}
	return validators, err

}

func (sb *Backend) SealHash(header *types.Header) common.Hash {
	return types.SigHash(header)
}

// Protocol implements consensus.Engine.Protocol
func (sb *Backend) ProtocolOld() consensus.Protocol {
	return consensus.Protocol{
		Name:     "istanbul",
		Versions: []uint{64},
		Lengths:  []uint64{18},
	}
}
