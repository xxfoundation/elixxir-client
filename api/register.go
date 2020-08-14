////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import (
	"crypto/sha256"
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/bots"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/registration"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/crypto/tls"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"os"
	"sync"
	"time"
)

const SaltSize = 32

//RegisterWithPermissioning registers the user and returns the User ID.
// Returns an error if registration fails.
func (cl *Client) RegisterWithPermissioning(preCan bool, registrationCode string) (*id.ID, error) {
	//Check the regState is in proper state for registration
	regState, err := cl.sessionV2.GetRegState()
	if err != nil {
		return nil, err
	}

	if regState != user.KeyGenComplete {
		return nil, errors.Errorf("Attempting to register before key generation!")
	}
	userData, err := cl.sessionV2.GetUserData()
	if err != nil {
		return nil, err
	}
	usr := userData.ThisUser
	UID := usr.User

	//Initialized response from Registration Server
	regValidationSignature := make([]byte, 0)

	//Handle registration
	if preCan {
		// Either perform a precanned registration for a precanned user
		cl.opStatus(globals.REG_PRECAN)
		globals.Log.INFO.Printf("Registering precanned user...")
		var nodeKeyMap map[id.ID]user.NodeKeys
		usr, UID, nodeKeyMap, err = cl.precannedRegister(registrationCode)
		if err != nil {
			globals.Log.ERROR.Printf("Unable to complete precanned registration: %+v", err)
			return &id.ZeroUser, err
		}

		//overwrite the user object
		usr.Precan = true
		userData.ThisUser = usr
		cl.sessionV2.CommitUserData(userData)

		//store the node keys
		for n, k := range nodeKeyMap {
			cl.sessionV2.PushNodeKey(&n, k)
		}

		//update the state
		err = cl.sessionV2.SetRegState(user.PermissioningComplete)
		if err != nil {
			return &id.ZeroUser, err
		}

	} else {
		// Or register with the permissioning server and generate user information
		regValidationSignature, err = cl.registerWithPermissioning(
			registrationCode,
			userData.RSAPublicKey)
		if err != nil {
			globals.Log.INFO.Printf(err.Error())
			return &id.ZeroUser, err
		}
		//update the session with the registration
		err = cl.sessionV2.SetRegState(user.PermissioningComplete)
		if err != nil {
			return nil, err
		}

		err = cl.sessionV2.SetRegValidationSig(regValidationSignature)
		if err != nil {
			return nil, err
		}

	}

	//Set the registration secure state
	cl.opStatus(globals.REG_SECURE_STORE)

	//store the updated session
	err = cl.session.StoreSession()

	if err != nil {
		return nil, err
	}

	return UID, nil
}

//RegisterWithUDB uses the account's email to register with the UDB for
// User discovery.  Must be called after Register and InitNetwork.
// It will fail if the user has already registered with UDB
func (cl *Client) RegisterWithUDB(username string, timeout time.Duration) error {
	regState, err := cl.sessionV2.GetRegState()
	if err != nil {
		return err
	}
	userData, err := cl.sessionV2.GetUserData()
	if err != nil {
		return err
	}

	if regState != user.PermissioningComplete {
		return errors.New("Cannot register with UDB when registration " +
			"state is not PermissioningComplete")
	}

	if username != "" {
		userData.ThisUser.Username = username
		cl.sessionV2.CommitUserData(userData)

		globals.Log.INFO.Printf("Registering user as %s with UDB", username)

		valueType := "EMAIL"

		publicKeyBytes := userData.E2EDHPublicKey.Bytes()
		err = bots.Register(valueType, username, publicKeyBytes, cl.opStatus, timeout)
		if err != nil {
			return errors.Errorf("Could not register with UDB: %s", err)
		}
		globals.Log.INFO.Printf("Registered with UDB!")
	} else {
		globals.Log.INFO.Printf("Not registering with UDB because no " +
			"email found")
	}

	//set the registration state
	err = cl.sessionV2.SetRegState(user.UDBComplete)
	if err != nil {
		return errors.Wrap(err, "UDB Registration Failed")
	}

	cl.opStatus(globals.REG_SECURE_STORE)

	errStore := cl.session.StoreSession()

	// FIXME If we have an error here, the session that gets created
	// doesn't get immolated. Immolation should happen in a deferred
	//  call instead.
	if errStore != nil {
		err = errors.New(fmt.Sprintf(
			"UDB Register: could not register due to failed session save"+
				": %s", errStore.Error()))
		return err
	}

	return nil
}

//RegisterWithNodes registers the client with all the nodes within the ndf
func (cl *Client) RegisterWithNodes() error {

	userData, err := cl.sessionV2.GetUserData()
	if err != nil {
		return err
	}

	cl.opStatus(globals.REG_NODE)
	session := cl.GetSession()
	//Load Cmix keys & group
	cmixDHPrivKey := userData.CMIXDHPrivateKey
	cmixDHPubKey := userData.CMIXDHPublicKey
	cmixGrp := userData.CmixGrp

	//Load the rsa keys
	rsaPubKey := userData.RSAPublicKey
	rsaPrivKey := userData.RSAPrivateKey

	//Load the user ID
	UID := userData.ThisUser.User
	usr := userData.ThisUser
	//Load the registration signature
	regSignature, err := cl.sessionV2.GetRegValidationSig()
	if err != nil && !os.IsNotExist(err) {
		return errors.Errorf("Failed to get registration signature: %v",
			err)
	}

	// Storage of the registration signature was broken in previous releases.
	// get the signature again from permissioning if it is absent
	var regPubKey *rsa.PublicKey
	if cl.ndf.Registration.TlsCertificate != "" {
		var err error
		regPubKey, err = extractPublicKeyFromCert(cl.ndf)
		if err != nil {
			return err
		}
	}

	// Storage of the registration signature was broken in previous releases.
	// get the signature again from permissioning if it is absent
	if !usr.Precan && !rsa.IsValidSignature(regPubKey, regSignature) {
		// Or register with the permissioning server and generate user information
		regSignature, err := cl.registerWithPermissioning("", userData.RSAPublicKey)
		if err != nil {
			globals.Log.INFO.Printf(err.Error())
			return err
		}
		//update the session with the registration
		//HACK HACK HACK
		sesObj := cl.session.(*user.SessionObj)
		err = cl.sessionV2.SetRegValidationSig(regSignature)
		if err != nil {
			return err
		}

		err = sesObj.StoreSession()

		if err != nil {
			return err
		}
	}

	//make the wait group to wait for all node registrations to complete
	var wg sync.WaitGroup
	errChan := make(chan error, len(cl.ndf.Gateways))

	registeredNodes, err := cl.sessionV2.GetNodeKeys()
	if err != nil {
		return err
	}

	salt := userData.Salt

	// This variable keeps track of whether there were new registrations
	// required, thus requiring the state file to be saved again
	newRegistrations := false

	for i := range cl.ndf.Gateways {
		localI := i
		nodeID, err := id.Unmarshal(cl.ndf.Nodes[i].ID)
		if err != nil {
			return nil
		}
		//Register with node if the node has not been registered with already
		if _, ok := registeredNodes[nodeID.String()]; !ok {
			wg.Add(1)
			newRegistrations = true
			go func() {
				cl.registerWithNode(localI, salt, regSignature, UID, rsaPubKey, rsaPrivKey,
					cmixDHPubKey, cmixDHPrivKey, cmixGrp, errChan)
				wg.Done()
			}()
		}
	}
	wg.Wait()
	//See if the registration returned errors at all
	var errs error
	for len(errChan) > 0 {
		err := <-errChan
		if errs != nil {
			errs = errors.Wrap(errs, err.Error())
		} else {
			errs = err
		}

	}
	//If an error every occurred, return with error
	if errs != nil {
		cl.opStatus(globals.REG_FAIL)
		return errs
	}

	// Store the user session if there were changes during node registration
	if newRegistrations {
		cl.opStatus(globals.REG_SECURE_STORE)
		errStore := session.StoreSession()
		if errStore != nil {
			err := errors.New(fmt.Sprintf(
				"Register: could not register due to failed session save"+
					": %s", errStore.Error()))
			return err
		}
	}

	return nil
}

//registerWithNode serves as a helper for RegisterWithNodes
// It registers a user with a specific in the client's ndf.
func (cl *Client) registerWithNode(index int, salt, registrationValidationSignature []byte, UID *id.ID,
	publicKeyRSA *rsa.PublicKey, privateKeyRSA *rsa.PrivateKey,
	cmixPublicKeyDH, cmixPrivateKeyDH *cyclic.Int,
	cmixGrp *cyclic.Group, errorChan chan error) {

	gatewayID, err := id.Unmarshal(cl.ndf.Gateways[index].ID)
	if err != nil {
		errorChan <- err
		return
	}

	// Initialise blake2b hash for transmission keys and sha256 for reception
	// keys
	transmissionHash, _ := hash.NewCMixHash()
	receptionHash := sha256.New()

	// Request nonce message from gateway
	globals.Log.INFO.Printf("Register: Requesting nonce from gateway %v/%v",
		index+1, len(cl.ndf.Gateways))
	nonce, dhPub, err := cl.requestNonce(salt, registrationValidationSignature, cmixPublicKeyDH,
		publicKeyRSA, privateKeyRSA, gatewayID)

	if err != nil {
		errMsg := fmt.Sprintf("Register: Failed requesting nonce from gateway: %+v", err)
		errorChan <- errors.New(errMsg)
		return
	}

	// Load server DH pubkey
	serverPubDH := cmixGrp.NewIntFromBytes(dhPub)

	// Confirm received nonce
	globals.Log.INFO.Println("Register: Confirming received nonce")
	err = cl.confirmNonce(UID.Bytes(), nonce, privateKeyRSA, gatewayID)
	if err != nil {
		errMsg := fmt.Sprintf("Register: Unable to confirm nonce: %v", err)
		errorChan <- errors.New(errMsg)
		return
	}
	nodeID := cl.topology.GetNodeAtIndex(index)
	key := user.NodeKeys{
		TransmissionKey: registration.GenerateBaseKey(cmixGrp,
			serverPubDH, cmixPrivateKeyDH, transmissionHash),
		ReceptionKey: registration.GenerateBaseKey(cmixGrp, serverPubDH,
			cmixPrivateKeyDH, receptionHash),
	}
	cl.sessionV2.PushNodeKey(nodeID, key)
}

//registerWithPermissioning serves as a helper function for RegisterWithPermissioning.
// It sends the registration message containing the regCode to permissioning
func (cl *Client) registerWithPermissioning(registrationCode string,
	publicKeyRSA *rsa.PublicKey) (regValidSig []byte, err error) {
	//Set the opStatus and log registration
	globals.Log.INFO.Printf("Registering dynamic user...")

	// If Registration Server is not specified return an error
	if cl.ndf.Registration.Address == "" {
		return nil, errors.New("No registration attempted, " +
			"registration server not known")
	}

	// attempt to register with registration
	globals.Log.INFO.Println("Register: Registering with registration server")
	cl.opStatus(globals.REG_PERM)
	regValidSig, err = cl.sendRegistrationMessage(registrationCode, publicKeyRSA)
	if err != nil {
		return nil, errors.Errorf("Register: Unable to send registration message: %+v", err)
	}

	globals.Log.INFO.Println("Register: successfully registered")

	return regValidSig, nil
}

// extractPublicKeyFromCert is a utility function which pulls out the public key from a certificate
func extractPublicKeyFromCert(definition *ndf.NetworkDefinition) (*rsa.PublicKey, error) {
	// Load certificate object
	cert, err := tls.LoadCertificate(definition.Registration.TlsCertificate)
	if err != nil {
		return nil, errors.Errorf("Failed to parse certificate: %+v", err)
	}
	//Extract public key from cert
	regPubKey, err := tls.ExtractPublicKey(cert)
	if err != nil {
		return nil, errors.Errorf("Failed to pull key from cert: %+v", err)
	}

	return regPubKey, nil

}
