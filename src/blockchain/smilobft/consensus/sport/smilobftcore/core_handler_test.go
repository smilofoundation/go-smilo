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
	"testing"

	"go-smilo/src/blockchain/smilobft/cmn"
	"go-smilo/src/blockchain/smilobft/consensus/sport"
)

// notice: the normal case have been tested in integration tests.
func TestHandleMsg(t *testing.T) {
	N := 4
	sys := NewTestSystemWithBackend(N)

	closer := sys.Run(true)
	defer closer()

	v0 := sys.backends[0]
	r0 := v0.engine.(*core)

	m, _ := Encode(&sport.Subject{
		View: &sport.View{
			Sequence: big.NewInt(0),
			Round:    big.NewInt(0),
		},
		Digest: cmn.StringToHash("1234567890"),
	})
	// with a matched payload. msgPreprepare should match with *sport.Preprepare in normal case.
	msg := &message{
		Code:          msgPreprepare,
		Msg:           m,
		Address:       v0.Address(),
		Signature:     []byte{},
		CommittedSeal: []byte{},
	}

	_, val := v0.Fullnodes(nil).GetByAddress(v0.Address())
	if err := r0.handleCheckedMsg(msg, val); err != errFailedDecodePreprepare {
		t.Errorf("error mismatch: have %v, want %v", err, errFailedDecodePreprepare)
	}

	m, _ = Encode(&sport.Preprepare{
		View: &sport.View{
			Sequence: big.NewInt(0),
			Round:    big.NewInt(0),
		},
		BlockProposal: makeBlock(1),
	})
	// with a unmatched payload. msgPrepare should match with *sport.Subject in normal case.
	msg = &message{
		Code:          msgPrepare,
		Msg:           m,
		Address:       v0.Address(),
		Signature:     []byte{},
		CommittedSeal: []byte{},
	}

	_, val = v0.Fullnodes(nil).GetByAddress(v0.Address())
	if err := r0.handleCheckedMsg(msg, val); err != errFailedDecodePrepare {
		t.Errorf("error mismatch: have %v, want %v", err, errFailedDecodePreprepare)
	}

	m, _ = Encode(&sport.Preprepare{
		View: &sport.View{
			Sequence: big.NewInt(0),
			Round:    big.NewInt(0),
		},
		BlockProposal: makeBlock(2),
	})
	// with a unmatched payload. sport.MsgCommit should match with *sport.Subject in normal case.
	msg = &message{
		Code:          msgCommit,
		Msg:           m,
		Address:       v0.Address(),
		Signature:     []byte{},
		CommittedSeal: []byte{},
	}

	_, val = v0.Fullnodes(nil).GetByAddress(v0.Address())
	if err := r0.handleCheckedMsg(msg, val); err != errFailedDecodeCommit {
		t.Errorf("error mismatch: have %v, want %v", err, errFailedDecodeCommit)
	}

	m, _ = Encode(&sport.Preprepare{
		View: &sport.View{
			Sequence: big.NewInt(0),
			Round:    big.NewInt(0),
		},
		BlockProposal: makeBlock(3),
	})
	// invalid message code. message code is not exists in list
	msg = &message{
		Code:          uint64(99),
		Msg:           m,
		Address:       v0.Address(),
		Signature:     []byte{},
		CommittedSeal: []byte{},
	}

	_, val = v0.Fullnodes(nil).GetByAddress(v0.Address())
	if err := r0.handleCheckedMsg(msg, val); err == nil {
		t.Errorf("error mismatch: have %v, want nil", err)
	}

	// with malicious payload
	if err := r0.handleMsg([]byte{1}); err == nil {
		t.Errorf("error mismatch: have %v, want nil", err)
	}
}
