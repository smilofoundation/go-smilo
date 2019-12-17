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

package backend

import (
	"bytes"
	"io/ioutil"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	lru "github.com/hashicorp/golang-lru"

	"github.com/ethereum/go-ethereum/log"

	"go-smilo/src/blockchain/smilobft/consensus"
	"go-smilo/src/blockchain/smilobft/consensus/sportdao"
	"go-smilo/src/blockchain/smilobft/core/types"
	"go-smilo/src/blockchain/smilobft/p2p"
)

// HandleMsg implements consensus.Handler.HandleMsg
func (sb *Backend) HandleMsg(addr common.Address, msg p2p.Msg) (bool, error) {
	sb.coreMu.Lock()
	defer sb.coreMu.Unlock()

	if msg.Code == smilobftMsg {
		//log.Debug("backend/backend_handler_sportdao.go, HandleMsg(), Gossip, smilobftMsg message, GOT IT!!! ", "msg", msg.String())
		if !sb.coreStarted {
			return true, sportdao.ErrStoppedEngine
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
			err := sb.smilobftEventMux.Post(sportdao.MessageEvent{
				Payload: data,
			})
			if err != nil {
				log.Error("Could not send sb.smilobftEventMux.Post, sportdao.MessageEvent", "err", err, "msg", msg.String())
			} else {
				log.Error("Sent sb.smilobftEventMux.Post, sportdao.MessageEvent", "msg", msg.String())
			}
		}()

		return true, nil
	}
	if msg.Code == NewBlockMsg && sb.core.IsSpeaker() {
		// avoid race conditions
		log.Debug("Speaker received NewBlockMsg", "size", msg.Size, "payload.type", reflect.TypeOf(msg.Payload), "sender", addr, "msg", msg.String())
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
			if newRequestedBlock.Header().MixDigest == types.SportDigest && sb.core.IsCurrentBlockProposal(newRequestedBlock.Hash()) {
				log.Debug("Speaker already proposed this block", "hash", newRequestedBlock.Hash(), "sender", addr, "msg", msg.String())
				return true, nil
			}
		}
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
		return sportdao.ErrStoppedEngine
	}
	go func() {
		err := sb.smilobftEventMux.Post(sportdao.FinalCommittedEvent{})
		if err != nil {
			log.Error("NewChainHead, Could not send FinalCommittedEvent, ", "err", err)
		}

	}()
	return nil
}
