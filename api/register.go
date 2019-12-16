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
	"gitlab.com/elixxir/primitives/id"
	"sync"
	"time"
)

const SaltSize = 256

//RegisterWithPermissioning registers the user and returns the User ID.
// Returns an error if registration fails.
func (cl *Client) RegisterWithPermissioning(preCan bool, registrationCode, nick, email,
	password string, regInfo *RegisterInformation) (*id.User, error) {

	usr := regInfo.usr
	UID := regInfo.UID
	var err error

	//Set the status and make CMix keys array
	cl.opStatus(globals.REG_KEYGEN)
	nodeKeyMap := make(map[id.Node]user.NodeKeys)

	//Initialized response from Registration Server
	regValidationSignature := make([]byte, 0)

	//Handle registration
	if preCan {
		// Either perform a precanned registration for a precanned user
		cl.opStatus(globals.REG_PRECAN)
		globals.Log.INFO.Printf("Registering precanned user...")
		usr, UID, nodeKeyMap, err = cl.precannedRegister(registrationCode, nick, nodeKeyMap)
		if err != nil {
			globals.Log.ERROR.Printf("Unable to complete precanned registration: %+v", err)
			return id.ZeroID, err
		}
	} else {
		// Or register with the permissioning server and generate user information
		regValidationSignature, err = cl.registerWithPermissioning(registrationCode, nick, regInfo.rsaPublicKey)
		if err != nil {
			globals.Log.INFO.Printf(err.Error())
			return id.ZeroID, err
		}
	}

	//Set the registration secure state
	cl.opStatus(globals.REG_SECURE_STORE)

	usr.Email = email
	//Re-init in case pre-canned registration
	regInfo.usr = usr
	regInfo.UID = UID
	//Finalize session creation and store the session
	err = cl.finalizeSession(nodeKeyMap, regInfo, password, regValidationSignature)
	if err != nil {
		return id.ZeroID, err
	}

	return UID, nil
}

//RegisterWithUDB uses the account's email to register with the UDB for
// User discovery.  Must be called after Register and InitNetwork.
// It will fail if the user has already registered with UDB
func (cl *Client) RegisterWithUDB(timeout time.Duration) error {

	regState := cl.GetSession().GetRegState()

	if regState != user.PermissioningComplete {
		return errors.New("Cannot register with UDB when registration " +
			"state is not PermissioningComplete")
	}

	email := cl.session.GetCurrentUser().Email

	var err error

	if email != "" {
		globals.Log.INFO.Printf("Registering user as %s with UDB", email)

		valueType := "EMAIL"

		publicKeyBytes := cl.session.GetE2EDHPublicKey().Bytes()
		err = bots.Register(valueType, email, publicKeyBytes, cl.opStatus, timeout)
		if err == nil {
			globals.Log.INFO.Printf("Registered with UDB!")
		} else {
			globals.Log.WARN.Printf("Could not register with UDB: %s", err)
		}

	} else {
		globals.Log.INFO.Printf("Not registering with UDB because no " +
			"email found")
	}

	if err != nil {
		return errors.Wrap(err, "Could not register with UDB")
	}

	//set the registration state
	err = cl.session.SetRegState(user.UDBComplete)

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
	cl.opStatus(globals.REG_NODE)
	session := cl.GetSession()
	//Load Cmix keys & group
	cmixDHPrivKey := session.GetCMIXDHPrivateKey()
	cmixDHPubKey := session.GetCMIXDHPublicKey()
	cmixGrp := session.GetCmixGroup()

	//Load the rsa keys
	rsaPubKey := session.GetRSAPublicKey()
	rsaPrivKey := session.GetRSAPrivateKey()

	//Load the user ID
	UID := session.GetCurrentUser().User

	//Load the registration signature
	regSignature := session.GetRegistrationValidationSignature()

	var wg sync.WaitGroup
	errChan := make(chan error, len(cl.ndf.Gateways))

	//Get the registered node keys
	registeredNodes := session.GetNodes()

	salt := session.GetSalt()

	// This variable keeps track of whether there were new registrations
	// required, thus requiring the state file to be saved again
	newRegistrations := false

	for i := range cl.ndf.Gateways {
		localI := i
		nodeID := *id.NewNodeFromBytes(cl.ndf.Nodes[i].ID)
		//Register with node if the node has not been registered with already
		if _, ok := registeredNodes[nodeID]; !ok {
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
func (cl *Client) registerWithNode(index int, salt, registrationValidationSignature []byte, UID *id.User,
	publicKeyRSA *rsa.PublicKey, privateKeyRSA *rsa.PrivateKey,
	cmixPublicKeyDH, cmixPrivateKeyDH *cyclic.Int,
	cmixGrp *cyclic.Group, errorChan chan error) {

	gatewayID := id.NewNodeFromBytes(cl.ndf.Nodes[index].ID).NewGateway()

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
	}

	// Load server DH pubkey
	serverPubDH := cmixGrp.NewIntFromBytes(dhPub)

	// Confirm received nonce
	globals.Log.INFO.Println("Register: Confirming received nonce")
	err = cl.confirmNonce(UID.Bytes(), nonce, privateKeyRSA, gatewayID)
	if err != nil {
		errMsg := fmt.Sprintf("Register: Unable to confirm nonce: %v", err)
		errorChan <- errors.New(errMsg)
	}
	nodeID := cl.topology.GetNodeAtIndex(index)
	key := user.NodeKeys{
		TransmissionKey: registration.GenerateBaseKey(cmixGrp,
			serverPubDH, cmixPrivateKeyDH, transmissionHash),
		ReceptionKey: registration.GenerateBaseKey(cmixGrp, serverPubDH,
			cmixPrivateKeyDH, receptionHash),
	}
	cl.session.PushNodeKey(nodeID, key)
}

//finalizeSession serves as a helper function for RegisterWithPermissioning.
// It creates a session from all the generated values from registering and stores said session
func (cl *Client) finalizeSession(nodeKeyMap map[id.Node]user.NodeKeys,
	registrationInfo *RegisterInformation, password string, regSignature []byte) error {

	//Finalize session creation
	newSession := user.NewSession(cl.storage, registrationInfo.usr, nodeKeyMap, registrationInfo.rsaPublicKey,
		registrationInfo.rsaPrivateKey, registrationInfo.cmixPublicKey, registrationInfo.cmixPrivateKey,
		registrationInfo.e2ePublicKey, registrationInfo.e2ePrivateKey, registrationInfo.salt,
		registrationInfo.cmixGroup, registrationInfo.e2eGroup, password, regSignature)
	cl.opStatus(globals.REG_SAVE)

	//Set the registration state
	err := newSession.SetRegState(user.PermissioningComplete)
	if err != nil {
		return errors.Wrap(err, "Permissioning Registration "+
			"Failed")
	}

	//Store the user session
	errStore := newSession.StoreSession()
	if errStore != nil {
		return errors.Errorf("Permissioning Register: could not register due to failed session save: %s", errStore.Error())
	}

	//Set the client session as the newly created session
	cl.session = newSession

	return nil
}

//registerWithPermissioning serves as a helper function for RegisterWithPermissioning.
// It sends the registration message containing the regCode to permissioning
func (cl *Client) registerWithPermissioning(registrationCode, nickname string,
	publicKeyRSA *rsa.PublicKey) (regValidSig []byte, err error) {
	//Set the opStatus and log registration
	cl.opStatus(globals.REG_UID_GEN)
	globals.Log.INFO.Printf("Registering dynamic user...")

	// If Registration Server is specified, contact it
	// Only if registrationCode is set
	globals.Log.INFO.Println("Register: Contacting registration server")
	if cl.ndf.Registration.Address != "" && registrationCode != "" {
		cl.opStatus(globals.REG_PERM)
		regValidSig, err = cl.sendRegistrationMessage(registrationCode, publicKeyRSA)
		if err != nil {
			return nil, errors.Errorf("Register: Unable to send registration message: %+v", err)
		}
	}
	globals.Log.INFO.Println("Register: successfully passed Registration message")

	return regValidSig, nil
}
