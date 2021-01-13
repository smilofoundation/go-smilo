package config

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("got a nil default config")
	}

	if cfg.ProposerPolicy != WeightedRandomSampling {
		t.Fatal("default config is not RoundRobin")
	}
}
