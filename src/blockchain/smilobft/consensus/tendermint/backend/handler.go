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

package backend

import (
	"bytes"
	"context"
	"errors"
	"go-smilo/src/blockchain/smilobft/consensus"
	"go-smilo/src/blockchain/smilobft/consensus/tendermint/events"
	"go-smilo/src/blockchain/smilobft/core/types"
	"go-smilo/src/blockchain/smilobft/p2p"
	"io"
	"io/ioutil"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	lru "github.com/hashicorp/golang-lru"
)

const (
	tendermintMsg     = 0x11
	tendermintSyncMsg = 0x12
	NewBlockMsg       = 0x07
)

type UnhandledMsg struct {
	addr common.Address
	msg  p2p.Msg
}

var (
	// errDecodeFailed is returned when decode message fails
	errDecodeFailed = errors.New("fail to decode tendermint message")
)

// Protocol implements consensus.Handler.Protocol
func (sb *Backend) Protocol() (protocolName string, extraMsgCodes uint64) {
	return "tendermint", 2 //nolint
}

func (sb *Backend) HandleUnhandledMsgs(ctx context.Context) {
	for unhandled := sb.pendingMessages.Dequeue(); unhandled != nil; unhandled = sb.pendingMessages.Dequeue() {
		select {
		case <-ctx.Done():
			return
		default:
			// nothing to do
		}

		addr := unhandled.(UnhandledMsg).addr
		msg := unhandled.(UnhandledMsg).msg
		if _, err := sb.HandleMsg(addr, msg); err != nil {
			sb.logger.Error("could not handle cached message", "err", err)
		}
	}
}

// HandleMsg implements consensus.Handler.HandleMsg
func (sb *Backend) HandleMsg(addr common.Address, msg p2p.Msg) (bool, error) {
	if msg.Code != tendermintMsg && msg.Code != tendermintSyncMsg {
		return false, nil
	}

	sb.coreMu.Lock()
	defer sb.coreMu.Unlock()

	//if msg.Code != tendermintMsg && msg.Code != tendermintSyncMsg {
	//	return false, nil
	//}

	if msg.Code == tendermintMsg {
		if !sb.coreStarted {
			buffer := new(bytes.Buffer)
			if _, err := io.Copy(buffer, msg.Payload); err != nil {
				return true, errDecodeFailed
			}
			savedMsg := msg
			savedMsg.Payload = buffer
			sb.pendingMessages.Enqueue(UnhandledMsg{addr: addr, msg: savedMsg})
			return true, nil //return nil to avoid shutting down connection during block sync.
		}

		var data []byte
		if err := msg.Decode(&data); err != nil {
			return true, errDecodeFailed
		}

		hash := types.RLPHash(data)

		// Mark peer's message
		ms, ok := sb.recentMessages.Get(addr)
		var m *lru.ARCCache
		if ok {
			m, _ = ms.(*lru.ARCCache)
		} else {
			m, _ = lru.NewARC(inmemoryMessages)
			sb.recentMessages.Add(addr, m)
		}
		m.Add(hash, true)

		// Mark self known message
		if _, ok := sb.knownMessages.Get(hash); ok {
			return true, nil
		}
		sb.knownMessages.Add(hash, true)

		go func() {
			err := sb.eventMux.Post(events.MessageEvent{
				Payload: data,
			})
			if err != nil {
				log.Error("Could not send sb.eventMux.Post, tendermintMsg", "err", err, "msg", msg.String())
			} else {
				log.Error("Sent sb.eventMux.Post, tendermintMsg", "msg", msg.String())
			}
		}()
		return true, nil
	}
	if msg.Code == tendermintSyncMsg {
		if !sb.coreStarted {
			sb.logger.Info("Sync message received but core not running")
			return true, nil // we return nil as we don't want to shutdown the connection if core is stopped
		}
		sb.logger.Info("Received sync message", "from", addr)
		go func() {
			err := sb.eventMux.Post(events.SyncEvent{Addr: addr})
			if err != nil {
				log.Error("Could not send sb.eventMux.Post, tendermintSyncMsg", "err", err, "msg", msg.String())
			} else {
				log.Error("Sent sb.eventMux.Post, tendermintSyncMsg", "msg", msg.String())
			}
		}()

		return true, nil
	}

	if msg.Code == NewBlockMsg {
		log.Debug("Tendermint received NewBlockMsg", "size", msg.Size, "payload.type", reflect.TypeOf(msg.Payload), "sender", addr, "msg", msg.String())
		if reader, ok := msg.Payload.(*bytes.Reader); ok {
			payload, err := ioutil.ReadAll(reader)
			if err != nil {
				return true, err
			}
			reader.Reset(payload)
			defer reader.Reset(payload)
			var request struct {
				Block *types.Block
				TD    *big.Int
			}
			if err := msg.Decode(&request); err != nil {
				log.Debug("Speaker was unable to decode the NewBlockMsg", "error", err, "msg", msg.String())
				return false, nil
			}
			newRequestedBlock := request.Block
			if newRequestedBlock.Header().MixDigest == types.TendermintDigest {
				log.Debug("Speaker already proposed this block", "hash", newRequestedBlock.Hash(), "sender", addr, "msg", msg.String())
				return true, nil
			} else {
				log.Debug("newRequestedBlock.Header().MixDigest != types.BFTDigest", "MixDigest", newRequestedBlock.Header().MixDigest, "TendermintDigest", types.TendermintDigest)
			}
		}
	} else {
		log.Debug("Tendermint received other msg", "msg.Code", msg.Code, "size", msg.Size, "payload.type", reflect.TypeOf(msg.Payload), "sender", addr, "msg", msg.String())

	}

	return false, nil
}

// SetBroadcaster implements consensus.Handler.SetBroadcaster
func (sb *Backend) SetBroadcaster(broadcaster consensus.Broadcaster) {
	sb.broadcaster = broadcaster
}

func (sb *Backend) NewChainHead() error {
	sb.coreMu.RLock()
	defer sb.coreMu.RUnlock()
	if !sb.coreStarted {
		return ErrStoppedEngine
	}
	go sb.Post(events.CommitEvent{})
	return nil
}
