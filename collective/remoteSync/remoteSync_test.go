package remoteSync

import (
	"testing"

	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/csprng"
)

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

////////////////////////////////////////////////////////////////////////////////
// Mock Remote Sync Comms                                                     //
////////////////////////////////////////////////////////////////////////////////

type mockRemoteSyncComms struct {
	loggedIn bool
}

func (mRSC *mockRemoteSyncComms) Login(
	*connect.Host, *pb.RsAuthenticationRequest) (*pb.RsAuthenticationResponse, error) {
	mRSC.loggedIn = true
	return &pb.RsAuthenticationResponse{Token: []byte("token"), ExpiresAt: 1024}, nil
}

func (mRSC *mockRemoteSyncComms) Read(
	*connect.Host, *pb.RsReadRequest) (*pb.RsReadResponse, error) {
	if !mRSC.loggedIn {
		return nil, errNotLoggedIn
	}
	return &pb.RsReadResponse{}, nil
}

func (mRSC *mockRemoteSyncComms) Write(
	*connect.Host, *pb.RsWriteRequest) (*messages.Ack, error) {
	if !mRSC.loggedIn {
		return nil, errNotLoggedIn
	}
	return &messages.Ack{}, nil
}

func (mRSC *mockRemoteSyncComms) GetLastModified(
	*connect.Host, *pb.RsReadRequest) (*pb.RsTimestampResponse, error) {
	if !mRSC.loggedIn {
		return nil, errNotLoggedIn
	}
	return &pb.RsTimestampResponse{}, nil
}

func (mRSC *mockRemoteSyncComms) GetLastWrite(
	*connect.Host, *pb.RsLastWriteRequest) (*pb.RsTimestampResponse, error) {
	if !mRSC.loggedIn {
		return nil, errNotLoggedIn
	}
	return &pb.RsTimestampResponse{}, nil
}

func (mRSC *mockRemoteSyncComms) ReadDir(
	*connect.Host, *pb.RsReadRequest) (*pb.RsReadDirResponse, error) {
	if !mRSC.loggedIn {
		return nil, errNotLoggedIn
	}
	return &pb.RsReadDirResponse{}, nil
}