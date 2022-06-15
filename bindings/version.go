////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import "gitlab.com/elixxir/client/xxdk"

// GetVersion returns the api SEMVER
func GetVersion() string {
	return xxdk.SEMVER
}

// GetGitVersion rturns the api GITVERSION
func GetGitVersion() string {
	return xxdk.GITVERSION
}

// GetDependencies returns the api DEPENDENCIES
func GetDependencies() string {
	return xxdk.DEPENDENCIES
}
