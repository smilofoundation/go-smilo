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
	"crypto/ecdsa"
	"errors"
	"math/big"

	"go-smilo/src/blockchain/smilobft/core/rawdb"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"

	"go-smilo/src/blockchain/smilobft/consensus/sportdao"
	"go-smilo/src/blockchain/smilobft/core"
	"go-smilo/src/blockchain/smilobft/core/types"
	"go-smilo/src/blockchain/smilobft/core/vm"
	"go-smilo/src/blockchain/smilobft/params"
)

const EnodeStub = "enode://d73b857969c86415c0c000371bcebd9ed3cca6c376032b3f65e58e9e2b79276fbc6f59eb1e22fcd6356ab95f42a666f70afd4985933bd8f3e05beb1a2bf8fdde@172.25.0.11:30303"

// in this test, we can set n to 1, and it means we can process Sport and commit a
// block by one node. Otherwise, if n is larger than 1, we have to generate
// other fake events to process Sport.
func newBlockChain(n int) (*core.BlockChain, *Backend, error) {
	genesis, nodeKeys, err := getGenesisAndKeys(n)
	if err != nil {
		return nil, nil, err
	}
	memDB := rawdb.NewMemoryDatabase()
	config := sportdao.DefaultConfig
	// Use the first key as private key
	b := New(config, nodeKeys[0], memDB, genesis.Config, &vm.Config{})
	genesis.MustCommit(memDB)
	blockchain, err := core.NewBlockChain(memDB, nil, genesis.Config, b, vm.Config{}, nil)
	if err != nil {
		return nil, nil, err
	}

	err = b.Start(context.Background(), blockchain, blockchain.CurrentBlock, blockchain.HasBadBlock)
	if err != nil {
		panic(err)
	}

	validators := b.Fullnodes(0)
	if validators.Size() == 0 {
		return nil, nil, errors.New("failed to get validators")
	}
	proposerAddr := validators.GetSpeaker().Address()

	// find speaker key
	for _, key := range nodeKeys {
		addr := crypto.PubkeyToAddress(key.PublicKey)
		if addr.String() == proposerAddr.String() {
			b.privateKey = key
			b.address = addr
		}
	}

	return blockchain, b, nil
}

func getGenesisAndKeys(n int) (*core.Genesis, []*ecdsa.PrivateKey, error) {
	// Setup fullnodes
	var nodeKeys = make([]*ecdsa.PrivateKey, n)
	var addrs = make([]common.Address, n)
	for i := 0; i < n; i++ {
		nodeKeys[i], _ = crypto.GenerateKey()
		addrs[i] = crypto.PubkeyToAddress(nodeKeys[i].PublicKey)
	}

	// generate genesis block
	genesis := core.DefaultGenesisBlock()
	genesis.Config = params.TestChainConfig
	// force enable SportDAO engine
	genesis.Config.SportDAO = &params.SportDAOConfig{
		Epoch:    sportdao.DefaultConfig.Epoch,
		MinFunds: sportdao.DefaultConfig.MinFunds,
	}
	genesis.Config.AutonityContractConfig = &params.AutonityContractGenesis{}

	genesis.Config.Ethash = nil
	genesis.Difficulty = defaultDifficulty
	genesis.Nonce = emptyNonce.Uint64()
	genesis.Mixhash = types.BFTDigest

	appendFullnodes(genesis, addrs)
	err := genesis.Config.AutonityContractConfig.Prepare("")
	if err != nil {
		return nil, nil, err
	}

	return genesis, nodeKeys, nil
}

func appendFullnodes(genesis *core.Genesis, addrs []common.Address) {

	if len(genesis.ExtraData) < types.BFTExtraVanity {
		genesis.ExtraData = append(genesis.ExtraData, bytes.Repeat([]byte{0x00}, types.BFTExtraVanity)...)
	}
	genesis.ExtraData = genesis.ExtraData[:types.BFTExtraVanity]

	ist := &types.SportExtra{
		Fullnodes:     addrs,
		Seal:          []byte{},
		CommittedSeal: [][]byte{},
	}

	sportDAOPayload, err := rlp.EncodeToBytes(&ist)
	if err != nil {
		panic("failed to encode sportDAO extra")
	}
	genesis.ExtraData = append(genesis.ExtraData, sportDAOPayload...)

	for i := range addrs {
		genesis.Config.AutonityContractConfig.Users = append(
			genesis.Config.AutonityContractConfig.Users,
			params.User{
				Address: &addrs[i],
				Type:    params.UserValidator,
				Enode:   EnodeStub,
				Stake:   100,
			})
	}
}

func makeHeader(parent *types.Block, config *sportdao.Config) *types.Header {
	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     parent.Number().Add(parent.Number(), common.Big1),
		GasLimit:   core.CalcGasLimit(parent, 8000000, 8000000),
		GasUsed:    0,
		Extra:      parent.Extra(),
		Time:       new(big.Int).Add(new(big.Int).SetUint64(parent.Time()), new(big.Int).SetUint64(config.BlockPeriod)).Uint64(),
		Difficulty: defaultDifficulty,
	}
	return header
}

func makeBlock(chain *core.BlockChain, engine *Backend, parent *types.Block) (*types.Block, error) {
	block := makeBlockWithoutSeal(chain, engine, parent)
	block, err := engine.Seal(chain, block, nil)
	return block, err
}

func makeBlockWithoutSeal(chain *core.BlockChain, engine *Backend, parent *types.Block) *types.Block {
	header := makeHeader(parent, engine.config)
	engine.Prepare(chain, header)
	state, _, _ := chain.StateAt(parent.Root())
	block, _ := engine.Finalize(chain, header, state, nil, nil, nil)

	// Write state changes to db
	root, err := state.Commit(chain.Config().IsEIP158(block.Header().Number))
	if err != nil {
		return nil
	}
	if err := state.Database().TrieDB().Commit(root, false); err != nil {
		return nil
	}

	return block
}
