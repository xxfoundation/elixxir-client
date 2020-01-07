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
	"gitlab.com/elixxir/client/keyStore"
	"gitlab.com/elixxir/client/user"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/registration"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
)

const PermissioningAddrID = "Permissioning"

// precannedRegister is a helper function for Register
// It handles the precanned registration case
func (cl *Client) precannedRegister(registrationCode string) (*user.User, *id.User, map[id.Node]user.NodeKeys, error) {
	var successLook bool
	var UID *id.User
	var u *user.User
	var err error

	nk := make(map[id.Node]user.NodeKeys)

	UID, successLook = user.Users.LookupUser(registrationCode)

	globals.Log.DEBUG.Printf("UID: %+v, success: %+v", UID, successLook)

	if !successLook {
		return nil, nil, nil, errors.New("precannedRegister: could not register due to invalid HUID")
	}

	var successGet bool
	u, successGet = user.Users.GetUser(UID)

	if !successGet {
		err = errors.New("precannedRegister: could not register due to ID lookup failure")
		return nil, nil, nil, err
	}

	nodekeys, successKeys := user.Users.LookupKeys(u.User)

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
	err := AddPermissioningHost(cl.receptionManager, cl.ndf)

	if err != nil {
		if err == ErrNoPermissioning {
			return nil, errors.New("Didn't connect to permissioning to send registration message. Check the NDF")
		}
		return nil, errors.Wrap(err, "Couldn't connect to permissioning to send registration message")
	}

	regValidationSignature := make([]byte, 0)
	// Send registration code and public key to RegistrationServer
	host, ok := cl.receptionManager.Comms.GetHost(PermissioningAddrID)
	if !ok {
		return nil, errors.New("Failed to find permissioning host")
	}
	fmt.Println("in reg, pub key ", publicKeyRSA)
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
	privateKeyRSA *rsa.PrivateKey, gwID *id.Gateway) ([]byte, []byte, error) {
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
	host, ok := cl.receptionManager.Comms.GetHost(gwID.String())
	if !ok {
		return nil, nil, errors.Errorf("Failed to find host with ID %s", gwID.String())
	}
	nonceResponse, err := cl.receptionManager.Comms.
		SendRequestNonceMessage(host,
			&pb.NonceRequest{
				Salt:            salt,
				ClientRSAPubKey: string(rsa.CreatePublicKeyPem(publicKeyRSA)),
				ClientSignedByServer: &pb.RSASignature{
					Signature: regHash,
				},
				ClientDHPubKey: publicKeyDH.Bytes(),
				RequestSignature: &pb.RSASignature{
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
	privateKeyRSA *rsa.PrivateKey, gwID *id.Gateway) error {
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
		NonceSignedByClient: &pb.RSASignature{
			Signature: sig,
		},
	}

	host, ok := cl.receptionManager.Comms.GetHost(gwID.String())
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

func (cl *Client) registerUserE2E(partnerID *id.User,
	partnerPubKey []byte) error {

	// Check that the returned user is valid
	if partnerKeyStore := cl.session.GetKeyStore().GetSendManager(partnerID); partnerKeyStore != nil {
		return errors.New(fmt.Sprintf("UDB searched failed for %v because user has "+
			"been searched for before", partnerID))
	}

	if cl.session.GetCurrentUser().User.Cmp(partnerID) {
		return errors.New("cannot search for yourself on UDB")
	}

	// Get needed variables from session
	grp := cl.session.GetE2EGroup()
	userID := cl.session.GetCurrentUser().User

	// Create user private key and partner public key
	// in the group
	privKeyCyclic := cl.session.GetE2EDHPrivateKey()
	partnerPubKeyCyclic := grp.NewIntFromBytes(partnerPubKey)

	// Generate baseKey
	baseKey, _ := diffieHellman.CreateDHSessionKey(
		partnerPubKeyCyclic,
		privKeyCyclic,
		grp)

	// Generate key TTL and number of keys
	params := cl.session.GetKeyStore().GetKeyParams()
	keysTTL, numKeys := e2e.GenerateKeyTTL(baseKey.GetLargeInt(),
		params.MinKeys, params.MaxKeys, params.TTLParams)

	// Create Send KeyManager
	km := keyStore.NewManager(baseKey, privKeyCyclic,
		partnerPubKeyCyclic, partnerID, true,
		numKeys, keysTTL, params.NumRekeys)

	// Generate Send Keys
	km.GenerateKeys(grp, userID)
	cl.session.GetKeyStore().AddSendManager(km)

	// Create Receive KeyManager
	km = keyStore.NewManager(baseKey, privKeyCyclic,
		partnerPubKeyCyclic, partnerID, false,
		numKeys, keysTTL, params.NumRekeys)

	// Generate Receive Keys
	newE2eKeys := km.GenerateKeys(grp, userID)
	cl.session.GetKeyStore().AddRecvManager(km)
	cl.session.GetKeyStore().AddReceiveKeysByFingerprint(newE2eKeys)

	// Create RekeyKeys and add to RekeyManager
	rkm := cl.session.GetRekeyManager()

	keys := &keyStore.RekeyKeys{
		CurrPrivKey: privKeyCyclic,
		CurrPubKey:  partnerPubKeyCyclic,
	}

	rkm.AddKeys(partnerID, keys)

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

	cl.session = user.NewSession(cl.storage, usr, pubKey, privKey, cmixPubKey,
		cmixPrivKey, e2ePubKey, e2ePrivKey, salt, cmixGrp, e2eGrp, password)

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
func generateUserInformation(publicKeyRSA *rsa.PublicKey) ([]byte, *id.User, *user.User, error) {
	//Generate salt for UserID
	salt := make([]byte, SaltSize)
	_, err := csprng.NewSystemRNG().Read(salt)
	if err != nil {
		return nil, nil, nil,
			errors.Errorf("Register: Unable to generate salt! %s", err)
	}

	//Generate UserID by hashing salt and public key
	userId := registration.GenUserID(publicKeyRSA, salt)

	usr := user.Users.NewUser(userId, "")
	user.Users.UpsertUser(usr)

	return salt, userId, usr, nil
}
