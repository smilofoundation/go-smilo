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

package fullnode

import (
	"bytes"
	"encoding/json"
	"github.com/btcsuite/btcutil/base58"
	"github.com/ethereum/go-ethereum/common"
	"github.com/flynn/flynn/build/_go.d8625e472d70f10b454a5a6185a5fb04fb39cb60a2c1f5756ae991aa874de58e/src/fmt"
	"go-smilo/src/blockchain/smilobft/consensus/sport/fullnode/vrf"
	"go-smilo/src/blockchain/smilobft/swarm/log"

	"go-smilo/src/blockchain/smilobft/consensus/sport"
)

func calcSeed(fullnodeSet sport.FullnodeSet, speaker common.Address, round uint64) uint64 {
	offset := 0
	if idx, val := fullnodeSet.GetByAddress(speaker); val != nil {
		offset = idx
	}
	return uint64(offset) + round
}

func emptyAddress(addr common.Address) bool {
	return addr == common.Address{}
}

func roundRobinSpeaker(fullnodeSet sport.FullnodeSet, speaker common.Address, round uint64) sport.Fullnode {
	if fullnodeSet.Size() == 0 {
		return nil
	}
	seed := uint64(0)
	if emptyAddress(speaker) {
		seed = round
	} else {
		seed = calcSeed(fullnodeSet, speaker, round) + 1
	}
	pick := seed % uint64(fullnodeSet.Size())
	return fullnodeSet.GetByIndex(pick)
}


func lotterySpeaker(fullnodeSet sport.FullnodeSet, speaker common.Address, round uint64) sport.Fullnode {
	if fullnodeSet.Size() == 0 {
		return nil
	}

	participantsJson, err := json.Marshal(fullnodeSet)
	if err != nil {
		log.Error("Could not create lottery ",err)
		return nil
	}

	skb := vrf.PrivateKey(base58.Decode("test123"))

	provableMessage := append(participantsJson, []byte("\n"+fmt.Sprintf("%s",round))...)
	vrfBytes, proof := skb.Prove(provableMessage)
	pk, _ := skb.Public()
	verifyResult, vrfBytes2 := pk.Verify(provableMessage, proof)
	if !verifyResult || bytes.Compare(vrfBytes, vrfBytes2) != 0 {
		log.Error("Proof verification was failed")
		return nil
	}

	winners := vrf.PickUniquePseudorandomParticipants(vrfBytes[:], 1, fullnodeSet.List())

	firstWinner := winners[0]
	firstWinner.SetLotteryTicket(base58.Encode(proof))

	return firstWinner
}
