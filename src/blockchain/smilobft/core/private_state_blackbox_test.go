package core

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	osExec "os/exec"
	"path"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func runBlackbox() (*osExec.Cmd, error) {

	tempdir, err := ioutil.TempDir("", "blackbox")
	if err != nil {
		return nil, err
	}
	// create config.json file
	here, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	blackboxCMD := filepath.Join(here, "../../../../build/third-party", "blackbox-v2-0")
	blackboxDBFile := filepath.Join(tempdir, "blackbox.db")
	blackboxPeersDBFile := filepath.Join(tempdir, "blackbox-peers.db")

	if err = os.MkdirAll(path.Join(tempdir, "sdata"), 0755); err != nil {
		return nil, err
	}
	tmIPCFile := filepath.Join(tempdir, "sdata", "tm.ipc")
	keyData, err := ioutil.ReadFile(filepath.Join(here, "blackbox-test-keys", "tm1.key"))
	if err != nil {
		return nil, err
	}
	publicKeyData, err := ioutil.ReadFile(filepath.Join(here, "blackbox-test-keys", "tm1.pub"))
	if err != nil {
		return nil, err
	}
	blackboxConfigFile := filepath.Join(tempdir, "config.json")
	if err := ioutil.WriteFile(blackboxConfigFile, []byte(fmt.Sprintf(`
{
    "useWhiteList": false,   
    "server": {
        "port": 9000,
        "hostName": "http://localhost",
        "sslConfig": {
            "tls": "OFF",
            "generateKeyStoreIfNotExisted": true,
            "serverKeyStore": "./sdata/c1/server1-keystore",
            "serverKeyStorePassword": "smilo",
            "serverTrustStore": "./sdata/c1/server-truststore",
            "serverTrustStorePassword": "smilo",
            "serverTrustMode": "TOFU",
            "knownClientsFile": "./sdata/c1/knownClients",
            "clientKeyStore": "./c1/client1-keystore",
            "clientKeyStorePassword": "smilo",
            "clientTrustStore": "./c1/client-truststore",
            "clientTrustStorePassword": "smilo",
            "clientTrustMode": "TOFU",
            "knownServersFile": "./sdata/c1/knownServers"
        }
    },
    "peer": [
        {
            "url": "http://localhost:9000"
        }
    ],
    "keys": {
        "passwords": [],
        "keyData": [
            {
                "config": %s,
                "publicKey": "%s"
            }
        ]
    },
    "alwaysSendTo": [],
    "socket": "%s",
    "dbfile": "%s",
    "peersdbfile": "%s",
}
`, string(keyData), string(publicKeyData), tmIPCFile, blackboxDBFile, blackboxPeersDBFile)), os.FileMode(0644)); err != nil {
		return nil, err
	}

	cmdStatusChan := make(chan error)
	cmd := osExec.Command(blackboxCMD, "-configfile", blackboxConfigFile, "-dbfile", blackboxDBFile)
	// run blackbox
	go func() {
		err := cmd.Start()
		cmdStatusChan <- err
	}()
	// wait 30s for blackbox to come up
	var started bool
	go func() {

		for i := 0; i < 10; i++ {
			time.Sleep(3 * time.Second)
			if err := checkFunc(tmIPCFile); err != nil && err == doneErr {
				cmdStatusChan <- err
			} else {
				fmt.Println("Waiting for blackbox to start", "err", err)
			}
		}
		if !started {
			panic("Blackbox never managed to start!")
		}
	}()

	if err := <-cmdStatusChan; err != nil {
		return nil, err
	}
	// wait until blackbox is up
	return cmd, nil
}

// 600a600055600060006001a1
// [1] PUSH1 0x0a (store value)
// [3] PUSH1 0x00 (store addr)
// [4] SSTORE
// [6] PUSH1 0x00
// [8] PUSH1 0x00
// [10] PUSH1 0x01
// [11] LOG1
//
// Store then log
func TestPrivateTransactionBlackbox(t *testing.T) {
	//TODO: Add blackbox OSX/WIN compiled libs, detect os and run appropriate files
	if runtime.GOOS != "linux" {
		t.Skip()
	}

	var (
		key, _      = crypto.GenerateKey()
		helper      = MakeCallHelper()
		privateState  = helper.PrivateState
		publicState = helper.PublicState
	)

	blackboxCmd, err := runBlackbox()
	if err != nil {
		if strings.Contains(err.Error(), "executable file not found") {
			if blackboxCmd, err = runBlackbox(); err != nil {
				t.Fatal(err)
			}
		} else {
			t.Fatal(err)
		}
	}
	defer blackboxCmd.Process.Kill()

	privateContractAddr := common.Address{1}
	pubContractAddr := common.Address{2}
	privateState.SetCode(privateContractAddr, common.Hex2Bytes("600a600055600060006001a1"))
	privateState.SetState(privateContractAddr, common.Hash{}, common.Hash{9})
	publicState.SetCode(pubContractAddr, common.Hex2Bytes("6014600055"))
	publicState.SetState(pubContractAddr, common.Hash{}, common.Hash{19})

	if publicState.Exist(privateContractAddr) {
		t.Error("didn't expect vault contract address to exist on public state")
	}

	// Vault transaction 1
	err = helper.MakeCall(true, key, privateContractAddr, nil)
	if err != nil {
		t.Fatal(err)
	}
	stateEntry := privateState.GetState(privateContractAddr, common.Hash{}).Big()
	if stateEntry.Cmp(big.NewInt(10)) != 0 {
		t.Error("expected state to have 10, got", stateEntry)
	}
	if len(privateState.Logs()) != 1 {
		t.Error("expected vault state to have 1 log, got", len(privateState.Logs()))
	}
	if len(publicState.Logs()) != 0 {
		t.Error("expected public state to have 0 logs, got", len(publicState.Logs()))
	}
	if publicState.Exist(privateContractAddr) {
		t.Error("didn't expect vault contract address to exist on public state")
	}
	if !privateState.Exist(privateContractAddr) {
		t.Error("expected vault contract address to exist on vault state")
	}

	// Public transaction 1
	err = helper.MakeCall(false, key, pubContractAddr, nil)
	if err != nil {
		t.Fatal(err)
	}
	stateEntry = publicState.GetState(pubContractAddr, common.Hash{}).Big()
	if stateEntry.Cmp(big.NewInt(20)) != 0 {
		t.Error("expected state to have 20, got", stateEntry)
	}

	// Vault transaction 2
	err = helper.MakeCall(true, key, privateContractAddr, nil)
	stateEntry = privateState.GetState(privateContractAddr, common.Hash{}).Big()
	if stateEntry.Cmp(big.NewInt(10)) != 0 {
		t.Error("expected state to have 10, got", stateEntry)
	}

	if publicState.Exist(privateContractAddr) {
		t.Error("didn't expect vault contract address to exist on public state")
	}
	if privateState.Exist(pubContractAddr) {
		t.Error("didn't expect public contract address to exist on vault state")
	}
}
