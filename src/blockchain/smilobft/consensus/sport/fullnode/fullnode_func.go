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
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"go-smilo/src/blockchain/smilobft/consensus/sport/fullnode/vrf"
	"go-smilo/src/blockchain/smilobft/swarm/log"

	"fmt"
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

func lotterySpeaker(fullnodeSet sport.FullnodeSet, nodepk *ecdsa.PrivateKey, blockHash string) sport.Fullnode {
	if fullnodeSet.Size() == 0 {
		return nil
	}

	participantsJson, err := json.Marshal(fullnodeSet)
	if err != nil {
		log.Error("Could not create list of participants for lottery ", err)
		return nil
	}

	if nodepk == nil {
		log.Error("Could not create list of participants for lottery, PK is nil", err)
		return nil
	}

	pkhex := crypto.FromECDSA(nodepk)
	if pkhex == nil || len(pkhex) == 0 {
		log.Error("Could not create list of participants for lottery, PK FromECDSA is invalid", err)
		return nil
	}

	keyStr := hex.EncodeToString(pkhex)
	if keyStr == "" || len(pkhex) == 0 {
		log.Error("Could not create list of participants for lottery, PK EncodeToString is invalid", err)
		return nil
	}

	////keyStr := fmt.Sprintf("%x", nodepk.D.Bytes())
	log.Debug("Going to lotterySpeaker .... ", "key", keyStr)
	//
	skb := vrf.PrivateKey(keyStr)

	//skb, _ := vrf.GenerateKey(nil)
	//log.Debug("Going to lotterySpeaker for real .... ", "key", hex.EncodeToString(skb))

	provableMessage := append(participantsJson, []byte("\n"+fmt.Sprintf("%s", blockHash))...)
	vrfBytes, proof := skb.Prove(provableMessage)
	pk, _ := skb.Public()
	verifyResult, _ := pk.Verify(provableMessage, proof)
	if !verifyResult {
		log.Error("Proof lottery verification has failed")
		return nil
	}

	winners := vrf.PickUniquePseudorandomParticipants(vrfBytes[:], 1, fullnodeSet.List())
	var firstWinner sport.Fullnode

	if len(winners) > 0 {
		firstWinner = winners[0]
		firstWinner.SetLotteryTicket(proof, provableMessage)
	}

	return firstWinner
}
