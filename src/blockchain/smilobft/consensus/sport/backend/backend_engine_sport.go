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
	"bytes"
	"context"
	"github.com/ethereum/go-ethereum/crypto"
	"go-smilo/src/blockchain/smilobft/consensus/sport/fullnode"
	"go-smilo/src/blockchain/smilobft/core"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"go-smilo/src/blockchain/smilobft/consensus"
	"go-smilo/src/blockchain/smilobft/consensus/sport"
	"go-smilo/src/blockchain/smilobft/consensus/sport/smilobftcore"
	"go-smilo/src/blockchain/smilobft/core/state"
	"go-smilo/src/blockchain/smilobft/core/types"
)

// SetProposedBlockHash will set the proposed hash into the backend
func (sb *backend) SetProposedBlockHash(hash common.Hash) {
	sb.proposedBlockHash = hash
}


// verifySigner checks whether the signer is in parent's fullnode set
func (sb *backend) verifySigner(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	// Verifying the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}

	fullnodes, err := sb.retrieveValidators(header, parents, chain)
	if err != nil {
		return err
	}

	// resolve the authorization key and check against signers
	signer, err := types.Ecrecover(header)
	if err != nil {
		return err
	}

	// Signer should be in the fullnode set of previous block's extraData.
	for i := range fullnodes {
		if fullnodes[i] == signer {
			return nil
		}
	}
	return errUnauthorized
}

// verifyCommittedSeals checks whether every committed seal is signed by one of the parent's fullnodes
func (sb *backend) verifyCommittedSeals(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	number := header.Number.Uint64()
	// We don't need to verify committed seals in the genesis block
	if number == 0 {
		return nil
	}

	fullnodeAddresses, err := sb.retrieveValidators(header, parents, chain)
	if err != nil {
		return err
	}
	fullnodes := fullnode.NewSet(fullnodeAddresses, sb.config.GetProposerPolicy())

	extra, err := types.ExtractSportExtra(header)
	if err != nil {
		return err
	}
	// The length of Committed seals should be larger than 0
	if len(extra.CommittedSeal) == 0 {
		return types.ErrEmptyCommittedSeals
	}

	// Check whether the committed seals are generated by parent's fullnodes
	validSeal := 0
	proposalSeal := smilobftcore.PrepareCommittedSeal(header.Hash())
	// 1. Get committed seals from current header
	for _, seal := range extra.CommittedSeal {
		// 2. Get the original address by seal and parent block hash
		addr, err := types.GetSignatureAddress(proposalSeal, seal)
		if err != nil {
			sb.logger.Error("not a valid address", "err", err)
			return types.ErrInvalidSignature
		}
		// Every fullnode can have only one seal. If more than one seals are signed by a
		// fullnode, the fullnode cannot be found and errInvalidCommittedSeals is returned.
		if fullnodes.RemoveFullnode(addr) {
			validSeal += 1
		} else {
			return types.ErrInvalidCommittedSeals
		}
	}

	// The length of validSeal should be larger than number of faulty node + 1
	minApprovers := fullnodes.MinApprovers()
	if validSeal < minApprovers {
		//if actual block is bigger than planned fork OR (old scenario) old expression that leads to less confirmation
		if header.Number.Cmp(chain.Config().SixtySixPercentBlock) > 0 || validSeal < minApprovers-1 {
			sb.logger.Error("The length of validSeal should be larger or eq than number of 2x faulty nodes", "SixtySixPercentBlock", chain.Config().SixtySixPercentBlock, "validSeal", validSeal, "MinApprovers", minApprovers, "original", 2*fullnodes.Size(), "F", fullnodes.Size())
			return errInvalidCommittedSeals
		}
	}

	return nil
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

// update timestamp and signature of the block based on its number of transactions
func (sb *backend) updateBlock(block *types.Block) (*types.Block, error) {
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

// Start implements consensus.Sport.Start
func (sb *backend) Start(_ context.Context, chain consensus.ChainReader, currentBlock func() *types.Block, hasBadBlock func(hash common.Hash) bool) error {
	sb.coreMu.Lock()
	defer sb.coreMu.Unlock()
	if sb.coreStarted {
		return consensus.ErrStartedEngine
	}

	// clear previous data
	sb.proposedBlockHash = common.Hash{}
	if sb.commitChBlock != nil {
		close(sb.commitChBlock)
	}
	sb.commitChBlock = make(chan *types.Block, 1)

	sb.blockchain = chain.(*core.BlockChain)

	sb.chain = chain
	sb.currentBlock = currentBlock
	sb.hasBadBlock = hasBadBlock

	if err := sb.core.Start(); err != nil {
		return err
	}

	sb.coreStarted = true
	return nil
}

// Stop implements consensus.Sport.Stop
func (sb *backend) Stop() error {
	sb.coreMu.Lock()
	defer sb.coreMu.Unlock()
	if !sb.coreStarted {
		return sport.ErrStoppedEngine
	}
	if err := sb.core.Stop(); err != nil {
		return err
	}
	sb.coreStarted = false
	return nil
}

// prepareExtra returns a extra-data of the given header and fullnodes
func prepareExtra(header *types.Header, vals []common.Address) ([]byte, error) {
	var buf bytes.Buffer

	// compensate the lack bytes if header.Extra is not enough SportExtraVanity bytes.
	if len(header.Extra) < types.SportExtraVanity {
		header.Extra = append(header.Extra, bytes.Repeat([]byte{0x00}, types.SportExtraVanity-len(header.Extra))...)
	}
	buf.Write(header.Extra[:types.SportExtraVanity])

	ist := &types.SportExtra{
		Fullnodes:     vals,
		Seal:          []byte{},
		CommittedSeal: [][]byte{},
	}

	payload, err := rlp.EncodeToBytes(&ist)
	if err != nil {
		return nil, err
	}

	return append(buf.Bytes(), payload...), nil
}

// retrieve list of validators for the block at height passed as parameter
func (sb *backend) retrieveSavedValidators(number uint64, chain consensus.ChainReader) ([]common.Address, error) {
	if number == 0 {
		number = 1
	}

	header := chain.GetHeaderByNumber(number - 1)
	if header == nil {
		sb.logger.Error("Error when chain.GetHeaderByNumber, ", "errUnknownBlock", errUnknownBlock)
		return nil, errUnknownBlock
	}

	sportExtra, err := types.ExtractSportExtra(header)
	if err != nil {
		sb.logger.Error("Error when ExtractBFTHeaderExtra , ", "errUnknownBlock", errUnknownBlock)
		return nil, err
	}

	return sportExtra.Fullnodes, nil

}

// retrieve list of validators for the block header passed as parameter
func (sb *backend) retrieveValidators(header *types.Header, parents []*types.Header, chain consensus.ChainReader) ([]common.Address, error) {
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
		var sportExtra *types.SportExtra
		sportExtra, err = types.ExtractSportExtra(parent)
		if err == nil {
			validators = sportExtra.Fullnodes
		}
	} else {
		validators, err = sb.retrieveSavedValidators(header.Number.Uint64(), chain)
	}
	return validators, err

}



// writeSeal writes the extra-data field of the given header with the given seals.
// suggest to rename to writeSeal.
func writeSeal(h *types.Header, seal []byte) error {
	if len(seal)%types.BFTExtraSeal != 0 {
		return errInvalidSignature
	}

	sportExtra, err := types.ExtractSportExtra(h)
	if err != nil {
		return err
	}

	sportExtra.Seal = seal
	payload, err := rlp.EncodeToBytes(&sportExtra)
	if err != nil {
		return err
	}

	h.Extra = append(h.Extra[:types.SportExtraVanity], payload...)
	return nil
}

// writeCommittedSeals writes the extra-data field of a block header with given committed seals.
func writeCommittedSeals(h *types.Header, committedSeals [][]byte) error {
	if len(committedSeals) == 0 {
		return errInvalidCommittedSeals
	}

	for _, seal := range committedSeals {
		if len(seal) != types.BFTExtraSeal {
			return errInvalidCommittedSeals
		}
	}

	sportExtra, err := types.ExtractSportExtra(h)
	if err != nil {
		return err
	}

	sportExtra.CommittedSeal = make([][]byte, len(committedSeals))
	copy(sportExtra.CommittedSeal, committedSeals)

	payload, err := rlp.EncodeToBytes(&sportExtra)
	if err != nil {
		return err
	}

	h.Extra = append(h.Extra[:types.SportExtraVanity], payload...)
	return nil
}


func (sb *backend) getValidators(header *types.Header, chain consensus.ChainReader, state *state.StateDB) ([]common.Address, error) {
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
