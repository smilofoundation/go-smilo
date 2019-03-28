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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"strconv"

	"go-smilo/src/blockchain/smilobft/cmn"
	"go-smilo/src/blockchain/smilobft/consensus/sport"
	"go-smilo/src/blockchain/smilobft/consensus/sport/fullnode"
)

func TestHandleCommit(t *testing.T) {

	expectedConsensus := map[uint64]int{7: 5, 8: 6, 9: 6, 10: 7}
	for N := range expectedConsensus {

		proposal := newTestBlockProposal()
		expectedSubject := &sport.Subject{
			View: &sport.View{
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
				"normal case " + strconv.FormatUint(N, 10),
				func() *testSystem {
					sys := NewTestSystemWithBackend(N)

					for i, backend := range sys.backends {
						c := backend.engine.(*core)
						c.fullnodeSet = backend.peers
						c.current = newTestRoundState(
							&sport.View{
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
				"future message " + strconv.FormatUint(N, 10),
				func() *testSystem {
					sys := NewTestSystemWithBackend(N)

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
								&sport.View{
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
				"subject not match " + strconv.FormatUint(N, 10),
				func() *testSystem {
					sys := NewTestSystemWithBackend(N)

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
								&sport.View{
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
				"jump state " + strconv.FormatUint(N, 10),
				func() *testSystem {
					sys := NewTestSystemWithBackend(N)

					for i, backend := range sys.backends {
						c := backend.engine.(*core)
						c.fullnodeSet = backend.peers
						c.current = newTestRoundState(
							&sport.View{
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

				for i := 0; i < expectedConsensus[N]-1; i++ { // v := range test.system.backends {
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

				//66% or more to approve
				MinApprovers := expectedConsensus[N]

				if r0.state == StateCommitted {
					t.Errorf("********* ERROR "+test.name+", committed with %v commit messages and must be at least %v", r0.current.Commits.Size(), MinApprovers)
					return
				}

				//Send duplicate message
				sendCommitMessage(r0, uint64(expectedConsensus[N]-2), test.system.backends[uint64(expectedConsensus[N]-2)].engine.(*core).current.Subject())

				if r0.state == StateCommitted {
					t.Errorf("********* ERROR "+test.name+", double committed message was counted twice at index %v", expectedConsensus[N]-2)
					return
				}

				sendCommitMessage(r0, uint64(expectedConsensus[N]-1), test.system.backends[uint64(expectedConsensus[N]-1)].engine.(*core).current.Subject())

				// prepared is normal case
				if r0.state != StateCommitted {
					// There are not enough commit messages in core
					if r0.state != StatePrepared {
						t.Errorf("********* ERROR "+test.name+", state mismatch: have %v, want %v", r0.state, StatePrepared)
					}

					if r0.current.Commits.Size() > MinApprovers {
						t.Errorf("********* ERROR "+test.name+", the size of commit messages should be less than %v", MinApprovers)
					}
					if r0.current.IsHashLocked() {
						t.Errorf("********* ERROR " + test.name + ", block should not be locked")
					}
					//continue
					return
				}

				// core should have 2F+E prepare messages
				if r0.current.Commits.Size() < MinApprovers {
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
				if signedCount < MinApprovers {
					t.Errorf("********* ERROR "+test.name+", the expected signed count should be larger or eq than %v, but got %v", MinApprovers, signedCount)
				}
				if !r0.current.IsHashLocked() {
					t.Errorf("********* ERROR " + test.name + ", block should be locked")
				}
			})
		}
	}
}

func sendCommitMessage(r0 *core, N uint64, subject *sport.Subject) error {
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
	fullnodeSet := fullnode.NewFullnodeSet([]common.Address{peer.Address()}, sport.RoundRobin)

	N := uint64(1)
	sys := NewTestSystemWithBackend(N)

	testCases := []struct {
		name       string
		expected   error
		commit     *sport.Subject
		roundState *roundState
	}{
		{
			// normal case
			name:     "normal case",
			expected: nil,
			commit: &sport.Subject{
				View:   &sport.View{Round: big.NewInt(0), Sequence: big.NewInt(0)},
				Digest: newTestBlockProposal().Hash(),
			},
			roundState: newTestRoundState(
				&sport.View{Round: big.NewInt(0), Sequence: big.NewInt(0)},
				fullnodeSet,
			),
		},
		{
			// old message
			name:     "old message",
			expected: errInconsistentSubject,
			commit: &sport.Subject{
				View:   &sport.View{Round: big.NewInt(0), Sequence: big.NewInt(0)},
				Digest: newTestBlockProposal().Hash(),
			},
			roundState: newTestRoundState(
				&sport.View{Round: big.NewInt(1), Sequence: big.NewInt(1)},
				fullnodeSet,
			),
		},
		{
			// different digest
			name:     "different digest",
			expected: errInconsistentSubject,
			commit: &sport.Subject{
				View:   &sport.View{Round: big.NewInt(0), Sequence: big.NewInt(0)},
				Digest: cmn.StringToHash("1234567890"),
			},
			roundState: newTestRoundState(
				&sport.View{Round: big.NewInt(1), Sequence: big.NewInt(1)},
				fullnodeSet,
			),
		},
		{
			// malicious package(lack of sequence)
			name:     "malicious package(lack of sequence)",
			expected: errInconsistentSubject,
			commit: &sport.Subject{
				View:   &sport.View{Round: big.NewInt(0), Sequence: nil},
				Digest: newTestBlockProposal().Hash(),
			},
			roundState: newTestRoundState(
				&sport.View{Round: big.NewInt(1), Sequence: big.NewInt(1)},
				fullnodeSet,
			),
		},
		{
			// wrong prepare message with same sequence but different round
			name:     "wrong prepare message with same sequence but different round",
			expected: errInconsistentSubject,
			commit: &sport.Subject{
				View:   &sport.View{Round: big.NewInt(1), Sequence: big.NewInt(0)},
				Digest: newTestBlockProposal().Hash(),
			},
			roundState: newTestRoundState(
				&sport.View{Round: big.NewInt(0), Sequence: big.NewInt(0)},
				fullnodeSet,
			),
		},
		{
			// wrong prepare message with same round but different sequence
			name:     "wrong prepare message with same round but different sequence",
			expected: errInconsistentSubject,
			commit: &sport.Subject{
				View:   &sport.View{Round: big.NewInt(0), Sequence: big.NewInt(1)},
				Digest: newTestBlockProposal().Hash(),
			},
			roundState: newTestRoundState(
				&sport.View{Round: big.NewInt(0), Sequence: big.NewInt(0)},
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
