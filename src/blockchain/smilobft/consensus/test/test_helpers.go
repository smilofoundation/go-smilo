package test

import (
	"crypto/ecdsa"
	"crypto/rand"
	"io/ioutil"
	"math/big"
	"net"
	"sync"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"

	"go-smilo/src/blockchain/smilobft/consensus"
	"go-smilo/src/blockchain/smilobft/consensus/tendermint/config"
	"go-smilo/src/blockchain/smilobft/core"
	"go-smilo/src/blockchain/smilobft/core/types"
	"go-smilo/src/blockchain/smilobft/eth"
	"go-smilo/src/blockchain/smilobft/eth/downloader"
	"go-smilo/src/blockchain/smilobft/node"
	"go-smilo/src/blockchain/smilobft/p2p"
	"go-smilo/src/blockchain/smilobft/p2p/enode"
	"go-smilo/src/blockchain/smilobft/params"
)

type networkRate struct {
	in  int64
	out int64
}

type testNode struct {
	isRunning      bool
	isInited       bool
	wasStopped     bool //fixme should be removed
	privateKey     *ecdsa.PrivateKey
	address        string
	port           int
	url            string
	listener       net.Listener
	node           *node.Node
	enode          *enode.Node
	service        *eth.Smilo
	eventChan      chan core.ChainEvent
	subscription   event.Subscription
	transactions   map[common.Hash]struct{}
	transactionsMu sync.Mutex
	blocks         map[uint64]block
	lastBlock      uint64
	txsSendCount   *int64
	txsChainCount  map[uint64]int64
}

type block struct {
	hash common.Hash
	txs  int
}

func sendTx(service *eth.Smilo, key *ecdsa.PrivateKey, fromAddr common.Address, toAddr common.Address, transactionGenerator func(nonce uint64, toAddr common.Address, key *ecdsa.PrivateKey) (*types.Transaction, error)) (*types.Transaction, error) {
	nonce := service.TxPool().Nonce(fromAddr)

	var tx *types.Transaction
	var err error

	for stop := 10; stop > 0; stop-- {
		tx, err = transactionGenerator(nonce, toAddr, key)
		if err != nil {
			nonce++
			continue
		}
		err = service.TxPool().AddLocal(tx)
		if err == nil {
			break
		}
		nonce++

	}
	if err != nil {
		return nil, err
	}
	return tx, nil
}

//func sendTx(service *eth.Ethereum, fromValidator *ecdsa.PrivateKey, fromAddr common.Address, toAddr common.Address) (*types.Transaction, error) {
//	nonce := service.TxPool().Nonce(fromAddr)
//
//	tx, err := txWithNonce(fromAddr, nonce, toAddr, fromValidator, service)
//	if err != nil {
//		return txWithNonce(fromAddr, nonce+1, toAddr, fromValidator, service)
//	}
//	return tx, nil
//}

func generateRandomTx(nonce uint64, toAddr common.Address, key *ecdsa.PrivateKey) (*types.Transaction, error) {
	randEth, err := rand.Int(rand.Reader, big.NewInt(10000000))
	if err != nil {
		return nil, err
	}

	return types.SignTx(
		types.NewTransaction(
			nonce,
			toAddr,
			big.NewInt(1),
			210000000,
			big.NewInt(100000000000+int64(randEth.Uint64())),
			nil,
		),
		types.HomesteadSigner{}, key)
}

func makeGenesis(validators []*testNode) *core.Genesis {
	// generate genesis block
	genesis := core.DefaultGenesisBlock()
	genesis.ExtraData = nil
	genesis.GasLimit = math.MaxUint64 - 1
	genesis.GasUsed = 0
	genesis.Difficulty = big.NewInt(1)
	genesis.Timestamp = 0
	genesis.Nonce = 0
	genesis.Mixhash = types.BFTDigest

	genesis.Config = params.TestChainConfig
	genesis.Config.Tendermint = &params.TendermintConfig{}
	genesis.Config.Ethash = nil
	genesis.Config.AutonityContractConfig = &params.AutonityContractGenesis{}

	genesis.Alloc = core.GenesisAlloc{}
	for _, validator := range validators {
		genesis.Alloc[crypto.PubkeyToAddress(validator.privateKey.PublicKey)] = core.GenesisAccount{
			Balance: new(big.Int).Exp(big.NewInt(2), big.NewInt(128), nil),
		}
	}

	validatorsAddresses := make([]common.Address, len(validators))
	for i, validator := range validators {
		validatorsAddresses[i] = crypto.PubkeyToAddress(validator.privateKey.PublicKey)
	}

	enodes := make([]string, len(validators))
	for i, validator := range validators {
		enodes[i] = validator.url
	}

	users := make([]params.User, len(validators))
	for i := range validators {
		users[i] = params.User{
			Address: validatorsAddresses[i],
			Enode:   enodes[i],
			Type:    params.UserValidator,
			Stake:   100,
		}
	}
	//generate one sh
	shKey, err := crypto.GenerateKey()
	if err != nil {
		log.Error("Make genesis error", "err", err)
	}
	users = append(users, params.User{
		Address: crypto.PubkeyToAddress(shKey.PublicKey),
		Type:    params.UserStakeHolder,
		Stake:   200,
	})
	genesis.Config.AutonityContractConfig.Users = users
	err = genesis.Config.AutonityContractConfig.AddDefault().Validate()
	if err != nil {
		panic(err)
	}

	err = genesis.SetBFT()
	if err != nil {
		panic(err)
	}

	return genesis
}

func makeValidator(genesis *core.Genesis, nodekey *ecdsa.PrivateKey, listenAddr string, inRate, outRate int64, cons func(basic consensus.Engine) consensus.Engine) (*node.Node, error) {
	// Define the basic configurations for the Ethereum node
	datadir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}

	if listenAddr == "" {
		listenAddr = "0.0.0.0:0"
	}

	configNode := &node.Config{
		Name:    "geth",
		Version: params.Version,
		DataDir: datadir,
		P2P: p2p.Config{
			ListenAddr:  listenAddr,
			NoDiscovery: true,
			MaxPeers:    25,
			PrivateKey:  nodekey,
		},
		NoUSB: true,
	}

	if inRate != 0 || outRate != 0 {
		configNode.P2P.IsRated = true
		configNode.P2P.InRate = inRate
		configNode.P2P.OutRate = outRate
	}

	// Start the node and configure a full Ethereum node on it
	stack, err := node.New(configNode)
	if err != nil {
		return nil, err
	}
	if err := stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		return eth.New(ctx, &eth.Config{
			Genesis:         genesis,
			NetworkId:       genesis.Config.ChainID.Uint64(),
			SyncMode:        downloader.FullSync,
			DatabaseCache:   256,
			DatabaseHandles: 256,
			TxPool:          core.DefaultTxPoolConfig,
			Tendermint:      *config.DefaultConfig(),
		}, cons)
	}); err != nil {
		return nil, err
	}

	// Start the node and return if successful
	return stack, nil
}
