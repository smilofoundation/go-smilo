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
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"go-smilo/src/blockchain/smilobft/cmn"
	"go-smilo/src/blockchain/smilobft/consensus/sport"
	"go-smilo/src/blockchain/smilobft/consensus/sport/fullnode"
)

func TestHandlePrepare(t *testing.T) {

	expectedConsensus := map[int]int{7: 5, 8: 6, 9: 6, 10: 7}
	for availableNodes := range expectedConsensus {

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
				fmt.Sprintf("normal case %d", availableNodes),
				func() *testSystem {
					sys := NewTestSystemWithBackend(availableNodes)

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
							c.state = StatePreprepared
						}
					}
					return sys
				}(),
				nil,
			},
			{
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
				fmt.Sprintf("old message %d", availableNodes),
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
				fmt.Sprintf("subject not match  %d", availableNodes),
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
								&sport.View{
									Round:    big.NewInt(0),
									Sequence: big.NewInt(1)},
								c.fullnodeSet,
							)
						}
					}
					return sys
				}(),
				errInconsistentSubject,
			},
			{
				fmt.Sprintf("less than 66 percent %d", availableNodes),
				func() *testSystem {
					sys := NewTestSystemWithBackend(availableNodes)

					// save less than 2*F+E replica
					sys.backends = sys.backends[expectedConsensus[availableNodes]:]

					for i, backend := range sys.backends {
						c := backend.engine.(*core)
						c.fullnodeSet = backend.peers
						c.current = newTestRoundState(
							expectedSubject.View,
							c.fullnodeSet,
						)

						if i == 0 {
							// replica 0 is the speaker
							c.state = StatePreprepared
						}
					}
					return sys
				}(),
				nil,
			},
		}

		for _, test := range testCases {
			test.system.Run(false)

			t.Run(test.name, func(t *testing.T) {

				v0 := test.system.backends[0]
				r0 := v0.engine.(*core)

				//66% or more to approve
				minApprovers := expectedConsensus[availableNodes]

				numMessages := minApprovers
				if len(test.system.backends) < minApprovers {
					numMessages = len(test.system.backends)
				}

				for i := 0; i < numMessages-1; i++ {
					err := sendPrepareMessage(r0, i, test.system.backends[i])
					if err != nil {
						if err != test.expectedErr {
							t.Errorf("error mismatch: have %v, want %v", err, test.expectedErr)
						}
						if r0.current.IsHashLocked() {
							t.Errorf("block should not be locked")
						}
						return
					}
				}

				// core should have 66% PREPARE messages
				err := sendPrepareMessage(r0, numMessages-2, test.system.backends[numMessages-2])
				require.NoError(t, err)

				if r0.state == StatePrepared {
					t.Errorf("Reached consensus before 66%% nodes agreed, %v nodes prepared and %v nodes required", r0.current.Prepares.Size(), minApprovers)
				}

				err = sendPrepareMessage(r0, numMessages-1, test.system.backends[numMessages-1])
				require.NoError(t, err)

				// prepared is normal case
				if r0.state != StatePrepared {
					// There are not enough PREPARE messages in core
					if r0.state != StatePreprepared {
						t.Errorf("state mismatch: have %v, want %v", r0.state, StatePreprepared)
					}
					if r0.current.Prepares.Size() >= minApprovers {
						t.Errorf("the size of PREPARE messages should be less than %v", minApprovers)
					}
					if r0.current.IsHashLocked() {
						t.Errorf("block should not be locked")
					}

					return
				}

				if r0.current.Prepares.Size() < minApprovers {
					t.Errorf("the size of PREPARE messages should be equal or larger than %v(66%%): size %v", minApprovers, r0.current.Commits.Size())
				}

				// a message will be delivered to backend if 66% reached
				if int64(len(v0.sentMsgs)) != 1 {
					t.Errorf("the Send() should be called once: times %v", len(test.system.backends[0].sentMsgs))
				}

				// verify COMMIT messages
				decodedMsg := new(message)
				err = decodedMsg.FromPayload(v0.sentMsgs[0], nil)
				if err != nil {
					t.Errorf("error mismatch: have %v, want nil", err)
				}

				if decodedMsg.Code != msgCommit {
					t.Errorf("message code mismatch: have %v, want %v", decodedMsg.Code, msgCommit)
				}
				var m *sport.Subject
				err = decodedMsg.Decode(&m)
				if err != nil {
					t.Errorf("error mismatch: have %v, want nil", err)
				}
				if !reflect.DeepEqual(m, expectedSubject) {
					t.Errorf("subject mismatch: have %v, want %v", m, expectedSubject)
				}
				if !r0.current.IsHashLocked() {
					t.Errorf("block should be locked")
				}
			})
		}
	}
}

func sendPrepareMessage(r0 *core, i int, v *testSystemBackend) error {
	thisfullnode := r0.fullnodeSet.GetByIndex(uint64(i))
	m, _ := Encode(v.engine.(*core).current.Subject())
	err := r0.handlePrepare(&message{
		Code:    msgPrepare,
		Msg:     m,
		Address: thisfullnode.Address(),
	}, thisfullnode)
	return err
}

// round is not checked for now
func TestVerifyPrepare(t *testing.T) {
	// for log purpose
	privateKey, _ := crypto.GenerateKey()
	peer := fullnode.NewFullNode(getPublicKeyAddress(privateKey))
	fullnodeSet := fullnode.NewFullnodeSet([]common.Address{peer.Address()}, sport.RoundRobin)

	availableNodes := 1

	sys := NewTestSystemWithBackend(availableNodes)

	testCases := []struct {
		name     string
		expected error

		prepare    *sport.Subject
		roundState *roundState
	}{
		{
			name:     "normal case",
			expected: nil,
			prepare: &sport.Subject{
				View:   &sport.View{Round: big.NewInt(0), Sequence: big.NewInt(0)},
				Digest: newTestBlockProposal().Hash(),
			},
			roundState: newTestRoundState(
				&sport.View{Round: big.NewInt(0), Sequence: big.NewInt(0)},
				fullnodeSet,
			),
		},
		{
			name:     "old message",
			expected: errInconsistentSubject,
			prepare: &sport.Subject{
				View:   &sport.View{Round: big.NewInt(0), Sequence: big.NewInt(0)},
				Digest: newTestBlockProposal().Hash(),
			},
			roundState: newTestRoundState(
				&sport.View{Round: big.NewInt(1), Sequence: big.NewInt(1)},
				fullnodeSet,
			),
		},
		{
			name:     "different digest",
			expected: errInconsistentSubject,
			prepare: &sport.Subject{
				View:   &sport.View{Round: big.NewInt(0), Sequence: big.NewInt(0)},
				Digest: cmn.StringToHash("1234567890"),
			},
			roundState: newTestRoundState(
				&sport.View{Round: big.NewInt(1), Sequence: big.NewInt(1)},
				fullnodeSet,
			),
		},
		{
			name:     "malicious package(lack of sequence)",
			expected: errInconsistentSubject,
			prepare: &sport.Subject{
				View:   &sport.View{Round: big.NewInt(0), Sequence: nil},
				Digest: newTestBlockProposal().Hash(),
			},
			roundState: newTestRoundState(
				&sport.View{Round: big.NewInt(1), Sequence: big.NewInt(1)},
				fullnodeSet,
			),
		},
		{
			name:     "wrong PREPARE message with same sequence but different round",
			expected: errInconsistentSubject,
			prepare: &sport.Subject{
				View:   &sport.View{Round: big.NewInt(1), Sequence: big.NewInt(0)},
				Digest: newTestBlockProposal().Hash(),
			},
			roundState: newTestRoundState(
				&sport.View{Round: big.NewInt(0), Sequence: big.NewInt(0)},
				fullnodeSet,
			),
		},
		{
			name:     "wrong PREPARE message with same round but different sequence",
			expected: errInconsistentSubject,
			prepare: &sport.Subject{
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

			if err := c.verifyPrepare(test.prepare, peer); err != nil {
				if err != test.expected {
					t.Errorf("result %d: error mismatch: have %v, want %v", i, err, test.expected)
				}
			}
		})
	}
}
