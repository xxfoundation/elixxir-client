package gateway

import (
	"errors"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
)

// Mock structure adhering to HostManager which returns the error
// path for all it's methods
type MangerErrorPath struct{}

// Constructor for MangerErrorPath
func newErrorManager() *MangerErrorPath {
	return &MangerErrorPath{}
}

func (mep *MangerErrorPath) GetHost(hostId *id.ID) (*connect.Host, bool) {
	return nil, false
}
func (mep *MangerErrorPath) AddHost(hid *id.ID, address string,
	cert []byte, params connect.HostParams) (host *connect.Host, err error) {
	return nil, errors.New("Failed to add host")
}

func (mep *MangerErrorPath) RemoveHost(hid *id.ID) {}

// Mock structure adhering to HostManager to be used for happy path
type ManagerHappyPath struct {
	hosts map[string]*connect.Host
}

// Constructor for ManagerHappyPath
func newHappyManager() *ManagerHappyPath {
	return &ManagerHappyPath{
		hosts: make(map[string]*connect.Host),
	}
}

func (mhp *ManagerHappyPath) GetHost(hostId *id.ID) (*connect.Host, bool) {
	h, ok := mhp.hosts[hostId.String()]
	return h, ok
}

func (mhp *ManagerHappyPath) AddHost(hid *id.ID, address string,
	cert []byte, params connect.HostParams) (host *connect.Host, err error) {
	host, err = connect.NewHost(hid, address, cert, params)
	if err != nil {
		return nil, err
	}

	mhp.hosts[hid.String()] = host

	return
}

func (mhp *ManagerHappyPath) RemoveHost(hid *id.ID) {
	delete(mhp.hosts, hid.String())
}

// Returns a mock
func getTestNdf(face interface{}) *ndf.NetworkDefinition {
	return &ndf.NetworkDefinition{
		Gateways: []ndf.Gateway{{
			ID:      id.NewIdFromUInt(0, id.Gateway, face)[:],
			Address: "0.0.0.1",
		}, {
			ID:      id.NewIdFromUInt(1, id.Gateway, face)[:],
			Address: "0.0.0.2",
		}, {
			ID:      id.NewIdFromUInt(2, id.Gateway, face)[:],
			Address: "0.0.0.3",
		}, {
			ID:      id.NewIdFromUInt(3, id.Gateway, face)[:],
			Address: "0.0.0.1",
		}, {
			ID:      id.NewIdFromUInt(4, id.Gateway, face)[:],
			Address: "0.0.0.2",
		}, {
			ID:      id.NewIdFromUInt(5, id.Gateway, face)[:],
			Address: "0.0.0.3",
		}, {
			ID:      id.NewIdFromUInt(6, id.Gateway, face)[:],
			Address: "0.0.0.1",
		}, {
			ID:      id.NewIdFromUInt(7, id.Gateway, face)[:],
			Address: "0.0.0.2",
		}, {
			ID:      id.NewIdFromUInt(8, id.Gateway, face)[:],
			Address: "0.0.0.3",
		}, {
			ID:      id.NewIdFromUInt(9, id.Gateway, face)[:],
			Address: "0.0.0.1",
		}, {
			ID:      id.NewIdFromUInt(10, id.Gateway, face)[:],
			Address: "0.0.0.2",
		}, {
			ID:      id.NewIdFromUInt(11, id.Gateway, face)[:],
			Address: "0.0.0.3",
		}},
		Nodes: []ndf.Node{{
			ID:      id.NewIdFromUInt(0, id.Node, face)[:],
			Address: "0.0.0.1",
		}, {
			ID:      id.NewIdFromUInt(1, id.Node, face)[:],
			Address: "0.0.0.2",
		}, {
			ID:      id.NewIdFromUInt(2, id.Node, face)[:],
			Address: "0.0.0.3",
		}, {
			ID:      id.NewIdFromUInt(3, id.Node, face)[:],
			Address: "0.0.0.1",
		}, {
			ID:      id.NewIdFromUInt(4, id.Node, face)[:],
			Address: "0.0.0.2",
		}, {
			ID:      id.NewIdFromUInt(5, id.Node, face)[:],
			Address: "0.0.0.3",
		}, {
			ID:      id.NewIdFromUInt(6, id.Node, face)[:],
			Address: "0.0.0.1",
		}, {
			ID:      id.NewIdFromUInt(7, id.Node, face)[:],
			Address: "0.0.0.2",
		}, {
			ID:      id.NewIdFromUInt(8, id.Node, face)[:],
			Address: "0.0.0.3",
		}, {
			ID:      id.NewIdFromUInt(9, id.Node, face)[:],
			Address: "0.0.0.1",
		}, {
			ID:      id.NewIdFromUInt(10, id.Node, face)[:],
			Address: "0.0.0.2",
		}, {
			ID:      id.NewIdFromUInt(11, id.Node, face)[:],
			Address: "0.0.0.3",
		}},
	}
}
