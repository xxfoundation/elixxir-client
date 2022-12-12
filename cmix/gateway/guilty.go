package gateway

import (
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/ndf"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer"
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

var errorMap = make(map[string]struct{})

func init() {
	for _, str := range errorsList {
		errorMap[str] = struct{}{}
	}
}

// IsGuilty returns true if the error means the host
// will get kickled out of the pool
func IsGuilty(err error) bool {
	_, exists := errorMap[err.Error()]
	return exists
}
