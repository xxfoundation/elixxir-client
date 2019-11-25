package api

import (
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/io"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
)

// Checks version and connects to gateways using TLS filepaths to create
// credential information for connection establishment
func (cl *Client) InitNetwork() error {
	//InitNetwork to permissioning
	isConnected, err := AddPermissioningHost(cl.commManager, cl.ndf)
	if err != nil {
		return err
	}
	if !isConnected {
		err = errors.New("Couldn't connect to permissioning")
		return err
	}
	//Get remote version and update
	ver, err := cl.commManager.GetRemoteVersion()
	if err != nil {
		return err
	}
	cl.registrationVersion = ver

	//Request a new ndf from permissioning
	def, err = io.GetUpdatedNDF(cl.ndf, cl.commManager.Comms)
	if err != nil {
		return err
	}
	if def != nil {
		cl.ndf = def
	}

	// Only check the version if we got a remote version
	// The remote version won't have been populated if we didn't connect to permissioning
	if cl.GetRegistrationVersion() != "" {
		ok, err := globals.CheckVersion(cl.GetRegistrationVersion())
		if err != nil {
			return err
		}
		if !ok {
			err = errors.New(fmt.Sprintf("Couldn't connect to gateways: Versions incompatible; Local version: %v; remote version: %v", globals.SEMVER,
				cl.GetRegistrationVersion()))
			return err
		}
	} else {
		globals.Log.WARN.Printf("Not checking version from " +
			"registration server, because it's not populated. Do you have " +
			"access to the registration server?")
	}

	//build the topology
	nodeIDs := make([]*id.Node, len(cl.ndf.Nodes))
	for i, node := range cl.ndf.Nodes {
		nodeIDs[i] = id.NewNodeFromBytes(node.ID)
	}

	cl.topology = connect.NewCircuit(nodeIDs)

	return AddGatewayHosts(cl.commManager, cl.ndf)
}

// Connects to gateways using tls filepaths to create credential information
// for connection establishment
func AddGatewayHosts(rm *io.ReceptionManager, definition *ndf.NetworkDefinition) error {
	if len(definition.Gateways) < 1 {
		return errors.New("could not connect due to invalid number of nodes")
	}

	// connect to all gateways
	var errs error = nil
	for i, gateway := range definition.Gateways {
		gwID := id.NewNodeFromBytes(definition.Nodes[i].ID).NewGateway()
		err := addHost(rm, gwID.String(), gateway.Address, gateway.TlsCertificate, false)
		errs = handleError(errs, err, gwID.String(), gateway.Address)
	}
	return errs
}

func addHost(rm *io.ReceptionManager, id, address, cert string, disableTimeout bool) error {
	var creds []byte
	if cert != "" {
		creds = []byte(cert)
	}
	_, err := rm.Comms.AddHost(id, address, creds, disableTimeout)
	if err != nil {
		return err
	}
	return nil
}

func handleError(base, err error, id, addr string) error {
	if err != nil {
		err = errors.Errorf("Failed to create host for gateway %s at %s: %+v",
			id, addr, err)
		if base != nil {
			base = errors.Wrap(base, err.Error())
		} else {
			base = err
		}
	}
	return base
}

// There's currently no need to keep connected to permissioning constantly,
// so we have functions to connect to and disconnect from it when a connection
// to permissioning is needed
func AddPermissioningHost(rm *io.ReceptionManager, definition *ndf.NetworkDefinition) (bool, error) {
	if definition.Registration.Address != "" {
		err := addHost(rm, PermissioningAddrID, definition.Registration.Address,
			definition.Registration.TlsCertificate, false)
		if err != nil {
			return false, errors.New(fmt.Sprintf(
				"Failed connecting to create host for permissioning: %+v", err))
		}
		return true, nil
	} else {
		globals.Log.DEBUG.Printf("failed to connect to %v silently", definition.Registration.Address)
		// Without an NDF, we can't connect to permissioning, but this isn't an
		// error per se, because we should be phasing out permissioning at some
		// point
		return false, nil
	}
}
