///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package permissioning

import (
	"github.com/pkg/errors"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/version"
	"gitlab.com/xx_network/comms/connect"
)

// GetNetworkVersion contacts the permissioning server and returns the current
// supported client version.
// returns a bool which designates if the network is enforcing versioning
// (not enforcing versioning is mostly a debugging)
// returns the version and an error if problems arise
func (perm *Permissioning) GetNetworkVersion() (bool, version.Version, error) {
	return getRemoteVersion(perm.host, perm.comms)
}

type getRemoteClientVersionComms interface {
	SendGetCurrentClientVersionMessage(host *connect.Host) (*pb.ClientVersion, error)
}

// getRemoteVersion contacts the permissioning server and returns the current
// supported client version.
func getRemoteVersion(permissioningHost *connect.Host, comms getRemoteClientVersionComms) (bool, version.Version, error) {
	//gets the remote version
	response, err := comms.SendGetCurrentClientVersionMessage(
		permissioningHost)
	if err != nil {
		return false, version.Version{}, errors.WithMessage(err,
			"Failed to get minimum client version from network")
	}
	if response.Version == "" {
		return false, version.Version{}, nil
	}

	netVersion, err := version.ParseVersion(response.Version)
	if err != nil {
		return false, version.Version{}, errors.WithMessagef(err,
			"Failed to parse minimum client version %s from network",
			response.Version)
	}

	return true, netVersion, nil
}
