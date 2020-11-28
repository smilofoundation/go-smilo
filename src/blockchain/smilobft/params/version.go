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

package params

import (
	"fmt"
)

const (
	//GETH
	VersionMajor = 1        // Major version component of the current release
	VersionMinor = 9        // Minor version component of the current release
	VersionPatch = 7        // Patch version component of the current release
	VersionMeta  = "stable" // Version metadata to append to the version string

	//Quorum
	QuorumVersionMajor = 2
	QuorumVersionMinor = 6
	QuorumVersionPatch = 0

	//Autonity
	AutonityVersionMajor = 0 // Major version component of the current release
	AutonityVersionMinor = 2 // Minor version component of the current release
	AutonityVersionPatch = 1 // Patch version component of the current release

	//Smilo
	SmiloVersionMajor = 1
	SmiloVersionMinor = 9
	SmiloVersionPatch = 7
	SmiloMinorPatch   = 0
)

// Version holds the textual GETH version string.
var Version = func() string {
	return fmt.Sprintf("%d.%d.%d", VersionMajor, VersionMinor, VersionPatch)
}()

// VersionWithMeta holds the textual GETH version string including the metadata.
var VersionWithMeta = func() string {
	v := Version
	if VersionMeta != "" {
		v += "-" + VersionMeta
	}
	return v
}()

// ArchiveVersion holds the textual version string used for Geth archives.
// e.g. "1.8.11-dea1ce05" for stable releases, or
//      "1.8.13-unstable-21c059b6" for unstable releases
func ArchiveVersion(gitCommit string) string {
	vsn := Version
	if VersionMeta != "stable" {
		vsn += "-" + VersionMeta
	}
	if len(gitCommit) >= 8 {
		vsn += "-" + gitCommit[:8]
	}
	return vsn
}

// SmiloVersion holds the textual Smilo version string.
var SmiloVersion = func() string {
	return fmt.Sprintf("%d.%d.%d.%d", SmiloVersionMajor, SmiloVersionMinor, SmiloVersionPatch, SmiloMinorPatch)
}()

// QuorumVersion holds the textual Quorum version string.
var QuorumVersion = func() string {
	return fmt.Sprintf("%d.%d.%d", QuorumVersionMajor, QuorumVersionMinor, QuorumVersionPatch)
}()

// AutonityVersion holds the textual Autonity version string.
var AutonityVersion = func() string {
	return fmt.Sprintf("%d.%d.%d", AutonityVersionMajor, AutonityVersionMinor, AutonityVersionPatch)
}()

func VersionWithCommit(gitCommit, gitDate string) string {
	vsn := VersionWithMeta
	if len(gitCommit) >= 8 {
		vsn += "-" + gitCommit[:8]
	}
	if (VersionMeta != "stable") && (gitDate != "") {
		vsn += "-" + gitDate
	}
	return vsn
}
