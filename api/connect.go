package api

import (
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/io"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
)

// Checks version and connects to gateways using TLS filepaths to create
// credential information for connection establishment
func (cl *Client) InitNetwork() error {
	//InitNetwork to permissioning
	if cl.ndf.Registration.Address != "" {
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

		//Request a new ndf from
		err = requestNdf(cl)
		if err != nil {
			return err

		}
	} else {
		globals.Log.WARN.Println("Registration not defined, not contacted")
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

	cl.topology = circuit.New(nodeIDs)

	return AddGatewayHosts(cl.commManager, cl.ndf)
}

// Connects to gateways using tls filepaths to create credential information
// for connection establishment
func AddGatewayHosts(rm *io.ReceptionManager, definition *ndf.NetworkDefinition) error { // tear out
	if len(definition.Gateways) < 1 {
		return errors.New("could not connect due to invalid number of nodes")
	}

	// connect to all gateways
	var errs error = nil
	for i, gateway := range definition.Gateways {

		var gwCreds []byte

		if gateway.TlsCertificate != "" {
			gwCreds = []byte(gateway.TlsCertificate)
		}
		gwID := id.NewNodeFromBytes(definition.Nodes[i].ID).NewGateway()
		gwAddr := gateway.Address

		_, err := rm.Comms.AddHost(gwID.String(), gwAddr, gwCreds, false)
		if err != nil {
			err = errors.Errorf("Failed to create host for gateway %s at %s: %+v",
				gwID.String(), gwAddr, err)
			if errs != nil {
				errs = errors.Wrap(errs, err.Error())
			} else {
				errs = err
			}
		}
	}
	return errs
}

// There's currently no need to keep connected to permissioning constantly,
// so we have functions to connect to and disconnect from it when a connection
// to permissioning is needed
func AddPermissioningHost(rm *io.ReceptionManager, definition *ndf.NetworkDefinition) (bool, error) { // this disappears, make host in simple call
	if definition.Registration.Address != "" {
		var regCert []byte
		if definition.Registration.TlsCertificate != "" {
			regCert = []byte(definition.Registration.TlsCertificate)
		}

		_, err := rm.Comms.AddHost(PermissioningAddrID, definition.Registration.Address, regCert, false)
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
