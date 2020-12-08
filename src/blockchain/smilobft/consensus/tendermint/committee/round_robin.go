package committee

import (
	"go-smilo/src/blockchain/smilobft/core/types"

	"github.com/ethereum/go-ethereum/common"
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
