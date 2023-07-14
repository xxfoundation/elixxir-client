package remoteSync

import (
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
)

// Comms interface for remote sync
type Comms interface {
	Login(host *connect.Host, msg *pb.RsAuthenticationRequest) (*pb.RsAuthenticationResponse, error)
	Read(host *connect.Host, msg *pb.RsReadRequest) (*pb.RsReadResponse, error)
	Write(host *connect.Host, msg *pb.RsWriteRequest) (*messages.Ack, error)
	GetLastModified(host *connect.Host, msg *pb.RsReadRequest) (*pb.RsTimestampResponse, error)
	GetLastWrite(host *connect.Host, msg *pb.RsLastWriteRequest) (*pb.RsTimestampResponse, error)
	ReadDir(host *connect.Host, msg *pb.RsReadRequest) (*pb.RsReadDirResponse, error)
}
