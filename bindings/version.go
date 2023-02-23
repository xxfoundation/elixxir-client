////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// version.go contains functions to report the client version.

package bindings

import "C"

import "gitlab.com/elixxir/client/v4/xxdk"

// GetVersion returns the xxdk.SEMVER.
//
//export GetVersion
func GetVersion() string {
	return xxdk.SEMVER
}

// GetGitVersion returns the xxdk.GITVERSION.
//
//export GetGitVersion
func GetGitVersion() string {
	return xxdk.GITVERSION
}

// GetDependencies returns the xxdk.DEPENDENCIES.
//
//export GetDependencies
func GetDependencies() string {
	return xxdk.DEPENDENCIES
}
