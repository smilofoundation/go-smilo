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
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"go-smilo/src/blockchain/smilobft/cmn"
	"go-smilo/src/blockchain/smilobft/consensus/sport"
	"go-smilo/src/blockchain/smilobft/consensus/sport/fullnode"
)

func TestHandlePrepare(t *testing.T) {
	N := uint64(4)
	F := uint64(1)
	E := uint64(1)

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
			"normal case",
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
						c.state = StatePreprepared
					}
				}
				return sys
			}(),
			nil,
		},
		{
			"future message",
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
			"subject not match",
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
			"subject not match",
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
			"less than 2F+E",
			func() *testSystem {
				sys := NewTestSystemWithBackend(N)

				// save less than 2*F+E replica
				sys.backends = sys.backends[2*int(F)+int(E):]

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
		// TODO: double send message
	}

	for _, test := range testCases {
		test.system.Run(false)

		t.Run(test.name, func(t *testing.T) {

			v0 := test.system.backends[0]
			r0 := v0.engine.(*core)

			for i, v := range test.system.backends {
				thisfullnode := r0.fullnodeSet.GetByIndex(uint64(i))
				m, _ := Encode(v.engine.(*core).current.Subject())
				if err := r0.handlePrepare(&message{
					Code:    msgPrepare,
					Msg:     m,
					Address: thisfullnode.Address(),
				}, thisfullnode); err != nil {
					if err != test.expectedErr {
						t.Errorf("error mismatch: have %v, want %v", err, test.expectedErr)
					}
					if r0.current.IsHashLocked() {
						t.Errorf("block should not be locked")
					}
					return
				}
			}

			// core should have 2F+E PREPARE messages
			//2F+E
			MinApprovers := r0.fullnodeSet.MinApprovers()

			// prepared is normal case
			if r0.state != StatePrepared {
				// There are not enough PREPARE messages in core
				if r0.state != StatePreprepared {
					t.Errorf("state mismatch: have %v, want %v", r0.state, StatePreprepared)
				}
				if r0.current.Prepares.Size() >= MinApprovers {
					t.Errorf("the size of PREPARE messages should be less than %v", MinApprovers)
				}
				if r0.current.IsHashLocked() {
					t.Errorf("block should not be locked")
				}

				return
			}

			if r0.current.Prepares.Size() < MinApprovers {
				t.Errorf("the size of PREPARE messages should be larger than 2F+E: size %v", r0.current.Commits.Size())
			}

			// a message will be delivered to backend if 2F+E
			if int64(len(v0.sentMsgs)) != 1 {
				t.Errorf("the Send() should be called once: times %v", len(test.system.backends[0].sentMsgs))
			}

			// verify COMMIT messages
			decodedMsg := new(message)
			err := decodedMsg.FromPayload(v0.sentMsgs[0], nil)
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

// round is not checked for now
func TestVerifyPrepare(t *testing.T) {
	// for log purpose
	privateKey, _ := crypto.GenerateKey()
	peer := fullnode.NewFullNode(getPublicKeyAddress(privateKey))
	fullnodeSet := fullnode.NewFullnodeSet([]common.Address{peer.Address()}, sport.RoundRobin)

	N := uint64(1)

	sys := NewTestSystemWithBackend(N)

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
