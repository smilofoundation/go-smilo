package committee

import (
	"github.com/ethereum/go-ethereum/common"
)

func roundRobinProposer(valSet Set, proposer common.Address, round uint64) Validator {
	size := valSet.Size()
	if size == 0 {
		return nil
	}

	seed := round
	if proposer != (common.Address{}) {
		seed = calcSeed(valSet, proposer, round) + 1
	}

	pick := seed % uint64(size)
	return valSet.GetByIndex(pick)
}
