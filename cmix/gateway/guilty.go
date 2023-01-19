////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package gateway

import (
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/ndf"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer"
	"strings"
)

// List of errors that initiate a Host replacement
var errorsList = []string{
	context.DeadlineExceeded.Error(),
	"connection refused",
	"host disconnected",
	"transport is closing",
	balancer.ErrTransientFailure.Error(),
	"Last try to connect",
	ndf.NO_NDF,
	"Host is in cool down",
	grpc.ErrClientConnClosing.Error(),
	connect.TooManyProxyError,
	"Failed to fetch",
	"NetworkError when attempting to fetch resource.",
}

// IsGuilty returns true if the error means the host
// will get kicked out of the pool
func IsGuilty(err error) bool {
	for i := range errorsList {
		if strings.Contains(err.Error(), errorsList[i]) {
			return true
		}
	}
	return false
}
