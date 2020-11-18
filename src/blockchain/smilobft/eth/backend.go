// Copyright 2014 The go-ethereum Authors
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

// Package eth implements the Smilo protocol.
package eth

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/event"

	"go-smilo/src/blockchain/smilobft/accounts/abi/bind"
	"go-smilo/src/blockchain/smilobft/cmn"
	"go-smilo/src/blockchain/smilobft/consensus/istanbul"
	istanbulBackend "go-smilo/src/blockchain/smilobft/consensus/istanbul/backend"
	"go-smilo/src/blockchain/smilobft/consensus/sportdao"
	tendermintBackend "go-smilo/src/blockchain/smilobft/consensus/tendermint/backend"
	tendermintCore "go-smilo/src/blockchain/smilobft/consensus/tendermint/core"
	"go-smilo/src/blockchain/smilobft/p2p/enode"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"go-smilo/src/blockchain/smilobft/rpc"

	"go-smilo/src/blockchain/smilobft/accounts"
	"go-smilo/src/blockchain/smilobft/consensus"
	"go-smilo/src/blockchain/smilobft/consensus/clique"
	"go-smilo/src/blockchain/smilobft/consensus/ethash"
	"go-smilo/src/blockchain/smilobft/consensus/sport"
	smiloBackend "go-smilo/src/blockchain/smilobft/consensus/sport/backend"
	smiloDAOBackend "go-smilo/src/blockchain/smilobft/consensus/sportdao/backend"
	"go-smilo/src/blockchain/smilobft/core"
	"go-smilo/src/blockchain/smilobft/core/bloombits"
	"go-smilo/src/blockchain/smilobft/core/rawdb"
	"go-smilo/src/blockchain/smilobft/core/types"
	"go-smilo/src/blockchain/smilobft/core/vm"
	"go-smilo/src/blockchain/smilobft/eth/downloader"
	"go-smilo/src/blockchain/smilobft/eth/filters"
	"go-smilo/src/blockchain/smilobft/eth/gasprice"
	"go-smilo/src/blockchain/smilobft/ethdb"
	"go-smilo/src/blockchain/smilobft/internal/ethapi"
	"go-smilo/src/blockchain/smilobft/miner"
	"go-smilo/src/blockchain/smilobft/node"
	"go-smilo/src/blockchain/smilobft/p2p"
	"go-smilo/src/blockchain/smilobft/params"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	APIs() []rpc.API
	Protocols() []p2p.Protocol
	SetBloomBitsIndexer(bbIndexer *core.ChainIndexer)
	SetContractBackend(bind.ContractBackend)
}

// Smilo implements the Smilo full node service.
type Smilo struct {
	config      *Config
	chainConfig *params.ChainConfig

	// Channel for shutting down the service
	shutdownChan chan bool

	server *p2p.Server

	// Handlers
	txPool          *core.TxPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager
	lesServer       LesServer

	// DB interfaces
	chainDb ethdb.Database // Block chain database

	eventMux       *cmn.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	APIBackend *EthAPIBackend

	miner     *miner.Miner
	gasPrice  *big.Int
	etherbase common.Address

	networkID     uint64
	netRPCService *ethapi.PublicNetAPI

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and etherbase)
	//protocol Protocol

	glienickeCh  chan core.WhitelistEvent
	glienickeSub event.Subscription
}

func (s *Smilo) ChainConfig() *params.ChainConfig {
	return s.chainConfig
}

func (s *Smilo) AddLesServer(ls LesServer) {
	s.lesServer = ls
	ls.SetBloomBitsIndexer(s.bloomIndexer)
}

// SetClient sets a rpc client which connecting to our local node.
func (s *Smilo) SetContractBackend(backend bind.ContractBackend) {
	// Pass the rpc client to les server if it is enabled.
	if s.lesServer != nil {
		s.lesServer.SetContractBackend(backend)
	}
}

// New creates a new Smilo object (including the
// initialisation of the common Smilo object)
func New(ctx *node.ServiceContext, config *Config, cons func(basic consensus.Engine) consensus.Engine) (*Smilo, error) {
	log.Info("$$$$$$$ Going to creates a new Smilo backend config object ")
	// Ensure configuration values are compatible and sane
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run eth.Smilo in light sync mode, use les.LightEthereum")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
	if config.Miner.GasPrice == nil || config.Miner.GasPrice.Cmp(common.Big0) <= 0 {
		log.Warn("Sanitizing invalid miner gas price", "provided", config.Miner.GasPrice, "updated", DefaultConfig.Miner.GasPrice)
		config.Miner.GasPrice = new(big.Int).Set(DefaultConfig.Miner.GasPrice)
	}
	if config.NoPruning && config.TrieDirtyCache > 0 {
		config.TrieCleanCache += config.TrieDirtyCache
		config.TrieDirtyCache = 0
	}
	log.Info("Allocated trie memory caches", "clean", common.StorageSize(config.TrieCleanCache)*1024*1024, "dirty", common.StorageSize(config.TrieDirtyCache)*1024*1024)

	// Assemble the Smilo object
	chainDb, err := ctx.OpenDatabaseWithFreezer("chaindata", config.DatabaseCache, config.DatabaseHandles, config.DatabaseFreezer, "eth/db/chaindata/")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlockWithOverride(chainDb, config.Genesis, config.OverrideIstanbul)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig, "config", config)

	// changes to manipulate the chain id for migration from 2.0.2 and below version to 2.0.3
	// version of Quorum  - this is applicable for v2.0.3 onwards
	if chainConfig.IsSmilo {
		if (chainConfig.ChainID != nil && chainConfig.ChainID.Int64() == 1) || config.NetworkId == 1 {
			return nil, errors.New("cannot have chain id or network id as 1")
		}
	}

	if !core.GetIsSmiloEIP155Activated(chainDb) && chainConfig.ChainID != nil {
		//Upon starting the node, write the flag to disallow changing ChainID/EIP155 block after HF
		core.WriteSmiloEIP155Activation(chainDb)
	}

	var (
		vmConfig = vm.Config{
			EnablePreimageRecording: config.EnablePreimageRecording,
			EWASMInterpreter:        config.EWASMInterpreter,
			EVMInterpreter:          config.EVMInterpreter,
		}
		cacheConfig = &core.CacheConfig{
			TrieCleanLimit: config.TrieCleanCache,
			TrieDirtyLimit: config.TrieDirtyCache,
			TrieTimeLimit:  config.TrieTimeout,
		}
	)

	consEngine := CreateConsensusEngine(ctx, chainConfig, config, config.Miner.Notify, config.Miner.Noverify, chainDb, &vmConfig)
	if cons != nil {
		consEngine = cons(consEngine)
	}

	eth := &Smilo{
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         consEngine,
		shutdownChan:   make(chan bool),
		networkID:      config.NetworkId,
		gasPrice:       config.Miner.GasPrice,
		etherbase:      config.Miner.Etherbase,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   NewBloomIndexer(chainDb, params.BloomBitsBlocks, params.BloomConfirms),
		glienickeCh:    make(chan core.WhitelistEvent),
	}

	//// Quorum: Set protocol Name/Version
	//if chainConfig.IsSmilo {
	//	quorumProtocol := eth.engine.Protocol()
	//	protocolName = quorumProtocol.Name
	//	ProtocolVersions = quorumProtocol.Versions
	//}

	// force to set the etherbase to node key address
	if chainConfig.Istanbul != nil || chainConfig.SportDAO != nil || chainConfig.Tendermint != nil || chainConfig.Sport != nil {
		log.Info("force to set the etherbase to node key address")
		eth.etherbase = crypto.PubkeyToAddress(ctx.NodeKey().PublicKey)
	}

	//if h, ok := eth.engine.(consensus.Handler); ok {
	//	protocolName, extraMsgCodes := h.Protocol()
	//	eth.protocol.Name = protocolName
	//	eth.protocol.Versions = ProtocolVersions
	//	eth.protocol.Lengths = make([]uint64, len(protocolLengths))
	//	for i := range eth.protocol.Lengths {
	//		eth.protocol.Lengths[i] = protocolLengths[uint(i)] + extraMsgCodes
	//	}
	//} else {
	//	eth.protocol = EthDefaultProtocol
	//}

	bcVersion := rawdb.ReadDatabaseVersion(chainDb)
	var dbVer = "<nil>"
	if bcVersion != nil {
		dbVer = fmt.Sprintf("%d", *bcVersion)
	}
	log.Info("$$$$$$$$$$$$ Initialising protocol", "versions", ProtocolVersions, "network", config.NetworkId, "dbversion", dbVer)

	if !config.SkipBcVersionCheck {
		if bcVersion != nil && *bcVersion > core.BlockChainVersion {
			return nil, fmt.Errorf("database version is v%d, Geth %s only supports v%d", *bcVersion, params.VersionWithMeta, core.BlockChainVersion)
		} else if bcVersion == nil || *bcVersion < core.BlockChainVersion {
			log.Warn("Upgrade blockchain database version", "from", dbVer, "to", core.BlockChainVersion)
			rawdb.WriteDatabaseVersion(chainDb, core.BlockChainVersion)
		}
	}

	eth.blockchain, err = core.NewBlockChain(chainDb, cacheConfig, eth.chainConfig, eth.engine, vmConfig, eth.shouldPreserve)
	if err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		eth.blockchain.SetHead(compat.RewindTo)
		rawdb.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	eth.bloomIndexer.Start(eth.blockchain)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	if config.TxPool.Blacklist != "" {
		config.TxPool.Blacklist = ctx.ResolvePath(config.TxPool.Blacklist)
	}

	eth.txPool = core.NewTxPool(config.TxPool, chainConfig, eth.blockchain)

	// Permit the downloader to use the trie cache allowance during fast sync
	cacheLimit := cacheConfig.TrieCleanLimit + cacheConfig.TrieDirtyLimit
	checkpoint := config.Checkpoint
	if checkpoint == nil {
		checkpoint = params.TrustedCheckpoints[genesisHash]
	}
	if eth.protocolManager, err = NewProtocolManager(chainConfig, checkpoint, config.SyncMode, config.NetworkId, eth.eventMux, eth.txPool, eth.engine, eth.blockchain, chainDb, cacheLimit, config.Whitelist, config.EnableNodePermissionFlag); err != nil {
		return nil, err
	}
	var MinBlocksEmptyMining = big.NewInt(20000000)
	if (chainConfig.Sport != nil || chainConfig.Istanbul != nil ||
		chainConfig.SportDAO != nil || chainConfig.Tendermint != nil) && config.Sport.MinBlocksEmptyMining != nil {
		MinBlocksEmptyMining = config.Sport.MinBlocksEmptyMining
		log.Warn("Will set MinBlocksEmptyMining", "MinBlocksEmptyMining", MinBlocksEmptyMining)
	}

	eth.miner = miner.New(eth, &config.Miner, chainConfig, eth.EventMux(), eth.engine, eth.isLocalBlock, MinBlocksEmptyMining)
	log.Info("$$$$$$ Prepare extradata", "Miner", config.Miner, "chainConfig", eth.chainConfig)
	extradata := makeExtraData(config.Miner.ExtraData, eth.chainConfig.IsSmilo)
	log.Info("$$$$$$ makeExtraData", "extradata", cmn.HexToHash(string(extradata)))
	err = eth.miner.SetExtra(extradata)
	if err != nil {
		log.Error("Could not set Extra on miner, WTF ? ")
		return nil, err
	}

	hexNodeId := fmt.Sprintf("%x", crypto.FromECDSAPub(&ctx.NodeKey().PublicKey)[1:]) // Quorum
	eth.APIBackend = &EthAPIBackend{ctx.ExtRPCEnabled(), eth, nil, hexNodeId}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.Miner.GasPrice
	}
	eth.APIBackend.gpo = gasprice.NewOracle(eth.APIBackend, gpoParams)

	return eth, nil
}

func makeExtraData(extra []byte, isSmilo bool) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"geth",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.GetMaximumExtraDataSize(isSmilo) {
		log.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", params.GetMaximumExtraDataSize(isSmilo))
		extra = nil
	}
	return extra
}

// CreateConsensusEngine creates the required type of consensus engine instance for an Smilo service
func CreateConsensusEngine(ctx *node.ServiceContext, chainConfig *params.ChainConfig, config *Config, notify []string, noverify bool, db ethdb.Database, vmConfig *vm.Config) consensus.Engine {
	log.Info("****************** Going to create the required type of consensus engine instance for an Smilo service !!!!!!!!!!!!!!!!!!!")

	// If proof-of-authority is requested, set it up
	if chainConfig.Clique != nil {
		chainConfig.Clique.AllowedFutureBlockTime = config.AllowedFutureBlockTime //Quorum
		log.Warn("$$$ Clique is requested, set it up", "chainConfig", chainConfig)
		return clique.New(chainConfig.Clique, db)
	}
	// If Sport is requested, set it up
	if chainConfig.Sport != nil {
		log.Warn("$$$ Sport Consensus activated, will set it up", "chainConfig.Sport", chainConfig.Sport, "chainConfig", chainConfig)
		if chainConfig.Sport.Epoch != 0 {
			config.Sport.Epoch = chainConfig.Sport.Epoch
		}
		config.Sport.SpeakerPolicy = sport.SpeakerPolicy(chainConfig.Sport.SpeakerPolicy)

		if chainConfig.Sport.MinFunds != 0 {
			config.Sport.MinFunds = chainConfig.Sport.MinFunds
		}

		return smiloBackend.New(&config.Sport, ctx.NodeKey(), db)
	}
	if chainConfig.SportDAO != nil {
		if chainConfig.SportDAO.Epoch != 0 {
			config.SportDAO.Epoch = chainConfig.SportDAO.Epoch
		}
		config.SportDAO.SpeakerPolicy = sportdao.SpeakerPolicy(chainConfig.SportDAO.SpeakerPolicy)
		if chainConfig.SportDAO.MinFunds != 0 {
			config.SportDAO.MinFunds = chainConfig.SportDAO.MinFunds
		}
		if config.SportDAO.MaxTimeout == 0 {
			config.SportDAO.MaxTimeout = sportdao.DefaultConfig.MaxTimeout
		}
		if chainConfig.SportDAO.MinFunds == 0 {
			config.SportDAO.MinFunds = sportdao.DefaultConfig.MinFunds
		}
		if chainConfig.SportDAO.MinFunds == 0 {
			config.SportDAO.MinFunds = sportdao.DefaultConfig.MinFunds
		}

		log.Warn("$$$ SportDAO Consensus activated, will set it up", "&config.SportDAO", &config.SportDAO, "chainConfig", chainConfig)
		return smiloDAOBackend.New(&config.SportDAO, ctx.NodeKey(), db, chainConfig, vmConfig)
	}

	if chainConfig.Istanbul != nil {
		if config.Istanbul.MaxTimeout == 0 {
			config.Istanbul.MaxTimeout = istanbul.DefaultConfig.MaxTimeout
		}
		log.Warn("$$$ Istanbul Consensus activated, will set it up", "chainConfig.Istanbul", chainConfig.Istanbul, "chainConfig", chainConfig)
		return istanbulBackend.New(&config.Istanbul, ctx.NodeKey(), db, chainConfig, vmConfig)
	}
	if chainConfig.Tendermint != nil {
		log.Warn("$$$ Tendermint Consensus activated, will set it up", "chainConfig.Tendermint", chainConfig.Tendermint, "chainConfig", chainConfig)
		back := tendermintBackend.New(&config.Tendermint, ctx.NodeKey(), db, chainConfig, vmConfig)
		return tendermintCore.New(back, &config.Tendermint)
	}

	// Otherwise assume proof-of-work
	log.Warn("$$$ Assume proof-of-work consensus to be selected, ", "config.PowMode", config.PowMode, "chainConfig", chainConfig)
	switch config.PowMode {
	case ModeFake:
		log.Warn("Ethash used in fake mode")
		return ethash.NewFaker()
	case ModeTest:
		log.Warn("Ethash used in test mode")
		return ethash.NewTester()
	case ModeShared:
		log.Warn("Ethash used in shared mode")
		return ethash.NewShared()
	default:
		log.Warn("Ethash used in full fake mode")
		return ethash.NewFullFaker()
	}
}

// APIs return the collection of RPC services the ethereum package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *Smilo) APIs() []rpc.API {
	apis := ethapi.GetAPIs(s.APIBackend)

	// Append any APIs exposed explicitly by the les server
	if s.lesServer != nil {
		apis = append(apis, s.lesServer.APIs()...)
	}
	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append any APIs exposed explicitly by the les server
	if s.lesServer != nil {
		apis = append(apis, s.lesServer.APIs()...)
	}

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "eth",
			Version:   "1.0",
			Service:   NewPublicEthereumAPI(s),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.APIBackend, false),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(s),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s.chainConfig, s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *Smilo) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *Smilo) Etherbase() (eb common.Address, err error) {
	s.lock.RLock()
	etherbase := s.etherbase
	s.lock.RUnlock()

	if etherbase != (common.Address{}) {
		return etherbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			etherbase := accounts[0].Address

			s.lock.Lock()
			s.etherbase = etherbase
			s.lock.Unlock()

			log.Info("Etherbase automatically configured", "address", etherbase)
			return etherbase, nil
		}
	}
	return common.Address{}, fmt.Errorf("etherbase must be explicitly specified")
}

// isLocalBlock checks whether the specified block is mined
// by local miner accounts.
//
// We regard two types of accounts as local miner account: etherbase
// and accounts specified via `txpool.locals` flag.
func (s *Smilo) isLocalBlock(block *types.Block) bool {
	author, err := s.engine.Author(block.Header())
	if err != nil {
		log.Warn("Failed to retrieve block author", "number", block.NumberU64(), "hash", block.Hash(), "err", err)
		return false
	}
	// Check whether the given address is etherbase.
	s.lock.RLock()
	etherbase := s.etherbase
	s.lock.RUnlock()
	if author == etherbase {
		return true
	}
	// Check whether the given address is specified by `txpool.local`
	// CLI flag.
	for _, account := range s.config.TxPool.Locals {
		if account == author {
			return true
		}
	}
	return false
}

// shouldPreserve checks whether we should preserve the given block
// during the chain reorg depending on whether the author of block
// is a local account.
func (s *Smilo) shouldPreserve(block *types.Block) bool {
	// The reason we need to disable the self-reorg preserving for clique
	// is it can be probable to introduce a deadlock.
	//
	// e.g. If there are 7 available signers
	//
	// r1   A
	// r2     B
	// r3       C
	// r4         D
	// r5   A      [X] F G
	// r6    [X]
	//
	// In the round5, the inturn signer E is offline, so the worst case
	// is A, F and G sign the block of round5 and reject the block of opponents
	// and in the round6, the last available signer B is offline, the whole
	// network is stuck.
	if _, ok := s.engine.(*clique.Clique); ok {
		return false
	}

	//if _, ok := s.engine.(*consensus.BFT); ok {
	//	return false
	//}

	return s.isLocalBlock(block)
}

// SetEtherbase sets the mining reward address.
func (s *Smilo) SetEtherbase(etherbase common.Address) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.engine.(consensus.BFT); ok {
		log.Error("Cannot set etherbase in BFT consensus")
		return
	}
	s.etherbase = etherbase

	s.miner.SetEtherbase(etherbase)
}

// StartMining starts the miner with the given number of CPU threads. If mining
// is already running, this method adjust the number of threads allowed to use
// and updates the minimum price required by the transaction pool.
func (s *Smilo) StartMining(threads int) error {
	log.Warn("mining", "n", threads)
	// Update the thread count within the consensus engine
	type threaded interface {
		SetThreads(threads int)
	}
	if th, ok := s.engine.(threaded); ok {
		log.Info("Updated mining threads", "threads", threads)
		if threads == 0 {
			threads = -1 // Disable the miner from within
		}
		th.SetThreads(threads)
	}
	// If the miner was not running, initialize it
	if !s.IsMining() {

		log.Info("Miner is not running, initializing it ...", "threads", threads)
		// Propagate the initial price point to the transaction pool
		s.lock.RLock()
		price := s.gasPrice
		s.lock.RUnlock()
		s.txPool.SetGasPrice(price)

		// Configure the local mining address
		eb, err := s.Etherbase()
		if err != nil {
			log.Error("Cannot start mining without etherbase", "err", err)
			return fmt.Errorf("etherbase missing: %v", err)
		}
		if clique, ok := s.engine.(*clique.Clique); ok {
			wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
			if wallet == nil || err != nil {
				log.Error("Etherbase account unavailable locally", "err", err)
				return fmt.Errorf("signer missing: %v", err)
			}
			clique.Authorize(eb, wallet.SignData)
		}
		// If mining is started, we can disable the transaction rejection mechanism
		// introduced to speed sync times.
		atomic.StoreUint32(&s.protocolManager.acceptTxs, 1)

		go s.miner.Start(eb)
	}
	return nil
}

// StopMining terminates the miner, both at the consensus engine level as well as
// at the block creation level.
func (s *Smilo) StopMining() {
	// Update the thread count within the consensus engine
	type threaded interface {
		SetThreads(threads int)
	}
	if th, ok := s.engine.(threaded); ok {
		th.SetThreads(-1)
	}
	// Stop the block creating itself
	s.miner.Stop()
}

func (s *Smilo) IsMining() bool      { return s.miner.Mining() }
func (s *Smilo) Miner() *miner.Miner { return s.miner }

func (s *Smilo) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *Smilo) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *Smilo) TxPool() *core.TxPool               { return s.txPool }
func (s *Smilo) EventMux() *cmn.TypeMux             { return s.eventMux }
func (s *Smilo) Engine() consensus.Engine           { return s.engine }
func (s *Smilo) ChainDb() ethdb.Database            { return s.chainDb }
func (s *Smilo) IsListening() bool                  { return true } // Always listening
func (s *Smilo) EthVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *Smilo) NetVersion() uint64                 { return s.networkID }
func (s *Smilo) Downloader() *downloader.Downloader { return s.protocolManager.downloader }
func (s *Smilo) Synced() bool                       { return atomic.LoadUint32(&s.protocolManager.acceptTxs) == 1 }
func (s *Smilo) ArchiveMode() bool                  { return s.config.NoPruning }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *Smilo) Protocols() []p2p.Protocol {
	var protos []p2p.Protocol
	//protos := make([]p2p.Protocol, len(ProtocolVersions))
	//
	//for i, vsn := range ProtocolVersions {
	//	protos[i] = s.protocolManager.makeProtocol(vsn)
	//	protos[i].Attributes = []enr.Entry{s.currentEthEntry()}
	//}
	//if s.lesServer != nil {
	//	protos = append(protos, s.lesServer.Protocols()...)
	//}

	protos = append(protos, s.protocolManager.SubProtocols...)

	return protos
}

// Start implements node.Service, starting all internal goroutines needed by the
// Smilo protocol implementation.
func (s *Smilo) Start(srvr *p2p.Server) error {

	if (s.blockchain.Config().Tendermint != nil || s.blockchain.Config().Istanbul != nil || s.blockchain.Config().SportDAO != nil) && s.blockchain.Config().AutonityContractConfig != nil {
		//if srvr.EnableNodePermissionFlag {
		// Subscribe to Autonity updates events
		log.Warn("eth/backend.go, Start(), Will Subscribe to Autonity updates events")
		s.glienickeSub = s.blockchain.SubscribeAutonityEvents(s.glienickeCh)
		savedList := rawdb.ReadEnodeWhitelist(s.chainDb, srvr.EnableNodePermissionFlag)
		log.Info("eth/backend.go, Start(), Reading Whitelist", "list", savedList.StrList)
		go s.glienickeEventLoop(srvr)
		srvr.UpdateWhitelist(savedList.List)
	} else {
		log.Warn("eth/backend.go, Start(), EnableNodePermissionFlag false, will not Subscribe to Autonity updates events")
	}
	s.startEthEntryUpdate(srvr.LocalNode())

	// Start the bloom bits servicing goroutines
	s.startBloomHandlers(params.BloomBitsBlocks)

	// Start the RPC service
	s.netRPCService = ethapi.NewPublicNetAPI(srvr, s.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers
	if s.config.LightServ > 0 {
		if s.config.LightPeers >= srvr.MaxPeers {
			return fmt.Errorf("invalid peer config: light peer count (%d) >= total peer count (%d)", s.config.LightPeers, srvr.MaxPeers)
		}
		maxPeers -= s.config.LightPeers
	}
	// Start the networking layer and the light server if requested
	s.protocolManager.Start(maxPeers)
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	return nil
}

// Whitelist updating loop. Act as a relay between state processing logic and DevP2P
// for updating the list of authorized enodes
func (s *Smilo) glienickeEventLoop(server *p2p.Server) {
	for {
		select {
		case event := <-s.glienickeCh:
			whitelist := append([]*enode.Node{}, event.Whitelist...)
			// Filter the list of need to be dropped peers depending on TD.
			for _, connectedPeer := range s.protocolManager.peers.Peers() {
				found := false
				connectedEnode := connectedPeer.Node()
				for _, whitelistedEnode := range whitelist {
					if connectedEnode.ID() == whitelistedEnode.ID() {
						found = true
						break
					}
				}

				if !found { // this node is no longer in the whitelist
					peerID := fmt.Sprintf("%x", connectedEnode.ID().Bytes()[:8])
					peer := s.protocolManager.peers.Peer(peerID)
					localTd := s.blockchain.CurrentHeader().Number.Uint64() + 1
					if peer != nil && peer.td.Uint64() > localTd {
						whitelist = append(whitelist, connectedEnode)
					}
				}
			}
			server.UpdateWhitelist(whitelist)
		// Err() channel will be closed when unsubscribing.
		case <-s.glienickeSub.Err():
			return
		}
	}
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Smilo protocol.
func (s *Smilo) Stop() error {
	s.bloomIndexer.Close()
	if s.glienickeSub != nil {
		s.glienickeSub.Unsubscribe()
	}
	s.blockchain.Stop()
	s.engine.Close()
	s.protocolManager.Stop()
	if s.lesServer != nil {
		s.lesServer.Stop()
	}
	s.txPool.Stop()
	s.miner.Stop()
	s.eventMux.Stop()

	s.chainDb.Close()
	close(s.shutdownChan)
	return nil
}

func (s *Smilo) CalcGasLimit(block *types.Block) uint64 {
	return core.CalcGasLimit(block, s.config.Miner.GasFloor, s.config.Miner.GasCeil)
}
