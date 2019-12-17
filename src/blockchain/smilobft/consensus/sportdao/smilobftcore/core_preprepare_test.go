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

	"go-smilo/src/blockchain/smilobft/consensus/sportdao"
)

func newTestPreprepare(v *sportdao.View) *sportdao.Preprepare {
	return &sportdao.Preprepare{
		View:          v,
		BlockProposal: newTestBlockProposal(),
	}
}

func TestHandlePreprepare(t *testing.T) {
	availableNodes := 4 // replica 0 is the speaker, it will send messages to others

	testCases := []struct {
		name            string
		system          *testSystem
		expectedRequest sportdao.BlockProposal
		expectedErr     error
		existingBlock   bool
	}{
		{
			"normal case",
			func() *testSystem {
				sys := NewTestSystemWithBackend(availableNodes)

				for i, backend := range sys.backends {
					c := backend.engine.(*core)
					c.fullnodeSet = backend.peers
					if i != 0 {
						c.state = StateAcceptRequest
					}
				}
				return sys
			}(),
			newTestBlockProposal(),
			nil,
			false,
		},
		{
			"future message",
			func() *testSystem {
				sys := NewTestSystemWithBackend(availableNodes)

				for i, backend := range sys.backends {
					c := backend.engine.(*core)
					c.fullnodeSet = backend.peers
					if i != 0 {
						c.state = StateAcceptRequest
						// hack: force set subject that future message can be simulated
						c.current = newTestRoundState(
							&sportdao.View{
								Round:    big.NewInt(0),
								Sequence: big.NewInt(0),
							},
							c.fullnodeSet,
						)

					} else {
						c.current.SetSequence(big.NewInt(10))
					}
				}
				return sys
			}(),
			makeBlock(1),
			errFutureMessage,
			false,
		},
		{
			"non-speaker",
			func() *testSystem {
				sys := NewTestSystemWithBackend(availableNodes)

				// force remove replica 0, let replica 1 be the speaker
				sys.backends = sys.backends[1:]

				for i, backend := range sys.backends {
					c := backend.engine.(*core)
					c.fullnodeSet = backend.peers
					if i != 0 {
						// replica 0 is the speaker
						c.state = StatePreprepared
					}
				}
				return sys
			}(),
			makeBlock(1),
			errNotFromSpeaker,
			false,
		},
		{
			"errOldMessage",
			func() *testSystem {
				sys := NewTestSystemWithBackend(availableNodes)

				for i, backend := range sys.backends {
					c := backend.engine.(*core)
					c.fullnodeSet = backend.peers
					if i != 0 {
						c.state = StatePreprepared
						c.current.SetSequence(big.NewInt(10))
						c.current.SetRound(big.NewInt(10))
					}
				}
				return sys
			}(),
			makeBlock(1),
			errOldMessage,
			false,
		},
	}

	for _, test := range testCases {

		t.Run(test.name, func(t *testing.T) {
			test.system.Run(false)

			v0 := test.system.backends[0]
			r0 := v0.engine.(*core)

			curView := r0.currentView()

			preprepare := &sportdao.Preprepare{
				View:          curView,
				BlockProposal: test.expectedRequest,
			}

			for i, v := range test.system.backends {
				// i == 0 is primary backend, it is responsible for send PRE-PREPARE messages to others.
				if i == 0 {
					continue
				}

				c := v.engine.(*core)

				m, _ := Encode(preprepare)
				_, val := r0.fullnodeSet.GetByAddress(v0.Address())
				// run each backends and verify handlePreprepare function.
				if err := c.handlePreprepare(&message{
					Code:    msgPreprepare,
					Msg:     m,
					Address: v0.Address(),
				}, val); err != nil {
					if err != test.expectedErr {
						t.Errorf("test %s, error mismatch: have %v, want %v", test.name, err, test.expectedErr)
					}
					return
				}

				if c.state != StatePreprepared {
					t.Errorf("test %s, state mismatch: have %v, want %v", test.name, c.state, StatePreprepared)
				}

				if !test.existingBlock && !reflect.DeepEqual(c.current.Subject().View, curView) {
					t.Errorf("test %s, view mismatch: have %v, want %v", test.name, c.current.Subject().View, curView)
				}

				// verify prepare messages
				decodedMsg := new(message)
				err := decodedMsg.FromPayload(v.sentMsgs[0], nil)
				if err != nil {
					t.Errorf("test %s, error mismatch: have %v, want nil", test.name, err)
				}

				expectedCode := msgPrepare
				if test.existingBlock {
					expectedCode = msgCommit
				}
				if decodedMsg.Code != expectedCode {
					t.Errorf("test %s, message code mismatch: have %v, want %v", test.name, decodedMsg.Code, expectedCode)
				}

				var subject *sportdao.Subject
				err = decodedMsg.Decode(&subject)
				if err != nil {
					t.Errorf("test %s, error mismatch: have %v, want nil", test.name, err)
				}
				if !test.existingBlock && !reflect.DeepEqual(subject, c.current.Subject()) {
					t.Errorf("test %s, subject mismatch: have %v, want %v", test.name, subject, c.current.Subject())
				}

			}
		})
	}
}

func TestHandlePreprepareWithLock(t *testing.T) {
	availableNodes := 4 // replica 0 is the speaker, it will send messages to others
	proposal := newTestBlockProposal()
	mismatchBlockProposal := makeBlock(10)
	newSystem := func() *testSystem {
		sys := NewTestSystemWithBackend(availableNodes)

		for i, backend := range sys.backends {
			c := backend.engine.(*core)
			c.fullnodeSet = backend.peers
			if i != 0 {
				c.state = StateAcceptRequest
			}
			c.roundChangeSet = newRoundChangeSet(c.fullnodeSet)
		}
		return sys
	}

	testCases := []struct {
		name              string
		system            *testSystem
		proposal          sportdao.BlockProposal
		lockBlockProposal sportdao.BlockProposal
	}{
		{
			"normal proposal",
			newSystem(),
			proposal,
			proposal,
		},
		{
			"mismatch proposal",
			newSystem(),
			proposal,
			mismatchBlockProposal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			test.system.Run(false)
			v0 := test.system.backends[0]
			r0 := v0.engine.(*core)
			curView := r0.currentView()
			preprepare := &sportdao.Preprepare{
				View:          curView,
				BlockProposal: test.proposal,
			}
			lockPreprepare := &sportdao.Preprepare{
				View:          curView,
				BlockProposal: test.lockBlockProposal,
			}

			for i, v := range test.system.backends {
				// i == 0 is primary backend, it is responsible for send PRE-PREPARE messages to others.
				if i == 0 {
					continue
				}

				c := v.engine.(*core)
				c.current.SetPreprepare(lockPreprepare)
				c.current.LockHash()
				m, _ := Encode(preprepare)
				_, val := r0.fullnodeSet.GetByAddress(v0.Address())
				if err := c.handlePreprepare(&message{
					Code:    msgPreprepare,
					Msg:     m,
					Address: v0.Address(),
				}, val); err != nil {
					t.Errorf("test %s, error mismatch: have %v, want nil", test.name, err)
				}
				if test.proposal == test.lockBlockProposal {
					if c.state != StatePrepared {
						t.Errorf("test %s, state mismatch: have %v, want %v", test.name, c.state, StatePreprepared)
					}
					if !reflect.DeepEqual(curView, c.currentView()) {
						t.Errorf("test %s, view mismatch: have %v, want %v", test.name, c.currentView(), curView)
					}
				} else {
					// Should stay at StateAcceptRequest
					if c.state != StateAcceptRequest {
						t.Errorf("test %s, state mismatch: have %v, want %v", test.name, c.state, StateAcceptRequest)
					}
					// Should have triggered a round change
					expectedView := &sportdao.View{
						Sequence: curView.Sequence,
						Round:    big.NewInt(1),
					}
					if !reflect.DeepEqual(expectedView, c.currentView()) {
						t.Errorf("test %s, view mismatch: have %v, want %v", test.name, c.currentView(), expectedView)
					}
				}
			}
		})
	}
}
