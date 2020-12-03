package committee

import (
	"github.com/ethereum/go-ethereum/common"
	"go-smilo/src/blockchain/smilobft/core/types"
)

func stickyProposer(valSet Set, proposer common.Address, round int64) types.CommitteeMember {
	size := valSet.Size()
	seed := int(round)
	if proposer != (common.Address{}) {
		seed = calcSeed(valSet, proposer, round)
	}

	pick := seed % size
	selectedProposer, _ := valSet.GetByIndex(pick)
	return selectedProposer
}

func calcSeed(valSet Set, proposer common.Address, round int64) int {
	offset := 0
	if idx, _, err := valSet.GetByAddress(proposer); err == nil {
		offset = idx
	}
	return offset + int(round)
}
