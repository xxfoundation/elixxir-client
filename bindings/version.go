///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// version.go contains functions to report the client version.

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
