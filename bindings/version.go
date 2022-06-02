////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import "gitlab.com/elixxir/client/api"

// GetVersion returns the api SEMVER
func GetVersion() string {
	return api.SEMVER
}

// GetGitVersion rturns the api GITVERSION
func GetGitVersion() string {
	return api.GITVERSION
}

// GetDependencies returns the api DEPENDENCIES
func GetDependencies() string {
	return api.DEPENDENCIES
}
