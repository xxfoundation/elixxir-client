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

var ErrNoPermissioning = errors.New("No Permissioning In NDF")

// Checks version and connects to gateways using TLS filepaths to create
// credential information for connection establishment
func (cl *Client) InitNetwork() error {
	//InitNetwork to permissioning
	err := AddPermissioningHost(cl.receptionManager, cl.ndf)

	if err != nil {
		if err != ErrNoPermissioning {
			// Permissioning has an error so stop running
			return err
		}
		globals.Log.WARN.Print("Skipping connection to permissioning, most likely no permissioning information in NDF")
	}

	runPermissioning := err != ErrNoPermissioning

	if runPermissioning {
		err = cl.setupPermissioning()

		if err != nil {
			return err
		}
	}

	//build the topology
	nodeIDs := make([]*id.Node, len(cl.ndf.Nodes))
	for i, node := range cl.ndf.Nodes {
		nodeIDs[i] = id.NewNodeFromBytes(node.ID)
	}

	cl.topology = connect.NewCircuit(nodeIDs)

	return AddGatewayHosts(cl.receptionManager, cl.ndf)
}

// DisableTls disables tls for communications
func (cl *Client) DisableTls() {
	globals.Log.INFO.Println("Running client without tls")
	cl.receptionManager.Tls = false
}

func (cl *Client) setupPermissioning() error {
	// Permissioning was found in ndf run corresponding code

	//Get remote version and update
	ver, err := cl.receptionManager.GetRemoteVersion()
	if err != nil {
		return err
	}
	cl.registrationVersion = ver

	//Request a new ndf from permissioning
	def, err = io.PollNdf(cl.ndf, cl.receptionManager.Comms)
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

	return nil

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
		err := addHost(rm, gwID.String(), gateway.Address, gateway.TlsCertificate, false, true)
		if err != nil {
			err = errors.Errorf("Failed to create host for gateway %s at %s: %+v",
				gwID.String(), gateway.Address, err)
			if errs != nil {
				errs = err
			} else {
				errs = errors.Wrap(errs, err.Error())
			}
		}
	}
	return errs
}

func addHost(rm *io.ReceptionManager, id, address, cert string, disableTimeout, enableAuth bool) error {
	var creds []byte
	if cert != "" && rm.Tls {
		creds = []byte(cert)
	}
	_, err := rm.Comms.AddHost(id, address, creds, disableTimeout, enableAuth)
	if err != nil {
		return err
	}
	return nil
}

// There's currently no need to keep connected to permissioning constantly,
// so we have functions to connect to and disconnect from it when a connection
// to permissioning is needed
func AddPermissioningHost(rm *io.ReceptionManager, definition *ndf.NetworkDefinition) error {
	if definition.Registration.Address != "" {
		err := addHost(rm, PermissioningAddrID, definition.Registration.Address,
			definition.Registration.TlsCertificate, false, true)
		if err != nil {
			return errors.New(fmt.Sprintf(
				"Failed connecting to create host for permissioning: %+v", err))
		}
		return nil
	} else {
		globals.Log.DEBUG.Printf("failed to connect to %v silently", definition.Registration.Address)
		// Without an NDF, we can't connect to permissioning, but this isn't an
		// error per se, because we should be phasing out permissioning at some
		// point
		return ErrNoPermissioning
	}
}
