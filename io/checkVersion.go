////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"github.com/pkg/errors"
	"strconv"
	"strings"
)

type clientVersion struct {
	major int
	minor int
	patch string
}

func (v *clientVersion) String() string {
	return strconv.Itoa(v.major) + "." + strconv.Itoa(v.minor) + "." + v.patch
}

func parseClientVersion(versionString string) (*clientVersion, error) {
	versions := strings.SplitN(versionString, ".", 3)
	if len(versions) != 3 {
		return nil, errors.New("Client version string must contain a major, minor, and patch version separated by \".\"")
	}
	major, err := strconv.Atoi(versions[0])
	if err != nil {
		return nil, errors.New("Major client version couldn't be parsed as integer")
	}
	minor, err := strconv.Atoi(versions[1])
	if err != nil {
		return nil, errors.New("Minor client version couldn't be parsed as integer")
	}
	return &clientVersion{
		major: major,
		minor: minor,
		patch: versions[2],
	}, nil
}

// Handle client version check
// Example valid version strings:
// 0.1.0
// 1.3.0-ff81cdae
// Major and minor versions should both be numbers, and patch versions can be
// anything, but they must be present
// receiver is the version from the registration server
func (v *clientVersion) isCompatible(ourVersion *clientVersion) bool {
	// Compare major version: must be equal to be deemed compatible
	if ourVersion.major != v.major {
		return false
	}
	// Compare minor version: our version must be greater than or equal to their version to be deemed compatible
	if ourVersion.minor < v.minor {
		return false
	}
	// Patch versions aren't supposed to affect compatibility, so they're ignored for the check

	return true
}
