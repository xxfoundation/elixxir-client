package permissioning

import (
	"github.com/pkg/errors"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/version"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"testing"
)

type MockVersionSender struct {
	// param passed to SendRegistrationMessage
	host *connect.Host
	// original host returned from GetHost
	getHost        *connect.Host
	succeedGetHost bool
	returnVersion  string
	returnErr      error
}

func (s *MockVersionSender) SendGetCurrentClientVersionMessage(
	_ *connect.Host) (*pb.ClientVersion, error) {
	return &pb.ClientVersion{Version: s.returnVersion}, s.returnErr
}

// Test happy path: get a version
func TestPermissioning_GetNetworkVersion(t *testing.T) {
	var sender MockVersionSender
	var err error
	sender.succeedGetHost = true
	sender.getHost, err = connect.NewHost(&id.Permissioning, "address", nil,
		connect.GetDefaultHostParams())
	if err != nil {
		t.Fatal(err)
	}
	sender.returnErr = nil
	sender.returnVersion = "0.1.0"
	ok, version, err := getRemoteVersion(sender.getHost, &sender)
	if err != nil {
		t.Error(err)
	}
	if ok != true {
		t.Error("ok should be true after getting a response from permissioning")
	}
	if version.String() != sender.returnVersion {
		t.Error("getRemoteVersion should have returned the version we asked for")
	}
}

// Test errors: version unparseable or missing, or error returned
func TestPermissioning_GetNetworkVersion_Errors(t *testing.T) {
	var sender MockVersionSender
	var err error
	sender.succeedGetHost = true
	sender.getHost, err = connect.NewHost(&id.Permissioning, "address", nil,
		connect.GetDefaultHostParams())
	if err != nil {
		t.Fatal(err)
	}

	// Case 1: RPC returns error
	sender.returnErr = errors.New("an error")
	sender.returnVersion = "0.1.0"
	ok, v, err := getRemoteVersion(sender.getHost, &sender)
	if ok {
		t.Error("shouldn't have gotten OK in error case")
	}
	if !reflect.DeepEqual(v, version.Version{}) {
		t.Error("returned version should be empty")
	}
	if err == nil {
		t.Error("error should exist")
	}

	// Case 2: RPC returns an empty string
	sender.returnErr = nil
	sender.returnVersion = ""
	ok, v, err = getRemoteVersion(sender.getHost, &sender)
	if ok {
		t.Error("shouldn't have gotten OK in error case")
	}
	if !reflect.DeepEqual(v, version.Version{}) {
		t.Error("returned version should be empty")
	}
	if err != nil {
		t.Error("returning an empty string and no error isn't an error case, so no error should be returned")
	}

	// Case 3: RPC returns an unparseable string
	sender.returnErr = nil
	sender.returnVersion = "flooble doodle"
	ok, v, err = getRemoteVersion(sender.getHost, &sender)
	if ok {
		t.Error("shouldn't have gotten OK in error case")
	}
	if !reflect.DeepEqual(v, version.Version{}) {
		t.Error("returned version should be empty")
	}
	if err == nil {
		t.Error("Should return an error indicating the version string was unparseable")
	}
}
