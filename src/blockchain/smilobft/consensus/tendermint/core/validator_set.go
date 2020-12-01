package core

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"

	"go-smilo/src/blockchain/smilobft/consensus/tendermint/config"
	"go-smilo/src/blockchain/smilobft/consensus/tendermint/committee"
)

type validatorSet struct {
	sync.RWMutex
	committee.Set
}

func (v *validatorSet) set(valSet committee.Set) {
	v.Lock()
	v.Set = valSet
	v.Unlock()
}

func (v *validatorSet) Size() int {
	v.RLock()
	defer v.RUnlock()
	if v.Set == nil {
		return 0
	}
	size := v.Set.Size()
	return size
}

func (v *validatorSet) List() []committee.Validator {
	v.RLock()
	defer v.RUnlock()
	if v.Set == nil {
		return nil
	}

	list := v.Set.List()
	return list
}

func (v *validatorSet) GetByIndex(i uint64) committee.Validator {
	v.RLock()
	defer v.RUnlock()
	if v.Set == nil {
		return nil
	}
	val := v.Set.GetByIndex(i)
	return val
}

func (v *validatorSet) GetByAddress(addr common.Address) (int, committee.Validator) {
	v.RLock()
	defer v.RUnlock()
	if v.Set == nil {
		return -1, nil
	}
	i, val := v.Set.GetByAddress(addr)
	return i, val
}

func (v *validatorSet) GetProposer() committee.Validator {
	v.RLock()
	defer v.RUnlock()
	if v.Set == nil {
		return nil
	}
	val := v.Set.GetProposer()
	return val
}

func (v *validatorSet) Copy() committee.Set {
	v.RLock()
	defer v.RUnlock()
	if v.Set == nil {
		return nil
	}
	valSet := v.Set.Copy()
	return valSet
}

func (v *validatorSet) Policy() config.ProposerPolicy {
	v.RLock()
	defer v.RUnlock()
	if v.Set == nil {
		return 0
	}
	policy := v.Set.Policy()
	return policy
}

func (v *validatorSet) CalcProposer(lastProposer common.Address, round uint64) {
	v.RLock()
	defer v.RUnlock()
	if v.Set == nil {
		return
	}
	v.Set.CalcProposer(lastProposer, round)
}

func (v *validatorSet) IsProposer(address common.Address) bool {
	v.RLock()
	defer v.RUnlock()
	if v.Set == nil {
		return false
	}

	return v.Set.IsProposer(address)
}

func (v *validatorSet) AddValidator(address common.Address) bool {
	v.RLock()
	defer v.RUnlock()
	if v.Set == nil {
		return false
	}

	return v.Set.AddValidator(address)
}

func (v *validatorSet) RemoveValidator(address common.Address) bool {
	v.RLock()
	defer v.RUnlock()
	if v.Set == nil {
		return false
	}
	return v.Set.RemoveValidator(address)
}
