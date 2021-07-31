package backend

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go-smilo/src/blockchain/smilobft/cmn/acdefault"
	"go-smilo/src/blockchain/smilobft/contracts/autonity_tendermint_060"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"

	"go-smilo/src/blockchain/smilobft/consensus"
	"go-smilo/src/blockchain/smilobft/core/types"
	"go-smilo/src/blockchain/smilobft/rpc"
)

func TestGetCommittee(t *testing.T) {
	want := types.Committee{}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	c := consensus.NewMockChainReader(ctrl)
	h := &types.Header{Number: big.NewInt(1)}
	c.EXPECT().GetHeaderByNumber(uint64(1)).Return(h)
	API := &API{
		chain: c,
		getCommittee: func(header *types.Header, chain consensus.ChainReader) (types.Committee, error) {
			if header == h && chain == c {
				return want, nil
			}
			return nil, nil
		},
	}

	bn := rpc.BlockNumber(1)

	got, err := API.GetCommittee(&bn)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestGetCommitteeAtHash(t *testing.T) {
	t.Run("unknown block given, error returned", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		hash := common.HexToHash("0x0123456789")

		chain := consensus.NewMockChainReader(ctrl)
		chain.EXPECT().GetHeaderByHash(hash).Return(nil)

		API := &API{
			chain: chain,
		}

		_, err := API.GetCommitteeAtHash(hash)
		if err != errUnknownBlock {
			t.Fatalf("expected %v, got %v", errUnknownBlock, err)
		}
	})

	t.Run("valid block given, committee returned", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		hash := common.HexToHash("0x0123456789")

		c := consensus.NewMockChainReader(ctrl)
		h := &types.Header{Number: big.NewInt(1)}
		c.EXPECT().GetHeaderByHash(hash).Return(h)

		want := types.Committee{}

		API := &API{
			chain: c,
			getCommittee: func(header *types.Header, chain consensus.ChainReader) (types.Committee, error) {
				if header == h && chain == c {
					return want, nil
				}
				return nil, nil
			},
		}

		got, err := API.GetCommitteeAtHash(hash)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})
}

func TestAPIGetContractABI(t *testing.T) {
	chain, engine := newBlockChain(1)
	block, err := makeBlock(chain, engine, chain.Genesis())
	require.Nil(t, err)
	_, err = chain.InsertChain(types.Blocks{block})
	require.Nil(t, err)

	want := acdefault.ABI()

	API := &API{
		tendermint: engine,
	}

	got := API.GetContractABI()
	assert.Equal(t, want, got)
}

func TestAPIGetContractAddress(t *testing.T) {
	chain, engine := newBlockChain(1)
	block, err := makeBlock(chain, engine, chain.Genesis())
	assert.Nil(t, err)
	_, err = chain.InsertChain(types.Blocks{block})
	assert.Nil(t, err)

	want := autonity_tendermint_060.ContractAddress

	API := &API{
		tendermint: engine,
	}

	got := API.GetContractAddress()
	assert.Equal(t, want, got)
}

func TestAPIGetWhitelist(t *testing.T) {
	chain, engine := newBlockChain(1)
	block, err := makeBlock(chain, engine, chain.Genesis())
	assert.Nil(t, err)
	_, err = chain.InsertChain(types.Blocks{block})
	assert.Nil(t, err)

	want := []string{"enode://d73b857969c86415c0c000371bcebd9ed3cca6c376032b3f65e58e9e2b79276fbc6f59eb1e22fcd6356ab95f42a666f70afd4985933bd8f3e05beb1a2bf8fdde@172.25.0.11:30303"}

	API := &API{
		tendermint: engine,
	}

	got := API.GetWhitelist()

	assert.Equal(t, want, got)
}
