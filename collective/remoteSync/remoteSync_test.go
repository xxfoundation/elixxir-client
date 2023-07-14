package remoteSync

import (
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/csprng"
	"testing"
)

type mockRemoteSyncComms struct {
	loggedIn bool
}

func (mrsc *mockRemoteSyncComms) Login(host *connect.Host, msg *pb.RsAuthenticationRequest) (*pb.RsAuthenticationResponse, error) {
	mrsc.loggedIn = true
	return &pb.RsAuthenticationResponse{Token: "token", ExpiresAt: 1024}, nil
}
func (mrsc *mockRemoteSyncComms) Read(host *connect.Host, msg *pb.RsReadRequest) (*pb.RsReadResponse, error) {
	if !mrsc.loggedIn {
		return nil, errNotLoggedIn
	}
	return &pb.RsReadResponse{}, nil
}
func (mrsc *mockRemoteSyncComms) Write(host *connect.Host, msg *pb.RsWriteRequest) (*messages.Ack, error) {
	if !mrsc.loggedIn {
		return nil, errNotLoggedIn
	}
	return &messages.Ack{}, nil
}
func (mrsc *mockRemoteSyncComms) GetLastModified(host *connect.Host, msg *pb.RsReadRequest) (*pb.RsTimestampResponse, error) {
	if !mrsc.loggedIn {
		return nil, errNotLoggedIn
	}
	return &pb.RsTimestampResponse{}, nil
}
func (mrsc *mockRemoteSyncComms) GetLastWrite(host *connect.Host, msg *pb.RsLastWriteRequest) (*pb.RsTimestampResponse, error) {
	if !mrsc.loggedIn {
		return nil, errNotLoggedIn
	}
	return &pb.RsTimestampResponse{}, nil
}
func (mrsc *mockRemoteSyncComms) ReadDir(host *connect.Host, msg *pb.RsReadRequest) (*pb.RsReadDirResponse, error) {
	if !mrsc.loggedIn {
		return nil, errNotLoggedIn
	}
	return &pb.RsReadDirResponse{}, nil
}

func TestManager_Read(t *testing.T) {
	m := manager{
		rng:     csprng.NewSystemRNG(),
		rsComms: &mockRemoteSyncComms{},
	}

	_, err := m.Read("/path/to/resource")
	if err != nil {
		t.Fatal(err)
	}
	if !m.rsComms.(*mockRemoteSyncComms).loggedIn {
		t.Fatal("Did not log in when error received")
	}
}

func TestManager_Write(t *testing.T) {
	m := manager{
		rng:     csprng.NewSystemRNG(),
		rsComms: &mockRemoteSyncComms{},
	}

	err := m.Write("/path/to/resource", []byte("Data"))
	if err != nil {
		t.Fatal(err)
	}
	if !m.rsComms.(*mockRemoteSyncComms).loggedIn {
		t.Fatal("Did not log in when error received")
	}
}

func TestManager_ReadDir(t *testing.T) {
	m := manager{
		rng:     csprng.NewSystemRNG(),
		rsComms: &mockRemoteSyncComms{},
	}

	_, err := m.ReadDir("/path/to/resource")
	if err != nil {
		t.Fatal(err)
	}
	if !m.rsComms.(*mockRemoteSyncComms).loggedIn {
		t.Fatal("Did not log in when error received")
	}
}

func TestManager_GetLastModified(t *testing.T) {
	m := manager{
		rng:     csprng.NewSystemRNG(),
		rsComms: &mockRemoteSyncComms{},
	}

	_, err := m.GetLastModified("/path/to/resource")
	if err != nil {
		t.Fatal(err)
	}
	if !m.rsComms.(*mockRemoteSyncComms).loggedIn {
		t.Fatal("Did not log in when error received")
	}
}

func TestManager_GetLastWrite(t *testing.T) {
	m := manager{
		rng:     csprng.NewSystemRNG(),
		rsComms: &mockRemoteSyncComms{},
	}

	_, err := m.GetLastWrite()
	if err != nil {
		t.Fatal(err)
	}
	if !m.rsComms.(*mockRemoteSyncComms).loggedIn {
		t.Fatal("Did not log in when error received")
	}
}
