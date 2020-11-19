package params

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

//Quorum - test key constant values modified by Quorum
func TestQuorumParams(t *testing.T) {
	type data struct {
		actual   uint64
		expected uint64
	}
	var testData = map[string]data{
		"GasLimitBoundDivisor":       {GasLimitBoundDivisor, 1024},
		"MinGasLimit":                {MinGasLimit, 180000000},
		"GenesisGasLimit":            {GenesisGasLimit, 210000000},
		"QuorumMaximumExtraDataSize": {SmiloMaximumExtraDataSize, 65},
		"QuorumMaxPayloadBufferSize": {QuorumMaxPayloadBufferSize, 128},
	}
	for k, v := range testData {
		assert.Equal(t, v.expected, v.actual, k+" value mismatch")
	}
}
