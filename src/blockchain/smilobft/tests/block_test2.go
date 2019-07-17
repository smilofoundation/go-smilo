package tests

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/stretchr/testify/require"

	"go-smilo/src/blockchain/smilobft/accounts/abi/bind"
	"go-smilo/src/blockchain/smilobft/accounts/abi/bind/backends"
	"go-smilo/src/blockchain/smilobft/core"
	"go-smilo/src/blockchain/smilobft/core/types"
)

var testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

var waitDeployedTests = map[string]struct {
	code        string
	gas         uint64
	wantAddress common.Address
	wantErr     error
}{
	"successful deploy": {
		code:        `6060604052600a8060106000396000f360606040526008565b00`,
		gas:         3000000,
		wantAddress: common.HexToAddress("0x3a220f351252089d385b29beca14e27f204c296a"),
	},
	//"empty code": {
	//	code:        ``,
	//	gas:         300000,
	//	wantErr:     bind.ErrNoCodeAfterDeploy,
	//	wantAddress: common.HexToAddress("0x3a220f351252089d385b29beca14e27f204c296a"),
	//},
}

func TestBlock(t *testing.T) {


	tx1 := types.NewTransaction(1, common.BytesToAddress(
		[]byte{0x11}), big.NewInt(111), 1111, big.NewInt(11111), []byte{0x11, 0x11, 0x11})
	tx2 := types.NewTransaction(2, common.BytesToAddress(
		[]byte{0x22}), big.NewInt(222), 2222, big.NewInt(22222), []byte{0x22, 0x22, 0x22})
	tx3 := types.NewTransaction(3, common.BytesToAddress(
		[]byte{0x33}), big.NewInt(333), 3333, big.NewInt(33333), []byte{0x33, 0x33, 0x33})
	txs := []*types.Transaction{tx1, tx2, tx3}

	block := types.NewBlock(&types.Header{Number: big.NewInt(314)}, txs, nil, nil)

	require.NotNil(t, block)
	require.NotEmpty(t, block.Hash())

	t.Log(block)

}

func TestWaitDeployed(t *testing.T) {
	return
	for name, test := range waitDeployedTests {
		backend := backends.NewSimulatedBackend(
			core.GenesisAlloc{
				crypto.PubkeyToAddress(testKey.PublicKey): {Balance: big.NewInt(10000000000)},
			}, 10000000,
		)

		// Create the transaction.
		tx := types.NewContractCreation(0, big.NewInt(0), test.gas, big.NewInt(1), common.FromHex(test.code))
		tx, _ = types.SignTx(tx, types.HomesteadSigner{}, testKey)

		// Wait for it to get mined in the background.
		var (
			err     error
			address common.Address
			mined   = make(chan struct{})
			ctx     = context.Background()
		)
		go func() {
			address, err = bind.WaitDeployed(ctx, backend, tx)
			close(mined)
		}()

		// Send and mine the transaction.
		backend.SendTransaction(ctx, tx)
		backend.Commit()

		select {
		case <-mined:
			if err != test.wantErr {
				t.Errorf("test %q: error mismatch: got %q, want %q", name, err, test.wantErr)
			}
			if address != test.wantAddress {
				t.Errorf("test %q: unexpected contract address %s", name, address.Hex())
			}
		case <-time.After(2 * time.Second):
			t.Errorf("test %q: timeout", name)
		}
	}
}

//.SendTransaction
