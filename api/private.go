////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package api

import (
	"crypto"
	"crypto/rand"
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/io"
	"gitlab.com/elixxir/client/keyStore"
	"gitlab.com/elixxir/client/storage"
	user2 "gitlab.com/elixxir/client/storage/user"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/client/userRegistry"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/xx"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
)

const PermissioningAddrID = "Permissioning"

// precannedRegister is a helper function for Register
// It handles the precanned registration case
func (cl *Client) precannedRegister(registrationCode string) (*user2.User, *id.ID, map[id.ID]user.NodeKeys, error) {
	var successLook bool
	var UID *id.ID
	var u *user2.User
	var err error

	nk := make(map[id.ID]user.NodeKeys)

	UID, successLook = userRegistry.Users.LookupUser(registrationCode)

	globals.Log.DEBUG.Printf("UID: %+v, success: %+v", UID, successLook)

	if !successLook {
		return nil, nil, nil, errors.New("precannedRegister: could not register due to invalid HUID")
	}

	var successGet bool
	u, successGet = userRegistry.Users.GetUser(UID)

	if !successGet {
		err = errors.New("precannedRegister: could not register due to ID lookup failure")
		return nil, nil, nil, err
	}

	nodekeys, successKeys := userRegistry.Users.LookupKeys(u.User)

	if !successKeys {
		err = errors.New("precannedRegister: could not register due to missing user keys")
		return nil, nil, nil, err
	}

	for i := 0; i < len(cl.ndf.Gateways); i++ {
		nk[*cl.topology.GetNodeAtIndex(i)] = *nodekeys
	}
	return u, UID, nk, nil
}

// sendRegistrationMessage is a helper for the Register function
// It sends a registration message and returns the registration signature
func (cl *Client) sendRegistrationMessage(registrationCode string,
	publicKeyRSA *rsa.PublicKey) ([]byte, error) {
	err := addPermissioningHost(cl.receptionManager, cl.ndf)

	if err != nil {
		if err == ErrNoPermissioning {
			return nil, errors.New("Didn't connect to permissioning to send registration message. Check the NDF")
		}
		return nil, errors.Wrap(err, "Couldn't connect to permissioning to send registration message")
	}

	regValidationSignature := make([]byte, 0)
	// Send registration code and public key to RegistrationServer
	host, ok := cl.receptionManager.Comms.GetHost(&id.Permissioning)
	if !ok {
		return nil, errors.New("Failed to find permissioning host")
	}

	response, err := cl.receptionManager.Comms.
		SendRegistrationMessage(host,
			&pb.UserRegistration{
				RegistrationCode: registrationCode,
				ClientRSAPubKey:  string(rsa.CreatePublicKeyPem(publicKeyRSA)),
			})
	if err != nil {
		err = errors.Wrap(err, "sendRegistrationMessage: Unable to contact Registration Server!")
		return nil, err
	}
	if response.Error != "" {
		return nil, errors.Wrapf(err, "sendRegistrationMessage: error handling message: %s", response.Error)
	}
	regValidationSignature = response.ClientSignedByServer.Signature
	// Disconnect from regServer here since it will not be needed
	return regValidationSignature, nil
}

// requestNonce is a helper for the Register function
// It sends a request nonce message containing the client's keys for signing
// Returns nonce if successful
func (cl *Client) requestNonce(salt, regHash []byte,
	publicKeyDH *cyclic.Int, publicKeyRSA *rsa.PublicKey,
	privateKeyRSA *rsa.PrivateKey, gwID *id.ID) ([]byte, []byte, error) {
	dhPub := publicKeyDH.Bytes()
	sha := crypto.SHA256
	opts := rsa.NewDefaultOptions()
	opts.Hash = sha
	h := sha.New()
	h.Write(dhPub)
	data := h.Sum(nil)

	// Sign DH pubkey
	rng := csprng.NewSystemRNG()
	signed, err := rsa.Sign(rng, privateKeyRSA, sha, data, opts)
	if err != nil {
		return nil, nil, err
	}

	// Send signed public key and salt for UserID to Server
	host, ok := cl.receptionManager.Comms.GetHost(gwID)
	if !ok {
		return nil, nil, errors.Errorf("Failed to find host with ID %s", gwID.String())
	}

	nonceResponse, err := cl.receptionManager.Comms.
		SendRequestNonceMessage(host,
			&pb.NonceRequest{
				Salt:            salt,
				ClientRSAPubKey: string(rsa.CreatePublicKeyPem(publicKeyRSA)),
				ClientSignedByServer: &messages.RSASignature{
					Signature: regHash,
				},
				ClientDHPubKey: publicKeyDH.Bytes(),
				RequestSignature: &messages.RSASignature{
					Signature: signed,
				},
			}) // TODO: modify this to return server DH
	if err != nil {
		err := errors.New(fmt.Sprintf(
			"requestNonce: Unable to request nonce! %s", err))
		return nil, nil, err
	}
	if nonceResponse.Error != "" {
		err := errors.New(fmt.Sprintf("requestNonce: nonceResponse error: %s", nonceResponse.Error))
		return nil, nil, err
	}
	// Use Client keypair to sign Server nonce
	return nonceResponse.Nonce, nonceResponse.DHPubKey, nil

}

// confirmNonce is a helper for the Register function
// It signs a nonce and sends it for confirmation
// Returns nil if successful, error otherwise
func (cl *Client) confirmNonce(UID, nonce []byte,
	privateKeyRSA *rsa.PrivateKey, gwID *id.ID) error {
	sha := crypto.SHA256
	opts := rsa.NewDefaultOptions()
	opts.Hash = sha
	h := sha.New()
	h.Write(nonce)
	data := h.Sum(nil)

	// Hash nonce & sign
	sig, err := rsa.Sign(rand.Reader, privateKeyRSA, sha, data, opts)
	if err != nil {
		globals.Log.ERROR.Printf(
			"Register: Unable to sign nonce! %s", err)
		return err
	}

	// Send signed nonce to Server
	// TODO: This returns a receipt that can be used to speed up registration
	msg := &pb.RequestRegistrationConfirmation{
		UserID: UID,
		NonceSignedByClient: &messages.RSASignature{
			Signature: sig,
		},
	}

	host, ok := cl.receptionManager.Comms.GetHost(gwID)
	if !ok {
		return errors.Errorf("Failed to find host with ID %s", gwID.String())
	}
	confirmResponse, err := cl.receptionManager.Comms.
		SendConfirmNonceMessage(host, msg)
	if err != nil {
		err := errors.New(fmt.Sprintf(
			"confirmNonce: Unable to send signed nonce! %s", err))
		return err
	}
	if confirmResponse.Error != "" {
		err := errors.New(fmt.Sprintf(
			"confirmNonce: Error confirming nonce: %s", confirmResponse.Error))
		return err
	}
	return nil
}

func (cl *Client) registerUserE2E(partner *storage.Contact) error {

	// Check that the returned user is valid
	if partnerKeyStore := cl.session.GetKeyStore().GetSendManager(partner.Id); partnerKeyStore != nil {
		return errors.New(fmt.Sprintf("UDB searched failed for %v because user has "+
			"been searched for before", partner.Id))
	}

	userData, err := cl.sessionV2.GetUserData()
	if err != nil {
		return err
	}

	if userData.ThisUser.User.Cmp(partner.Id) {
		return errors.New("cannot search for yourself on UDB")
	}

	// Get needed variables from session
	grp := userData.E2EGrp
	userID := userData.ThisUser.User

	// Create user private key and partner public key
	// in the group
	privKeyCyclic := userData.E2EDHPrivateKey
	publicKeyCyclic := grp.NewIntFromBytes(partner.PublicKey)

	// Generate baseKey
	baseKey, _ := diffieHellman.CreateDHSessionKey(
		publicKeyCyclic,
		privKeyCyclic,
		grp)

	// Generate key TTL and number of keys
	params := cl.session.GetKeyStore().GetKeyParams()
	keysTTL, numKeys := e2e.GenerateKeyTTL(baseKey.GetLargeInt(),
		params.MinKeys, params.MaxKeys, params.TTLParams)

	// Create Send KeyManager
	km := keyStore.NewManager(baseKey, privKeyCyclic,
		publicKeyCyclic, partner.Id, true,
		numKeys, keysTTL, params.NumRekeys)

	// Generate Send Keys
	km.GenerateKeys(grp, userID)
	cl.session.GetKeyStore().AddSendManager(km)

	// Create Receive KeyManager
	km = keyStore.NewManager(baseKey, privKeyCyclic,
		publicKeyCyclic, partner.Id, false,
		numKeys, keysTTL, params.NumRekeys)

	// Generate Receive Keys
	newE2eKeys := km.GenerateKeys(grp, userID)
	cl.session.GetKeyStore().AddRecvManager(km)
	cl.session.GetKeyStore().AddReceiveKeysByFingerprint(newE2eKeys)

	// Create RekeyKeys and add to RekeyManager
	rkm := cl.session.GetRekeyManager()

	keys := &keyStore.RekeyKeys{
		CurrPrivKey: privKeyCyclic,
		CurrPubKey:  publicKeyCyclic,
	}

	rkm.AddKeys(partner.Id, keys)

	return nil
}

//GenerateKeys generates the keys and user information used in the session object
func (cl *Client) GenerateKeys(rsaPrivKey *rsa.PrivateKey,
	password string) error {

	cl.opStatus(globals.REG_KEYGEN)

	//Generate keys and other necessary session information
	cmixGrp, e2eGrp := generateGroups(cl.ndf)
	privKey, pubKey, err := generateRsaKeys(rsaPrivKey)
	if err != nil {
		return err
	}
	cmixPrivKey, cmixPubKey, err := generateCmixKeys(cmixGrp)
	if err != nil {
		return err
	}
	e2ePrivKey, e2ePubKey, err := generateE2eKeys(cmixGrp, e2eGrp)
	if err != nil {
		return err
	}

	//Set callback status to user generation & generate user
	cl.opStatus(globals.REG_UID_GEN)
	salt, _, usr, err := generateUserInformation(pubKey)
	if err != nil {
		return err
	}

	cl.session = user.NewSession(cl.storage, password)
	locA, _ := cl.storage.GetLocation()
	err = cl.setStorage(locA, password)
	if err != nil {
		return err
	}

	userData := &user2.UserData{
		ThisUser: &user2.User{
			User:     usr.User,
			Username: usr.Username,
			Precan:   usr.Precan,
		},
		RSAPrivateKey:    privKey,
		RSAPublicKey:     pubKey,
		CMIXDHPrivateKey: cmixPrivKey,
		CMIXDHPublicKey:  cmixPubKey,
		E2EDHPrivateKey:  e2ePrivKey,
		E2EDHPublicKey:   e2ePubKey,
		CmixGrp:          cmixGrp,
		E2EGrp:           e2eGrp,
		Salt:             salt,
	}
	err = cl.sessionV2.CommitUserData(userData)
	if err != nil {
		return err
	}
	err = cl.sessionV2.SetRegState(user.KeyGenComplete)
	if err != nil {
		return err
	}

	newRm, err := io.NewReceptionManager(cl.rekeyChan, cl.quitChan,
		usr.User,
		rsa.CreatePrivateKeyPem(privKey),
		rsa.CreatePublicKeyPem(pubKey),
		salt, cl.switchboard)
	if err != nil {
		return errors.Wrap(err, "Couldn't create reception manager")
	}
	if cl.receptionManager != nil {
		// Use the old comms manager if it exists
		newRm.Comms.Manager = cl.receptionManager.Comms.Manager
	}
	cl.receptionManager = newRm

	cl.session.SetE2EGrp(userData.E2EGrp)
	cl.session.SetUser(userData.ThisUser.User)

	//store the session
	return cl.session.StoreSession()
}

//GenerateGroups serves as a helper function for RegisterUser.
// It generates the cmix and e2e groups from the ndf
func generateGroups(clientNdf *ndf.NetworkDefinition) (cmixGrp, e2eGrp *cyclic.Group) {
	largeIntBits := 16

	//Generate the cmix group
	cmixGrp = cyclic.NewGroup(
		large.NewIntFromString(clientNdf.CMIX.Prime, largeIntBits),
		large.NewIntFromString(clientNdf.CMIX.Generator, largeIntBits))
	//Generate the e2e group
	e2eGrp = cyclic.NewGroup(
		large.NewIntFromString(clientNdf.E2E.Prime, largeIntBits),
		large.NewIntFromString(clientNdf.E2E.Generator, largeIntBits))

	return cmixGrp, e2eGrp
}

//GenerateRsaKeys serves as a helper function for RegisterUser.
// It generates a private key if the one passed in is nil and a public key from said private key
func generateRsaKeys(rsaPrivKey *rsa.PrivateKey) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	var err error
	//Generate client RSA keys
	if rsaPrivKey == nil {
		rsaPrivKey, err = rsa.GenerateKey(csprng.NewSystemRNG(), rsa.DefaultRSABitLen)
		if err != nil {
			return nil, nil, errors.Errorf("Could not generate RSA keys: %+v", err)
		}
	}
	//Pull the public key from the private key
	publicKeyRSA := rsaPrivKey.GetPublic()

	return rsaPrivKey, publicKeyRSA, nil
}

//GenerateCmixKeys serves as a helper function for RegisterUser.
// It generates private and public keys within the cmix group
func generateCmixKeys(cmixGrp *cyclic.Group) (cmixPrivateKeyDH, cmixPublicKeyDH *cyclic.Int, err error) {
	if cmixGrp == nil {
		return nil, nil, errors.New("Cannot have a nil CMix group")
	}

	//Generate the private key
	cmixPrivKeyDHByte, err := csprng.GenerateInGroup(cmixGrp.GetPBytes(), 256, csprng.NewSystemRNG())
	if err != nil {
		return nil, nil,
			errors.Errorf("Could not generate CMix DH keys: %+v", err)
	}
	//Convert the keys into cyclic Ints and return
	cmixPrivateKeyDH = cmixGrp.NewIntFromBytes(cmixPrivKeyDHByte)
	cmixPublicKeyDH = cmixGrp.ExpG(cmixPrivateKeyDH, cmixGrp.NewMaxInt())

	return cmixPrivateKeyDH, cmixPublicKeyDH, nil
}

//GenerateE2eKeys serves as a helper function for RegisterUser.
// It generates public and private keys used in e2e communications
func generateE2eKeys(cmixGrp, e2eGrp *cyclic.Group) (e2ePrivateKey, e2ePublicKey *cyclic.Int, err error) {
	if cmixGrp == nil || e2eGrp == nil {
		return nil, nil, errors.New("Cannot have a nil group")
	}
	//Generate the private key in group
	e2ePrivKeyDHByte, err := csprng.GenerateInGroup(cmixGrp.GetPBytes(), 256, csprng.NewSystemRNG())
	if err != nil {
		return nil, nil,
			errors.Errorf("Could not generate E2E DH keys: %s", err)
	}
	//Convert the keys into cyclic Ints and return
	e2ePrivateKeyDH := e2eGrp.NewIntFromBytes(e2ePrivKeyDHByte)
	e2ePublicKeyDH := e2eGrp.ExpG(e2ePrivateKeyDH, e2eGrp.NewMaxInt())

	return e2ePrivateKeyDH, e2ePublicKeyDH, nil
}

//generateUserInformation serves as a helper function for RegisterUser.
// It generates a salt s.t. it can create a user and their ID
func generateUserInformation(publicKeyRSA *rsa.PublicKey) ([]byte, *id.ID,
	*user2.User, error) {
	//Generate salt for UserID
	salt := make([]byte, SaltSize)
	_, err := csprng.NewSystemRNG().Read(salt)
	if err != nil {
		return nil, nil, nil,
			errors.Errorf("Register: Unable to generate salt! %s", err)
	}

	//Generate UserID by hashing salt and public key
	userId, err := xx.NewID(publicKeyRSA, salt, id.User)
	if err != nil {
		return nil, nil, nil, err
	}

	usr := userRegistry.Users.NewUser(userId, "")
	userRegistry.Users.UpsertUser(usr)

	return salt, userId, usr, nil
}
