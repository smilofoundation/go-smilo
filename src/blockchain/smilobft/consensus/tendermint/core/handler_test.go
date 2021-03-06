package core

import (
	"context"
	"go-smilo/src/blockchain/smilobft/core/types"
	"math/big"
	"testing"

	"github.com/golang/mock/gomock"
	"golang.org/x/sync/errgroup"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
)

func TestHandleCheckedMessage(t *testing.T) {
	committeeSet, keysMap := newTestCommitteeSetWithKeys(4)
	currentValidator, _ := committeeSet.GetByIndex(0)
	sender, _ := committeeSet.GetByIndex(1)
	senderKey := keysMap[sender.Address]

	createPrevote := func(round int64, height int64) *Message {
		vote := &Vote{
			Round:             round,
			Height:            big.NewInt(height),
			ProposedBlockHash: common.BytesToHash([]byte{0x1}),
		}
		encoded, err := rlp.EncodeToBytes(&vote)
		if err != nil {
			t.Fatalf("could not encode vote")
		}
		return &Message{
			Code:    msgPrevote,
			Msg:     encoded,
			Address: sender.Address,
		}
	}

	createPrecommit := func(round int64, height int64) *Message {
		vote := &Vote{
			Round:             round,
			Height:            big.NewInt(height),
			ProposedBlockHash: common.BytesToHash([]byte{0x1}),
		}
		encoded, err := rlp.EncodeToBytes(&vote)
		if err != nil {
			t.Fatalf("could not encode vote")
		}
		data := PrepareCommittedSeal(common.BytesToHash([]byte{0x1}), vote.Round, vote.Height)
		hashData := crypto.Keccak256(data)
		commitSign, err := crypto.Sign(hashData, senderKey)
		if err != nil {
			t.Fatalf("error signing")
		}
		return &Message{
			Code:          msgPrecommit,
			Msg:           encoded,
			Address:       sender.Address,
			CommittedSeal: commitSign,
		}
	}

	cases := []struct {
		round   int64
		height  *big.Int
		step    Step
		message *Message
		outcome error
	}{
		{
			1,
			big.NewInt(2),
			propose,
			createPrevote(1, 2),
			errFutureStepMessage,
		},
		{
			1,
			big.NewInt(2),
			propose,
			createPrevote(2, 2),
			errFutureRoundMessage,
		},
		{
			0,
			big.NewInt(2),
			propose,
			createPrevote(0, 3),
			errFutureHeightMessage,
		},
		{
			0,
			big.NewInt(2),
			prevote,
			createPrevote(0, 2),
			nil,
		},
		{
			0,
			big.NewInt(2),
			precommit,
			createPrecommit(0, 2),
			nil,
		},
		{
			0,
			big.NewInt(5),
			precommit,
			createPrecommit(0, 10),
			errFutureHeightMessage,
		},
		{
			5,
			big.NewInt(2),
			precommit,
			createPrecommit(20, 2),
			errFutureRoundMessage,
		},
	}

	for _, testCase := range cases {
		logger := log.New("backend", "test", "id", 0)
		message := newMessagesMap()
		engine := core{
			logger:            logger,
			address:           currentValidator.Address,
			backlogs:          make(map[types.CommitteeMember]*prque.Prque),
			round:             testCase.round,
			height:            testCase.height,
			step:              testCase.step,
			futureRoundChange: make(map[int64]map[common.Address]uint64),
			messages:          message,
			curRoundMessages:  message.getOrCreate(0),
			committeeSet:      committeeSet,
			proposeTimeout:    newTimeout(propose, logger),
			prevoteTimeout:    newTimeout(prevote, logger),
			precommitTimeout:  newTimeout(precommit, logger),
		}

		err := engine.handleCheckedMsg(context.Background(), testCase.message, sender)

		if err != testCase.outcome {
			t.Fatal("unexpected handlecheckedmsg returning ",
				"err=", err, ", expecting=", testCase.outcome, " with msgCode=", testCase.message.Code)
		}

		if err != nil {
			backlogValue, _ := engine.backlogs[sender].Pop()
			msg := backlogValue.(*Message)
			if msg != testCase.message {
				t.Fatal("unexpected backlog message")
			}
		}
	}
}

func TestCoreStopDoesntPanic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	addr := common.HexToAddress("0x0123456789")

	backendMock := NewMockBackend(ctrl)
	backendMock.EXPECT().Address().AnyTimes().Return(addr)

	c := New(backendMock, nil)
	if err := c.Stop(); err != nil {
		t.Fatal(err)
	}
}

func TestCoreMultipleStopsDontPanic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	addr := common.HexToAddress("0x0123456789")

	backendMock := NewMockBackend(ctrl)
	backendMock.EXPECT().Address().AnyTimes().Return(addr)

	c := New(backendMock, nil)
	if err := c.Stop(); err != nil {
		t.Fatal(err)
	}

	if err := c.Stop(); err != nil {
		t.Fatal(err)
	}
}

func TestCoreMultipleConcurrentStopsDontPanic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	addr := common.HexToAddress("0x0123456789")

	backendMock := NewMockBackend(ctrl)
	backendMock.EXPECT().Address().AnyTimes().Return(addr)

	c := New(backendMock, nil)

	wg := errgroup.Group{}
	for i := 0; i < 10; i++ {
		wg.Go(c.Stop)
	}

	if err := wg.Wait(); err != nil {
		t.Fatal(err)
	}
}
