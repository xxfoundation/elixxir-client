////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package gateway

import (
	"github.com/pkg/errors"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"time"
)

// mockManager is a mock structure adhering to HostManager to be used for happy
// path.
type mockManager struct {
	hosts map[string]*connect.Host
}

// newMockManager is the constructor for mockManager.
func newMockManager() *mockManager {
	return &mockManager{
		hosts: make(map[string]*connect.Host),
	}
}

func (mhp *mockManager) GetHost(hostId *id.ID) (*connect.Host, bool) {
	h, ok := mhp.hosts[hostId.String()]
	return h, ok
}

func (mhp *mockManager) AddHost(hid *id.ID, address string,
	cert []byte, params connect.HostParams) (host *connect.Host, err error) {
	host, err = connect.NewHost(hid, address, cert, params)
	if err != nil {
		return nil, err
	}

	mhp.hosts[hid.String()] = host

	return
}

func (mhp *mockManager) RemoveHost(hid *id.ID) {
	delete(mhp.hosts, hid.String())
}

// Returns a mock
func getTestNdf(face interface{}) *ndf.NetworkDefinition {
	return &ndf.NetworkDefinition{
		Gateways: []ndf.Gateway{{
			ID:      id.NewIdFromUInt(0, id.Gateway, face)[:],
			Address: "0.0.0.1:11420",
		}, {
			ID:      id.NewIdFromUInt(1, id.Gateway, face)[:],
			Address: "0.0.0.2:11420",
		}, {
			ID:      id.NewIdFromUInt(2, id.Gateway, face)[:],
			Address: "0.0.0.3:11420",
		}, {
			ID:      id.NewIdFromUInt(3, id.Gateway, face)[:],
			Address: "0.0.0.1:11420",
		}, {
			ID:      id.NewIdFromUInt(4, id.Gateway, face)[:],
			Address: "0.0.0.2:11420",
		}, {
			ID:      id.NewIdFromUInt(5, id.Gateway, face)[:],
			Address: "0.0.0.3:11420",
		}, {
			ID:      id.NewIdFromUInt(6, id.Gateway, face)[:],
			Address: "0.0.0.1:11420",
		}, {
			ID:      id.NewIdFromUInt(7, id.Gateway, face)[:],
			Address: "0.0.0.2:11420",
		}, {
			ID:      id.NewIdFromUInt(8, id.Gateway, face)[:],
			Address: "0.0.0.3:11420",
		}, {
			ID:      id.NewIdFromUInt(9, id.Gateway, face)[:],
			Address: "0.0.0.1:11420",
		}, {
			ID:      id.NewIdFromUInt(10, id.Gateway, face)[:],
			Address: "0.0.0.2:11420",
		}, {
			ID:      id.NewIdFromUInt(11, id.Gateway, face)[:],
			Address: "0.0.0.3:11420",
		}},
		Nodes: []ndf.Node{{
			ID:      id.NewIdFromUInt(0, id.Node, face)[:],
			Address: "0.0.0.1:11420",
			Status:  ndf.Active,
		}, {
			ID:      id.NewIdFromUInt(1, id.Node, face)[:],
			Address: "0.0.0.2:11420",
			Status:  ndf.Active,
		}, {
			ID:      id.NewIdFromUInt(2, id.Node, face)[:],
			Address: "0.0.0.3:11420",
			Status:  ndf.Active,
		}, {
			ID:      id.NewIdFromUInt(3, id.Node, face)[:],
			Address: "0.0.0.1:11420",
			Status:  ndf.Active,
		}, {
			ID:      id.NewIdFromUInt(4, id.Node, face)[:],
			Address: "0.0.0.2:11420",
			Status:  ndf.Active,
		}, {
			ID:      id.NewIdFromUInt(5, id.Node, face)[:],
			Address: "0.0.0.3:11420",
			Status:  ndf.Active,
		}, {
			ID:      id.NewIdFromUInt(6, id.Node, face)[:],
			Address: "0.0.0.1:11420",
			Status:  ndf.Active,
		}, {
			ID:      id.NewIdFromUInt(7, id.Node, face)[:],
			Address: "0.0.0.2:11420",
			Status:  ndf.Active,
		}, {
			ID:      id.NewIdFromUInt(8, id.Node, face)[:],
			Address: "0.0.0.3:11420",
			Status:  ndf.Active,
		}, {
			ID:      id.NewIdFromUInt(9, id.Node, face)[:],
			Address: "0.0.0.1:11420",
			Status:  ndf.Active,
		}, {
			ID:      id.NewIdFromUInt(10, id.Node, face)[:],
			Address: "0.0.0.2:11420",
			Status:  ndf.Active,
		}, {
			ID:      id.NewIdFromUInt(11, id.Node, face)[:],
			Address: "0.0.0.3:11420",
			Status:  ndf.Active,
		}},
	}
}

const happyPathReturn = "happyPathReturn"

func SendToPreferredHappyPath(*connect.Host, *id.ID, time.Duration) (
	interface{}, error) {
	return happyPathReturn, nil
}

func SendToPreferredKnownError(*connect.Host, *id.ID, time.Duration) (
	interface{}, error) {
	return nil, errors.Errorf(errorsList[0])
}

func SendToPreferredUnknownError(*connect.Host, *id.ID, time.Duration) (
	interface{}, error) {
	return nil, errors.Errorf("Unexpected error: Oopsie")
}

func SendToAnyHappyPath(*connect.Host) (interface{}, error) {
	return happyPathReturn, nil
}

func SendToAnyKnownError(*connect.Host) (interface{}, error) {
	return nil, errors.Errorf(errorsList[0])
}

func SendToAnyUnknownError(*connect.Host) (interface{}, error) {
	return nil, errors.Errorf("Unexpected error: Oopsie")
}

func SendToSpecificHappyPath(_ *connect.Host, _ *id.ID) (
	interface{}, bool, error) {
	return happyPathReturn, false, nil
}

func SendToSpecificAbort(_ *connect.Host, _ *id.ID) (
	interface{}, bool, error) {
	return nil, true, errors.Errorf(errorsList[0])
}
