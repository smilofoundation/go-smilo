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
package sport

import "math/big"

type SpeakerPolicy uint64

const (
	RoundRobin SpeakerPolicy = iota
	Lottery
)

type Config struct {
	RequestTimeout       uint64        `toml:",omitempty"` // The timeout for each Sport round in milliseconds
	MaxTimeout           uint64        `toml:",omitempty"` // The Max Timeout for each Sport round in milliseconds
	BlockPeriod          uint64        `toml:",omitempty"` // Default minimum difference between two consecutive block's timestamps in second
	SpeakerPolicy        SpeakerPolicy `toml:",omitempty"` // The policy for speaker selection
	Epoch                uint64        `toml:",omitempty"` // The number of blocks after which to checkpoint and reset the pending votes
	DataDir              string        `toml:",omitempty"` // The default datadir for permissioned-nodes.json file
	MinFunds             int64         `toml:",omitempty"` // The minimum funds a node should have to be a full node
	CommunityAddress     string        `toml:",omitempty"` // The community address for miner donations
	MinBlocksEmptyMining *big.Int      `toml:",omitempty"` // Min Blocks to mine before Stop Mining Empty Blocks
}

var DefaultConfig = &Config{
	RequestTimeout:       10000,
	MaxTimeout:           60,
	BlockPeriod:          1,
	SpeakerPolicy:        Lottery,
	Epoch:                30000,
	MinFunds:             1,
	MinBlocksEmptyMining: big.NewInt(20000000),
}
