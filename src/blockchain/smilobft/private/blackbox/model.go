// Copyright 2019 The go-smilo Authors
// Copyright 2016 The go-ethereum Authors
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

package blackbox

import (
	"errors"
	"net/http"

	"github.com/BurntSushi/toml"
	"github.com/patrickmn/go-cache"
)

var (
	ErrBlackboxIsNotStarted = errors.New("blackbox is not started")
)

// --------------------------------------------------------------------

type Blackbox struct {
	node               *Client
	cache              *cache.Cache
	isBlackboxNotInUse bool
}

// --------------------------------------------------------------------

type Client struct {
	httpClient *http.Client
}

func CreateClient(socketPath string) (*Client, error) {
	return &Client{
		httpClient: unixClient(socketPath),
	}, nil
}

// --------------------------------------------------------------------

type Config struct {
	Socket  string `toml:"socket"`
	WorkDir string `toml:"workdir"`

	// Deprecated
	SocketPath string `toml:"socketPath"`
}

func LoadConfig(configPath string) (*Config, error) {
	cfg := new(Config)
	if _, err := toml.DecodeFile(configPath, cfg); err != nil {
		return nil, err
	}
	// Fall back to Blackbox 0.0.1 config format if necessary
	if cfg.Socket == "" {
		cfg.Socket = cfg.SocketPath
	}
	return cfg, nil
}

// --------------------------------------------------------------------
