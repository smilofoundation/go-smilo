package committee

import (
	"github.com/ethereum/go-ethereum/common"
	"go-smilo/src/blockchain/smilobft/core/types"
)

func roundRobinProposer(valSet Set, proposer common.Address, round int64) types.CommitteeMember {
	size := valSet.Size()
	seed := int(round)
	if proposer != (common.Address{}) {
		seed = calcSeed(valSet, proposer, round) + 1
	}
	pick := seed % size
	selectedProposer, _ := valSet.GetByIndex(pick)
	return selectedProposer
}
