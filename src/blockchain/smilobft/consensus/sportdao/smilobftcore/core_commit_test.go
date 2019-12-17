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

package smilobftcore

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"go-smilo/src/blockchain/smilobft/cmn"
	"go-smilo/src/blockchain/smilobft/consensus/sportdao"
	"go-smilo/src/blockchain/smilobft/consensus/sportdao/fullnode"
)

func TestHandleCommit(t *testing.T) {

	expectedConsensus := map[int]int{7: 5, 8: 6, 9: 6, 10: 7}
	for availableNodes := range expectedConsensus {

		proposal := newTestBlockProposal()
		expectedSubject := &sportdao.Subject{
			View: &sportdao.View{
				Round:    big.NewInt(0),
				Sequence: proposal.Number(),
			},
			Digest: proposal.Hash(),
		}

		testCases := []struct {
			name        string
			system      *testSystem
			expectedErr error
		}{
			{
				// normal case
				fmt.Sprintf("normal case %d", availableNodes),
				func() *testSystem {
					sys := NewTestSystemWithBackend(availableNodes)

					for i, backend := range sys.backends {
						c := backend.engine.(*core)
						c.fullnodeSet = backend.peers
						c.current = newTestRoundState(
							&sportdao.View{
								Round:    big.NewInt(0),
								Sequence: big.NewInt(1),
							},
							c.fullnodeSet,
						)

						if i == 0 {
							// replica 0 is the speaker
							c.state = StatePrepared
						}
					}
					return sys
				}(),
				nil,
			},
			{
				// future message
				fmt.Sprintf("future message %d", availableNodes),
				func() *testSystem {
					sys := NewTestSystemWithBackend(availableNodes)

					for i, backend := range sys.backends {
						c := backend.engine.(*core)
						c.fullnodeSet = backend.peers
						if i == 0 {
							// replica 0 is the speaker
							c.current = newTestRoundState(
								expectedSubject.View,
								c.fullnodeSet,
							)
							c.state = StatePreprepared
						} else {
							c.current = newTestRoundState(
								&sportdao.View{
									Round:    big.NewInt(2),
									Sequence: big.NewInt(3),
								},
								c.fullnodeSet,
							)
						}
					}
					return sys
				}(),
				errFutureMessage,
			},
			{
				//
				fmt.Sprintf("subject not match %d", availableNodes),
				func() *testSystem {
					sys := NewTestSystemWithBackend(availableNodes)

					for i, backend := range sys.backends {
						c := backend.engine.(*core)
						c.fullnodeSet = backend.peers
						if i == 0 {
							// replica 0 is the speaker
							c.current = newTestRoundState(
								expectedSubject.View,
								c.fullnodeSet,
							)
							c.state = StatePreprepared
						} else {
							c.current = newTestRoundState(
								&sportdao.View{
									Round:    big.NewInt(0),
									Sequence: big.NewInt(0),
								},
								c.fullnodeSet,
							)
						}
					}
					return sys
				}(),
				errOldMessage,
			},
			{
				// jump state
				fmt.Sprintf("jump state %d", availableNodes),
				func() *testSystem {
					sys := NewTestSystemWithBackend(availableNodes)

					for i, backend := range sys.backends {
						c := backend.engine.(*core)
						c.fullnodeSet = backend.peers
						c.current = newTestRoundState(
							&sportdao.View{
								Round:    big.NewInt(0),
								Sequence: proposal.Number(),
							},
							c.fullnodeSet,
						)

						// only replica0 stays at StatePreprepared
						// other replicas are at StatePrepared
						if i != 0 {
							c.state = StatePrepared
						} else {
							c.state = StatePreprepared
						}
					}
					return sys
				}(),
				nil,
			},
		}

		for _, test := range testCases {

			t.Run(test.name, func(t *testing.T) {

				test.system.Run(false)

				v0 := test.system.backends[0]
				r0 := v0.engine.(*core)

				//66% or more to approve
				minApprovers := expectedConsensus[availableNodes]

				for i := 0; i < minApprovers-1; i++ { // v := range test.system.backends {
					err := sendCommitMessage(r0, uint64(i), test.system.backends[i].engine.(*core).current.Subject())
					if err != nil {
						if err != test.expectedErr {
							t.Errorf("********* ERROR "+test.name+", error mismatch: have %v, want %v", err, test.expectedErr)
						}
						if r0.current.IsHashLocked() {
							t.Errorf("********* ERROR " + test.name + ", block should not be locked")
						}
						return
					}
				}

				if r0.state == StateCommitted {
					t.Errorf("********* ERROR "+test.name+", committed with %v commit messages and must be at least %v", r0.current.Commits.Size(), minApprovers)
					return
				}

				//Send duplicate message
				err := sendCommitMessage(r0, uint64(minApprovers-2), test.system.backends[uint64(minApprovers-2)].engine.(*core).current.Subject())
				require.NoError(t, err)

				if r0.state == StateCommitted {
					t.Errorf("********* ERROR "+test.name+", double committed message was counted twice at index %v", minApprovers-2)
					return
				}

				err = sendCommitMessage(r0, uint64(minApprovers-1), test.system.backends[uint64(minApprovers-1)].engine.(*core).current.Subject())
				require.NoError(t, err)

				// prepared is normal case
				if r0.state != StateCommitted {
					// There are not enough commit messages in core
					if r0.state != StatePrepared {
						t.Errorf("********* ERROR "+test.name+", state mismatch: have %v, want %v", r0.state, StatePrepared)
					}

					if r0.current.Commits.Size() > minApprovers {
						t.Errorf("********* ERROR "+test.name+", the size of commit messages should be less than %v", minApprovers)
					}
					if r0.current.IsHashLocked() {
						t.Errorf("********* ERROR " + test.name + ", block should not be locked")
					}
					//continue
					return
				}

				// core should have 2F+E prepare messages
				if r0.current.Commits.Size() < minApprovers {
					t.Errorf("********* ERROR "+test.name+", the size of commit messages should be larger than 2F+E: size %v", r0.current.Commits.Size())
				}

				// check signatures large than 2F+E
				signedCount := 0
				committedSeals := v0.committedMsgs[0].committedSeals
				for _, node := range r0.fullnodeSet.List() {
					for _, seal := range committedSeals {
						if bytes.Equal(node.Address().Bytes(), seal[:common.AddressLength]) {
							signedCount++
							break
						}
					}
				}
				if signedCount < minApprovers {
					t.Errorf("********* ERROR "+test.name+", the expected signed count should be larger or eq than %v, but got %v", minApprovers, signedCount)
				}
				if !r0.current.IsHashLocked() {
					t.Errorf("********* ERROR " + test.name + ", block should be locked")
				}
			})
		}
	}
}

func sendCommitMessage(r0 *core, N uint64, subject *sportdao.Subject) error {
	node := r0.fullnodeSet.GetByIndex(N)
	m, _ := Encode(subject)
	return r0.handleCommit(&message{
		Code:          msgCommit,
		Msg:           m,
		Address:       node.Address(),
		Signature:     []byte{},
		CommittedSeal: node.Address().Bytes(),
	}, node)
}

// round is not checked for now
func TestVerifyCommit(t *testing.T) {
	// for log purpose
	privateKey, _ := crypto.GenerateKey()
	peer := fullnode.NewFullNode(getPublicKeyAddress(privateKey))
	fullnodeSet := fullnode.NewFullnodeSet([]common.Address{peer.Address()}, sportdao.RoundRobin)

	N := 1
	sys := NewTestSystemWithBackend(N)

	testCases := []struct {
		name       string
		expected   error
		commit     *sportdao.Subject
		roundState *roundState
	}{
		{
			// normal case
			name:     "normal case",
			expected: nil,
			commit: &sportdao.Subject{
				View:   &sportdao.View{Round: big.NewInt(0), Sequence: big.NewInt(0)},
				Digest: newTestBlockProposal().Hash(),
			},
			roundState: newTestRoundState(
				&sportdao.View{Round: big.NewInt(0), Sequence: big.NewInt(0)},
				fullnodeSet,
			),
		},
		{
			// old message
			name:     "old message",
			expected: errInconsistentSubject,
			commit: &sportdao.Subject{
				View:   &sportdao.View{Round: big.NewInt(0), Sequence: big.NewInt(0)},
				Digest: newTestBlockProposal().Hash(),
			},
			roundState: newTestRoundState(
				&sportdao.View{Round: big.NewInt(1), Sequence: big.NewInt(1)},
				fullnodeSet,
			),
		},
		{
			// different digest
			name:     "different digest",
			expected: errInconsistentSubject,
			commit: &sportdao.Subject{
				View:   &sportdao.View{Round: big.NewInt(0), Sequence: big.NewInt(0)},
				Digest: cmn.StringToHash("1234567890"),
			},
			roundState: newTestRoundState(
				&sportdao.View{Round: big.NewInt(1), Sequence: big.NewInt(1)},
				fullnodeSet,
			),
		},
		{
			// malicious package(lack of sequence)
			name:     "malicious package(lack of sequence)",
			expected: errInconsistentSubject,
			commit: &sportdao.Subject{
				View:   &sportdao.View{Round: big.NewInt(0), Sequence: nil},
				Digest: newTestBlockProposal().Hash(),
			},
			roundState: newTestRoundState(
				&sportdao.View{Round: big.NewInt(1), Sequence: big.NewInt(1)},
				fullnodeSet,
			),
		},
		{
			// wrong prepare message with same sequence but different round
			name:     "wrong prepare message with same sequence but different round",
			expected: errInconsistentSubject,
			commit: &sportdao.Subject{
				View:   &sportdao.View{Round: big.NewInt(1), Sequence: big.NewInt(0)},
				Digest: newTestBlockProposal().Hash(),
			},
			roundState: newTestRoundState(
				&sportdao.View{Round: big.NewInt(0), Sequence: big.NewInt(0)},
				fullnodeSet,
			),
		},
		{
			// wrong prepare message with same round but different sequence
			name:     "wrong prepare message with same round but different sequence",
			expected: errInconsistentSubject,
			commit: &sportdao.Subject{
				View:   &sportdao.View{Round: big.NewInt(0), Sequence: big.NewInt(1)},
				Digest: newTestBlockProposal().Hash(),
			},
			roundState: newTestRoundState(
				&sportdao.View{Round: big.NewInt(0), Sequence: big.NewInt(0)},
				fullnodeSet,
			),
		},
	}
	for i, test := range testCases {

		t.Run(test.name, func(t *testing.T) {

			c := sys.backends[0].engine.(*core)
			c.current = test.roundState

			if err := c.verifyCommit(test.commit, peer); err != nil {
				if err != test.expected {
					t.Errorf("result %d: error mismatch: have %v, want %v", i, err, test.expected)
				}
			}
		})
	}
}
