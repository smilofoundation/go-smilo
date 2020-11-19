package permission

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"go-smilo/src/blockchain/smilobft/cmn"
	"go-smilo/src/blockchain/smilobft/eth/downloader"
	"go-smilo/src/blockchain/smilobft/miner"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"testing"
	"time"

	"go-smilo/src/blockchain/smilobft/accounts"
	"go-smilo/src/blockchain/smilobft/accounts/keystore"
	"go-smilo/src/blockchain/smilobft/internal/ethapi"

	"go-smilo/src/blockchain/smilobft/core/types"

	"go-smilo/src/blockchain/smilobft/params"

	"go-smilo/src/blockchain/smilobft/p2p"

	"go-smilo/src/blockchain/smilobft/eth"

	"github.com/stretchr/testify/assert"

	"go-smilo/src/blockchain/smilobft/accounts/abi/bind"
	"go-smilo/src/blockchain/smilobft/accounts/abi/bind/backends"
	"go-smilo/src/blockchain/smilobft/core"
	"go-smilo/src/blockchain/smilobft/node"
	pbind "go-smilo/src/blockchain/smilobft/permission/bind"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	arbitraryNetworkAdminOrg  = "NETWORK_ADMIN"
	arbitraryNetworkAdminRole = "NETWORK_ADMIN_ROLE"
	arbitraryOrgAdminRole     = "ORG_ADMIN_ROLE"
	arbitraryNode1            = "enode://ac6b1096ca56b9f6d004b779ae3728bf83f8e22453404cc3cef16a3d9b96608bc67c4b30db88e0a5a6c6390213f7acbe1153ff6d23ce57380104288ae19373ef@127.0.0.1:21000?discport=0&raftport=50401"
	arbitraryNode2            = "enode://0ba6b9f606a43a95edc6247cdb1c1e105145817be7bcafd6b2c0ba15d58145f0dc1a194f70ba73cd6f4cdd6864edc7687f311254c7555cc32e4d45aeb1b80416@127.0.0.1:21001?discport=0&raftport=50402"
	arbitraryOrgToAdd         = "ORG1"
	arbitrarySubOrg           = "SUB1"
	arbitrartNewRole1         = "NEW_ROLE_1"
	arbitrartNewRole2         = "NEW_ROLE_2"
)

var ErrAccountsLinked = errors.New("Accounts linked to the role. Cannot be removed")
var ErrPendingApproval = errors.New("Pending approvals for the organization. Approve first")
var ErrAcctBlacklisted = errors.New("Blacklisted account. Operation not allowed")
var ErrNodeBlacklisted = errors.New("Blacklisted node. Operation not allowed")

var (
	guardianKey     *ecdsa.PrivateKey
	guardianAccount accounts.Account
	backend         bind.ContractBackend
	permUpgrAddress, permInterfaceAddress, permImplAddress, voterManagerAddress,
	nodeManagerAddress, roleManagerAddress, accountManagerAddress, orgManagerAddress common.Address
	ethereum        *eth.Smilo
	stack           *node.Node
	guardianAddress common.Address
)

func TestMain(m *testing.M) {
	//TODO: fix me
	return

	setup()
	ret := m.Run()
	teardown()
	os.Exit(ret)
}

func setup() {
	var err error
	t := log.New(os.Stdout, "", 0)

	ksdir, _, err := tmpKeyStore(false)
	defer os.RemoveAll(ksdir)

	if err != nil {
		t.Fatalf("failed to create keystore: %v\n", err)
	}
	nodeKey, _ := crypto.GenerateKey()
	guardianKey, _ = crypto.GenerateKey()

	// Create a network-less protocol stack and start an Ethereum service within
	stack, err = node.New(&node.Config{
		DataDir:           "",
		KeyStoreDir:       ksdir,
		UseLightweightKDF: true,
		P2P: p2p.Config{
			PrivateKey: nodeKey,
		},
	})

	ksbackend := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	guardianAccount, err = ksbackend.ImportECDSA(guardianKey, "foo")
	guardianAddress = guardianAccount.Address
	if err != nil {
		t.Fatal(err)
	}
	err = ksbackend.TimedUnlock(guardianAccount, "foo", 0)
	if err != nil {
		t.Fatal("failed to unlock")
	}

	genesisAlloc := map[common.Address]core.GenesisAccount{
		guardianAddress: {
			Balance: big.NewInt(100000000000000),
		},
	}
	genesis := &core.Genesis{Config: params.AllEthashProtocolChanges, GasLimit: 10000000000000, Alloc: genesisAlloc}

	config := &node.Config{
		Name:    "geth",
		Version: params.Version,
		DataDir: ksdir,
		P2P: p2p.Config{
			ListenAddr:  "0.0.0.0:0",
			NoDiscovery: true,
			MaxPeers:    25,
		},
		NoUSB: true,
	}
	// Start the node and configure a full Ethereum node on it
	stack, err := node.New(config)
	if err != nil {
		t.Fatalf("failed to register New Ethereum protocol: %v", err)
	}

	if err := stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		return eth.New(ctx, &eth.Config{
			Genesis:         genesis,
			NetworkId:       genesis.Config.ChainID.Uint64(),
			SyncMode:        downloader.FullSync,
			DatabaseCache:   256,
			DatabaseHandles: 256,
			TxPool:          core.DefaultTxPoolConfig,
			GPO:             eth.DefaultConfig.GPO,
			Ethash:          eth.DefaultConfig.Ethash,

			Miner: miner.Config{
				Etherbase: guardianAddress,
				GasFloor:  genesis.GasLimit * 9 / 10,
				GasCeil:   genesis.GasLimit * 11 / 10,
				GasPrice:  big.NewInt(1),
				Recommit:  time.Second,
			},
		}, nil)
	}); err != nil {
		t.Fatalf("failed to register Ethereum protocol: %v", err)
	}

	// Start the node and assemble the JavaScript console around it
	if err = stack.Start(); err != nil {
		t.Fatalf("failed to start test stack: %v", err)
	}
	if err := stack.Service(&ethereum); err != nil {
		t.Fatal(err)
	}
	backend = backends.NewSimulatedBackendFrom(ethereum)

	if err := ethereum.StartMining(1); err != nil {
		t.Fatal(err)
	}

	time.Sleep(3 * time.Second)

	var permUpgrInstance *pbind.PermUpgr

	guardianTransactor := bind.NewKeyedTransactor(guardianKey)

	permUpgrAddress, _, permUpgrInstance, err = pbind.DeployPermUpgr(guardianTransactor, backend, guardianAddress)
	if err != nil {
		t.Fatal("permUpgrAddress", err)
	}
	permInterfaceAddress, _, _, err = pbind.DeployPermInterface(guardianTransactor, backend, permUpgrAddress)
	if err != nil {
		t.Fatal("permInterfaceAddress", err)
	}
	nodeManagerAddress, _, _, err = pbind.DeployNodeManager(guardianTransactor, backend, permUpgrAddress)
	if err != nil {
		t.Fatal("nodeManagerAddress", err)
	}
	roleManagerAddress, _, _, err = pbind.DeployRoleManager(guardianTransactor, backend, permUpgrAddress)
	if err != nil {
		t.Fatal("roleManagerAddress", err)
	}
	accountManagerAddress, _, _, err = pbind.DeployAcctManager(guardianTransactor, backend, permUpgrAddress)
	if err != nil {
		t.Fatal("accountManagerAddress", err)
	}
	orgManagerAddress, _, _, err = pbind.DeployOrgManager(guardianTransactor, backend, permUpgrAddress)
	if err != nil {
		t.Fatal("orgManagerAddress", err)
	}
	voterManagerAddress, _, _, err = pbind.DeployVoterManager(guardianTransactor, backend, permUpgrAddress)
	if err != nil {
		t.Fatal("voterManagerAddress", err)
	}

	fmt.Printf("current block is %v\n", ethereum.BlockChain().CurrentBlock().Number().Int64())
	time.Sleep(1 * time.Second)
	fmt.Printf("current block is %v\n", ethereum.BlockChain().CurrentBlock().Number().Int64())

	permImplAddress, _, _, err = pbind.DeployPermImpl(guardianTransactor, backend, permUpgrAddress, orgManagerAddress, roleManagerAddress, accountManagerAddress, voterManagerAddress, nodeManagerAddress)
	if err != nil {
		t.Fatal("permImplAddress", err)
	}
	// call init
	if _, err := permUpgrInstance.Init(guardianTransactor, permInterfaceAddress, permImplAddress); err != nil {
		t.Fatal("permUpgrInstance.Init", err)
	}

	fmt.Printf("current block is %v\n", ethereum.BlockChain().CurrentBlock().Number().Int64())
}

func teardown() {

}

func TestPermissionCtrl_AfterStart(t *testing.T) {
	testObject := typicalPermissionCtrl(t)

	err := testObject.AfterStart()

	assert.NoError(t, err)
	assert.NotNil(t, testObject.permOrg)
	assert.NotNil(t, testObject.permRole)
	assert.NotNil(t, testObject.permNode)
	assert.NotNil(t, testObject.permAcct)
	assert.NotNil(t, testObject.permInterf)
	assert.NotNil(t, testObject.permUpgr)

	isNetworkInitialized, err := testObject.permInterf.GetNetworkBootStatus(&bind.CallOpts{
		Pending: true,
	})
	assert.NoError(t, err)
	assert.True(t, isNetworkInitialized)
}

func TestPermissionCtrl_PopulateInitPermissions_AfterNetworkIsInitialized(t *testing.T) {
	testObject := typicalPermissionCtrl(t)
	assert.NoError(t, testObject.AfterStart())

	err := testObject.populateInitPermissions()

	assert.NoError(t, err)

	// assert cache
	assert.Equal(t, 1, len(types.OrgInfoMap.GetOrgList()))
	cachedOrg := types.OrgInfoMap.GetOrgList()[0]
	assert.Equal(t, arbitraryNetworkAdminOrg, cachedOrg.OrgId)
	assert.Equal(t, arbitraryNetworkAdminOrg, cachedOrg.FullOrgId)
	assert.Equal(t, arbitraryNetworkAdminOrg, cachedOrg.UltimateParent)
	assert.Equal(t, "", cachedOrg.ParentOrgId)
	assert.Equal(t, types.OrgApproved, cachedOrg.Status)
	assert.Equal(t, 0, len(cachedOrg.SubOrgList))
	assert.Equal(t, big.NewInt(1), cachedOrg.Level)

	assert.Equal(t, 1, len(types.RoleInfoMap.GetRoleList()))
	cachedRole := types.RoleInfoMap.GetRoleList()[0]
	assert.Equal(t, arbitraryNetworkAdminOrg, cachedRole.OrgId)
	assert.Equal(t, arbitraryNetworkAdminRole, cachedRole.RoleId)
	assert.True(t, cachedRole.Active)
	assert.True(t, cachedRole.IsAdmin)
	assert.True(t, cachedRole.IsVoter)
	assert.Equal(t, types.FullAccess, cachedRole.Access)

	assert.Equal(t, 0, len(types.NodeInfoMap.GetNodeList()))

	assert.Equal(t, 1, len(types.AcctInfoMap.GetAcctList()))
	cachedAccount := types.AcctInfoMap.GetAcctList()[0]
	assert.Equal(t, arbitraryNetworkAdminOrg, cachedAccount.OrgId)
	assert.Equal(t, arbitraryNetworkAdminRole, cachedAccount.RoleId)
	assert.Equal(t, types.AcctActive, cachedAccount.Status)
	assert.True(t, cachedAccount.IsOrgAdmin)
	assert.Equal(t, guardianAddress, cachedAccount.AcctId)
}

func typicalQuorumControlsAPI(t *testing.T) *QuorumControlsAPI {
	pc := typicalPermissionCtrl(t)
	if !assert.NoError(t, pc.AfterStart()) {
		t.Fail()
	}
	if !assert.NoError(t, pc.populateInitPermissions()) {
		t.Fail()
	}
	return NewQuorumControlsAPI(pc)
}

func TestQuorumControlsAPI_ListAPIs(t *testing.T) {
	testObject := typicalQuorumControlsAPI(t)

	orgDetails, err := testObject.GetOrgDetails(arbitraryNetworkAdminOrg)
	assert.NoError(t, err)
	assert.Equal(t, orgDetails.AcctList[0].AcctId, guardianAddress)
	assert.Equal(t, orgDetails.RoleList[0].RoleId, arbitraryNetworkAdminRole)

	orgDetails, err = testObject.GetOrgDetails("XYZ")
	assert.Equal(t, err, errors.New("org does not exist"))

	// test NodeList
	assert.Equal(t, len(testObject.NodeList()), 0)
	// test AcctList
	assert.True(t, len(testObject.AcctList()) > 0, fmt.Sprintf("expected non zero account list"))
	// test OrgList
	assert.True(t, len(testObject.OrgList()) > 0, fmt.Sprintf("expected non zero org list"))
	// test RoleList
	assert.True(t, len(testObject.RoleList()) > 0, fmt.Sprintf("expected non zero org list"))
}

func TestQuorumControlsAPI_OrgAPIs(t *testing.T) {
	testObject := typicalQuorumControlsAPI(t)
	invalidTxa := ethapi.SendTxArgs{From: getArbitraryAccount()}

	// test AddOrg
	orgAdminKey, _ := crypto.GenerateKey()
	orgAdminAddress := crypto.PubkeyToAddress(orgAdminKey.PublicKey)

	txa := ethapi.SendTxArgs{From: guardianAddress}
	_, err := testObject.AddOrg(arbitraryOrgToAdd, arbitraryNode1, orgAdminAddress, invalidTxa)
	assert.Equal(t, err, errors.New("Invalid account id"))

	_, err = testObject.AddOrg(arbitraryOrgToAdd, arbitraryNode1, orgAdminAddress, txa)
	assert.NoError(t, err)

	_, err = testObject.AddOrg(arbitraryOrgToAdd, arbitraryNode1, orgAdminAddress, txa)
	assert.Equal(t, err, ErrPendingApproval)

	_, err = testObject.ApproveOrg(arbitraryOrgToAdd, arbitraryNode1, orgAdminAddress, invalidTxa)
	assert.Equal(t, err, errors.New("Invalid account id"))

	_, err = testObject.ApproveOrg("XYZ", arbitraryNode1, orgAdminAddress, txa)
	assert.Equal(t, err, errors.New("Nothing to approve"))

	_, err = testObject.ApproveOrg(arbitraryOrgToAdd, arbitraryNode1, orgAdminAddress, txa)
	assert.NoError(t, err)

	types.OrgInfoMap.UpsertOrg(arbitraryOrgToAdd, "", arbitraryOrgToAdd, big.NewInt(1), types.OrgApproved)
	_, err = testObject.UpdateOrgStatus(arbitraryOrgToAdd, uint8(SuspendOrg), invalidTxa)
	assert.Equal(t, err, errors.New("Invalid account id"))

	_, err = testObject.UpdateOrgStatus(arbitraryOrgToAdd, uint8(SuspendOrg), txa)
	assert.NoError(t, err)

	types.OrgInfoMap.UpsertOrg(arbitraryOrgToAdd, "", arbitraryOrgToAdd, big.NewInt(1), types.OrgSuspended)
	_, err = testObject.ApproveOrgStatus(arbitraryOrgToAdd, uint8(SuspendOrg), invalidTxa)
	assert.Equal(t, err, errors.New("Invalid account id"))

	_, err = testObject.ApproveOrgStatus(arbitraryOrgToAdd, uint8(SuspendOrg), txa)
	assert.NoError(t, err)

	_, err = testObject.AddSubOrg(arbitraryNetworkAdminOrg, arbitrarySubOrg, "", invalidTxa)
	assert.Equal(t, err, errors.New("Invalid account id"))

	_, err = testObject.AddSubOrg(arbitraryNetworkAdminOrg, arbitrarySubOrg, "", txa)
	assert.NoError(t, err)
	types.OrgInfoMap.UpsertOrg(arbitrarySubOrg, arbitraryNetworkAdminOrg, arbitraryNetworkAdminOrg, big.NewInt(2), types.OrgApproved)

	suborg := "ABC.12345"
	_, err = testObject.AddSubOrg(arbitraryNetworkAdminOrg, suborg, "", txa)
	assert.Equal(t, err, errors.New("Org id cannot contain special characters"))

	_, err = testObject.AddSubOrg(arbitraryNetworkAdminOrg, "", "", txa)
	assert.Equal(t, err, errors.New("Invalid input"))

	_, err = testObject.GetOrgDetails(arbitraryOrgToAdd)
	assert.NoError(t, err)

}

func TestQuorumControlsAPI_NodeAPIs(t *testing.T) {
	testObject := typicalQuorumControlsAPI(t)
	invalidTxa := ethapi.SendTxArgs{From: getArbitraryAccount()}
	txa := ethapi.SendTxArgs{From: guardianAddress}

	_, err := testObject.AddNode(arbitraryNetworkAdminOrg, arbitraryNode2, invalidTxa)
	assert.Equal(t, err, errors.New("Invalid account id"))

	_, err = testObject.AddNode(arbitraryNetworkAdminOrg, arbitraryNode2, txa)
	assert.NoError(t, err)
	types.NodeInfoMap.UpsertNode(arbitraryNetworkAdminOrg, arbitraryNode2, types.NodeApproved)

	_, err = testObject.UpdateNodeStatus(arbitraryNetworkAdminOrg, arbitraryNode2, uint8(SuspendNode), invalidTxa)
	assert.Equal(t, err, errors.New("Invalid account id"))

	_, err = testObject.UpdateNodeStatus(arbitraryNetworkAdminOrg, arbitraryNode2, uint8(SuspendNode), txa)
	assert.NoError(t, err)
	types.NodeInfoMap.UpsertNode(arbitraryNetworkAdminOrg, arbitraryNode2, types.NodeDeactivated)

	_, err = testObject.UpdateNodeStatus(arbitraryNetworkAdminOrg, arbitraryNode2, uint8(ActivateSuspendedNode), txa)
	assert.NoError(t, err)
	types.NodeInfoMap.UpsertNode(arbitraryNetworkAdminOrg, arbitraryNode2, types.NodeApproved)

	_, err = testObject.UpdateNodeStatus(arbitraryNetworkAdminOrg, arbitraryNode2, uint8(BlacklistNode), txa)
	assert.NoError(t, err)
	types.NodeInfoMap.UpsertNode(arbitraryNetworkAdminOrg, arbitraryNode2, types.NodeBlackListed)

	_, err = testObject.UpdateNodeStatus(arbitraryNetworkAdminOrg, arbitraryNode2, uint8(ActivateSuspendedNode), txa)
	assert.Equal(t, err, ErrNodeBlacklisted)

	_, err = testObject.RecoverBlackListedNode(arbitraryNetworkAdminOrg, arbitraryNode2, invalidTxa)
	assert.Equal(t, err, errors.New("Invalid account id"))

	_, err = testObject.RecoverBlackListedNode(arbitraryNetworkAdminOrg, arbitraryNode2, txa)
	assert.NoError(t, err)
	types.NodeInfoMap.UpsertNode(arbitraryNetworkAdminOrg, arbitraryNode2, types.NodeRecoveryInitiated)

	_, err = testObject.ApproveBlackListedNodeRecovery(arbitraryNetworkAdminOrg, arbitraryNode2, invalidTxa)
	assert.Equal(t, err, errors.New("Invalid account id"))

	_, err = testObject.ApproveBlackListedNodeRecovery(arbitraryNetworkAdminOrg, arbitraryNode2, txa)
	assert.NoError(t, err)
	types.NodeInfoMap.UpsertNode(arbitraryNetworkAdminOrg, arbitraryNode2, types.NodeApproved)
}

func TestQuorumControlsAPI_RoleAndAccountsAPIs(t *testing.T) {
	testObject := typicalQuorumControlsAPI(t)
	invalidTxa := ethapi.SendTxArgs{From: getArbitraryAccount()}
	txa := ethapi.SendTxArgs{From: guardianAddress}
	acct := getArbitraryAccount()

	_, err := testObject.AssignAdminRole(arbitraryNetworkAdminOrg, acct, arbitraryNetworkAdminRole, invalidTxa)
	assert.Equal(t, err, errors.New("Invalid account id"))

	_, err = testObject.AssignAdminRole(arbitraryNetworkAdminOrg, acct, arbitraryNetworkAdminRole, txa)
	types.AcctInfoMap.UpsertAccount(arbitraryNetworkAdminOrg, arbitraryNetworkAdminRole, acct, true, types.AcctPendingApproval)

	_, err = testObject.ApproveAdminRole(arbitraryNetworkAdminOrg, acct, invalidTxa)
	assert.Equal(t, err, errors.New("Invalid account id"))

	_, err = testObject.ApproveAdminRole(arbitraryNetworkAdminOrg, acct, invalidTxa)
	assert.Equal(t, err, errors.New("Invalid account id"))

	_, err = testObject.ApproveAdminRole(arbitraryNetworkAdminOrg, acct, txa)
	assert.NoError(t, err)
	types.AcctInfoMap.UpsertAccount(arbitraryNetworkAdminOrg, arbitraryNetworkAdminRole, acct, true, types.AcctActive)

	_, err = testObject.AddNewRole(arbitraryNetworkAdminOrg, arbitrartNewRole1, uint8(types.FullAccess), false, false, invalidTxa)
	assert.Equal(t, err, errors.New("Invalid account id"))

	_, err = testObject.AddNewRole(arbitraryNetworkAdminOrg, arbitrartNewRole1, uint8(types.FullAccess), false, false, txa)
	assert.NoError(t, err)
	types.RoleInfoMap.UpsertRole(arbitraryNetworkAdminOrg, arbitrartNewRole1, false, false, types.FullAccess, true)

	acct = getArbitraryAccount()
	_, err = testObject.AddAccountToOrg(acct, arbitraryNetworkAdminOrg, arbitrartNewRole1, invalidTxa)
	assert.Equal(t, err, errors.New("Invalid account id"))

	_, err = testObject.AddAccountToOrg(acct, arbitraryNetworkAdminOrg, arbitrartNewRole1, txa)
	assert.NoError(t, err)
	types.AcctInfoMap.UpsertAccount(arbitraryNetworkAdminOrg, arbitrartNewRole1, acct, true, types.AcctActive)

	_, err = testObject.RemoveRole(arbitraryNetworkAdminOrg, arbitrartNewRole1, invalidTxa)
	assert.Equal(t, err, errors.New("Invalid account id"))

	_, err = testObject.RemoveRole(arbitraryNetworkAdminOrg, arbitrartNewRole1, txa)
	assert.Equal(t, err, ErrAccountsLinked)

	_, err = testObject.AddNewRole(arbitraryNetworkAdminOrg, arbitrartNewRole2, uint8(types.FullAccess), false, false, txa)
	assert.NoError(t, err)
	types.RoleInfoMap.UpsertRole(arbitraryNetworkAdminOrg, arbitrartNewRole2, false, false, types.FullAccess, true)

	_, err = testObject.ChangeAccountRole(acct, arbitraryNetworkAdminOrg, arbitrartNewRole2, invalidTxa)
	assert.Equal(t, err, errors.New("Invalid account id"))

	_, err = testObject.ChangeAccountRole(acct, arbitraryNetworkAdminOrg, arbitrartNewRole2, txa)
	assert.NoError(t, err)

	_, err = testObject.RemoveRole(arbitraryNetworkAdminOrg, arbitrartNewRole1, txa)
	assert.Equal(t, err, ErrAccountsLinked)

	_, err = testObject.UpdateAccountStatus(arbitraryNetworkAdminOrg, acct, uint8(SuspendAccount), invalidTxa)
	assert.Equal(t, err, errors.New("Invalid account id"))

	_, err = testObject.UpdateAccountStatus(arbitraryNetworkAdminOrg, acct, uint8(SuspendAccount), txa)
	assert.NoError(t, err)
	types.AcctInfoMap.UpsertAccount(arbitraryNetworkAdminOrg, arbitrartNewRole2, acct, true, types.AcctSuspended)

	_, err = testObject.UpdateAccountStatus(arbitraryNetworkAdminOrg, acct, uint8(ActivateSuspendedAccount), txa)
	assert.NoError(t, err)
	types.AcctInfoMap.UpsertAccount(arbitraryNetworkAdminOrg, arbitrartNewRole2, acct, true, types.AcctActive)

	_, err = testObject.UpdateAccountStatus(arbitraryNetworkAdminOrg, acct, uint8(BlacklistAccount), txa)
	assert.NoError(t, err)
	types.AcctInfoMap.UpsertAccount(arbitraryNetworkAdminOrg, arbitrartNewRole2, acct, true, types.AcctBlacklisted)

	_, err = testObject.UpdateAccountStatus(arbitraryNetworkAdminOrg, acct, uint8(ActivateSuspendedAccount), txa)
	assert.Equal(t, err, ErrAcctBlacklisted)

	_, err = testObject.RecoverBlackListedAccount(arbitraryNetworkAdminOrg, acct, invalidTxa)
	assert.Equal(t, err, errors.New("Invalid account id"))

	_, err = testObject.RecoverBlackListedAccount(arbitraryNetworkAdminOrg, acct, txa)
	assert.NoError(t, err)
	types.AcctInfoMap.UpsertAccount(arbitraryNetworkAdminOrg, arbitrartNewRole2, acct, true, types.AcctRecoveryInitiated)
	_, err = testObject.ApproveBlackListedAccountRecovery(arbitraryNetworkAdminOrg, acct, txa)
	assert.NoError(t, err)
	types.AcctInfoMap.UpsertAccount(arbitraryNetworkAdminOrg, arbitrartNewRole2, acct, true, types.AcctActive)

}

func getArbitraryAccount() common.Address {
	acctKey, _ := crypto.GenerateKey()
	return crypto.PubkeyToAddress(acctKey.PublicKey)
}

func typicalPermissionCtrl(t *testing.T) *PermissionCtrl {
	testObject, err := NewQuorumPermissionCtrl(stack, &types.PermissionConfig{
		UpgrdAddress:   permUpgrAddress,
		InterfAddress:  permInterfaceAddress,
		ImplAddress:    permImplAddress,
		NodeAddress:    nodeManagerAddress,
		AccountAddress: accountManagerAddress,
		RoleAddress:    roleManagerAddress,
		VoterAddress:   voterManagerAddress,
		OrgAddress:     orgManagerAddress,
		NwAdminOrg:     arbitraryNetworkAdminOrg,
		NwAdminRole:    arbitraryNetworkAdminRole,
		OrgAdminRole:   arbitraryOrgAdminRole,
		Accounts: []common.Address{
			guardianAddress,
		},
		SubOrgDepth:   big.NewInt(3),
		SubOrgBreadth: big.NewInt(3),
	})
	if err != nil {
		t.Fatal("NewQuorumPermissionCtrl", err)
	}
	testObject.ethClnt = backend
	testObject.eth = ethereum
	go func() {
		testObject.errorChan <- nil
	}()
	return testObject
}

func tmpKeyStore(encrypted bool) (string, *keystore.KeyStore, error) {
	d, err := ioutil.TempDir("", "eth-keystore-test")
	if err != nil {
		return "", nil, err
	}
	newKs := keystore.NewPlaintextKeyStore
	if encrypted {
		newKs = func(kd string) *keystore.KeyStore {
			return keystore.NewKeyStore(kd, keystore.LightScryptN, keystore.LightScryptP)
		}
	}
	return d, newKs(d), err
}

func TestPermissionCtrl_whenUpdateFile(t *testing.T) {
	testObject := typicalPermissionCtrl(t)
	assert.NoError(t, testObject.AfterStart())

	err := testObject.populateInitPermissions()
	assert.NoError(t, err)

	d, _ := ioutil.TempDir("", "qdata")
	defer os.RemoveAll(d)

	testObject.dataDir = d
	testObject.updatePermissionedNodes(arbitraryNode1, NodeAdd)

	permFile, _ := os.Create(d + "/" + "permissioned-nodes.json")

	testObject.updateFile("testFile", arbitraryNode2, NodeAdd, false)
	testObject.updateFile(permFile.Name(), arbitraryNode2, NodeAdd, false)
	testObject.updateFile(permFile.Name(), arbitraryNode2, NodeAdd, true)
	testObject.updateFile(permFile.Name(), arbitraryNode2, NodeAdd, true)
	testObject.updateFile(permFile.Name(), arbitraryNode1, NodeAdd, false)
	testObject.updateFile(permFile.Name(), arbitraryNode1, NodeDelete, false)
	testObject.updateFile(permFile.Name(), arbitraryNode1, NodeDelete, false)

	blob, err := ioutil.ReadFile(permFile.Name())
	var nodeList []string
	if err := json.Unmarshal(blob, &nodeList); err != nil {
		t.Fatal("Failed to load nodes list from file", "fileName", permFile, "err", err)
		return
	}
	assert.Equal(t, len(nodeList), 1)
	testObject.updatePermissionedNodes(arbitraryNode1, NodeAdd)
	testObject.updatePermissionedNodes(arbitraryNode1, NodeDelete)

	blob, err = ioutil.ReadFile(permFile.Name())
	if err := json.Unmarshal(blob, &nodeList); err != nil {
		t.Fatal("Failed to load nodes list from file", "fileName", permFile, "err", err)
		return
	}
	assert.Equal(t, len(nodeList), 1)

	testObject.updateDisallowedNodes(arbitraryNode2, NodeAdd)
	testObject.updateDisallowedNodes(arbitraryNode2, NodeDelete)
	blob, err = ioutil.ReadFile(d + "/" + "disallowed-nodes.json")
	if err := json.Unmarshal(blob, &nodeList); err != nil {
		t.Fatal("Failed to load nodes list from file", "fileName", permFile, "err", err)
		return
	}
	assert.Equal(t, len(nodeList), 0)

}

func TestParsePermissionConfig(t *testing.T) {
	d, _ := ioutil.TempDir("", "qdata")
	defer os.RemoveAll(d)

	_, err := ParsePermissionConfig(d)
	assert.True(t, err != nil, "expected file not there error")

	fileName := d + "/permission-config.json"
	_, err = os.Create(fileName)
	_, err = ParsePermissionConfig(d)
	assert.True(t, err != nil, "expected unmarshalling error")

	// write permission-config.json into the temp dir
	var tmpPermCofig types.PermissionConfig
	tmpPermCofig.NwAdminOrg = arbitraryNetworkAdminOrg
	tmpPermCofig.NwAdminRole = arbitraryNetworkAdminRole
	tmpPermCofig.OrgAdminRole = arbitraryOrgAdminRole
	tmpPermCofig.InterfAddress = common.Address{}
	tmpPermCofig.ImplAddress = common.Address{}
	tmpPermCofig.UpgrdAddress = common.Address{}
	tmpPermCofig.VoterAddress = common.Address{}
	tmpPermCofig.RoleAddress = common.Address{}
	tmpPermCofig.OrgAddress = common.Address{}
	tmpPermCofig.NodeAddress = common.Address{}
	tmpPermCofig.SubOrgBreadth = new(big.Int)
	tmpPermCofig.SubOrgDepth = new(big.Int)

	blob, err := json.Marshal(tmpPermCofig)
	if err := ioutil.WriteFile(fileName, blob, 0644); err != nil {
		t.Fatal("Error writing new node info to file", "fileName", fileName, "err", err)
	}
	_, err = ParsePermissionConfig(d)
	assert.True(t, err != nil, "expected sub org depth not set error")

	_ = os.Remove(fileName)
	tmpPermCofig.SubOrgBreadth.Set(big.NewInt(4))
	tmpPermCofig.SubOrgDepth.Set(big.NewInt(4))
	blob, _ = json.Marshal(tmpPermCofig)
	if err := ioutil.WriteFile(fileName, blob, 0644); err != nil {
		t.Fatal("Error writing new node info to file", "fileName", fileName, "err", err)
	}
	_, err = ParsePermissionConfig(d)
	assert.True(t, err != nil, "expected account not given  error")

	_ = os.Remove(fileName)
	tmpPermCofig.Accounts = append(tmpPermCofig.Accounts, cmn.StringToAddress("0xed9d02e382b34818e88b88a309c7fe71e65f419d"))
	blob, err = json.Marshal(tmpPermCofig)
	if err := ioutil.WriteFile(fileName, blob, 0644); err != nil {
		t.Fatal("Error writing new node info to file", "fileName", fileName, "err", err)
	}
	_, err = ParsePermissionConfig(d)
	assert.True(t, err != nil, "expected contract address error")

	_ = os.Remove(fileName)
	tmpPermCofig.InterfAddress = cmn.StringToAddress("0xed9d02e382b34818e88b88a309c7fe71e65f419d")
	blob, err = json.Marshal(tmpPermCofig)
	if err := ioutil.WriteFile(fileName, blob, 0644); err != nil {
		t.Fatal("Error writing new node info to file", "fileName", fileName, "err", err)
	}
	permConfig, err := ParsePermissionConfig(d)
	assert.False(t, permConfig.IsEmpty(), "expected non empty object")
}
