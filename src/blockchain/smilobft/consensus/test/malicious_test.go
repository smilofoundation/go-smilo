package test

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"go-smilo/src/blockchain/smilobft/consensus"
	tendermintCore "go-smilo/src/blockchain/smilobft/consensus/tendermint/core"
)

func TestTendermintOneMalicious(t *testing.T) {
	if testing.Short()  || CONSENSUS_TEST_MODE != "tendermint" {
		t.Skip("skipping test in short mode")
	}
	addedValidatorsBlocks := make(map[common.Hash]uint64)
	removedValidatorsBlocks := make(map[common.Hash]uint64)
	changedValidators := tendermintCore.NewChanges()

	cases := []*testCase{
		{
			name:          "replace a valid validator with invalid one",
			numValidators: 5,
			numBlocks:     10,
			txPerPeer:     1,
			maliciousPeers: map[string]injectors{
				"VE": {
					cons: func(basic consensus.Engine) consensus.Engine {
						return tendermintCore.NewReplaceValidatorCore(basic, changedValidators)
					},
					backs: func(basic tendermintCore.Backend) tendermintCore.Backend {
						return tendermintCore.NewChangeValidatorBackend(t, basic, changedValidators, removedValidatorsBlocks, addedValidatorsBlocks)
					},
				},
			},
			addedValidatorsBlocks:   addedValidatorsBlocks,
			removedValidatorsBlocks: removedValidatorsBlocks,
			changedValidators:       changedValidators,
		},
		{
			name:          "add a validator",
			numValidators: 5,
			numBlocks:     10,
			txPerPeer:     1,
			maliciousPeers: map[string]injectors{
				"VE": {
					cons: func(basic consensus.Engine) consensus.Engine {
						return tendermintCore.NewAddValidatorCore(basic, changedValidators)
					},
					backs: func(basic tendermintCore.Backend) tendermintCore.Backend {
						return tendermintCore.NewChangeValidatorBackend(t, basic, changedValidators, removedValidatorsBlocks, addedValidatorsBlocks)
					},
				},
			},
			addedValidatorsBlocks:   addedValidatorsBlocks,
			removedValidatorsBlocks: removedValidatorsBlocks,
			changedValidators:       changedValidators,
		},
		{
			name:          "remove a validator",
			numValidators: 5,
			numBlocks:     10,
			txPerPeer:     1,
			maliciousPeers: map[string]injectors{
				"VE": {
					cons: func(basic consensus.Engine) consensus.Engine {
						return tendermintCore.NewRemoveValidatorCore(basic, changedValidators)
					},
					backs: func(basic tendermintCore.Backend) tendermintCore.Backend {
						return tendermintCore.NewChangeValidatorBackend(t, basic, changedValidators, removedValidatorsBlocks, addedValidatorsBlocks)
					},
				},
			},
			addedValidatorsBlocks:   addedValidatorsBlocks,
			removedValidatorsBlocks: removedValidatorsBlocks,
			changedValidators:       changedValidators,
		},
	}

	for _, testCase := range cases {
		testCase := testCase
		t.Run(fmt.Sprintf("test case %s", testCase.name), func(t *testing.T) {
			runTest(t, testCase)
		})
	}
}
