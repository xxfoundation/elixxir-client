///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package api

/*
func (c *Client) Version() version.Version {
	v, err := version.ParseVersion(SEMVER)
	if err != nil {
		jww.FATAL.Panicf("Failed to parse the client version: %s", err)
	}
	return v
}

func (c *Client) checkVersion() error {
	clientVersion := c.Version()
	jww.INFO.Printf("Client Version: %s", clientVersion.String())

	has, netVersion, err := c.permissioning.GetNetworkVersion()
	if err != nil {
		return errors.WithMessage(err, "failed to get check "+
			"version compatibility")
	}
	if has {
		jww.INFO.Printf("Minimum Network Version: %v", netVersion)
		if !version.IsCompatible(netVersion, clientVersion) {
			return errors.Errorf("Client and Minimum Network Version are "+
				"incompatible\n"+
				"\tMinimum Network: %s\n"+
				"\tClient: %s", netVersion.String(), clientVersion.String())
		}
	} else {
		jww.INFO.Printf("Network requires no minimum version")
	}

	return nil
}
*/
