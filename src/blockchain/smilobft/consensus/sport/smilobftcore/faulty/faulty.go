// Copyright 2017 AMIS Technologies
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

package faulty

import (
	"math/rand"

	"go-smilo/src/blockchain/smilobft/consensus/sport"
)

func (c *core) random() bool {
	return c.config.FaultyMode == sport.Random.Uint64() && rand.Intn(2) == 1
}

func (c *core) notBroadcast() bool {
	return c.config.FaultyMode == sport.NotBroadcast.Uint64() || c.random()
}

func (c *core) sendWrongMsg() bool {
	return c.config.FaultyMode == sport.SendWrongMsg.Uint64() || c.random()
}

func (c *core) modifySig() bool {
	return c.config.FaultyMode == sport.ModifySig.Uint64() || c.random()
}

func (c *core) alwaysPropose() bool {
	return c.config.FaultyMode == sport.AlwaysPropose.Uint64() || c.random()
}

func (c *core) alwaysRoundChange() bool {
	return c.config.FaultyMode == sport.AlwaysRoundChange.Uint64() || c.random()
}
