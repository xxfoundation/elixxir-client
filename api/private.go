package api

import (
	"crypto"
	"crypto/rand"
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/io"
	"gitlab.com/elixxir/client/keyStore"
	"gitlab.com/elixxir/client/user"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/id"
)

const PermissioningAddrID = "Permissioning"

// precannedRegister is a helper function for Register
// It handles the precanned registration case
func (cl *Client) precannedRegister(registrationCode, nick string,
	nk map[id.Node]user.NodeKeys) (*user.User, *id.User, map[id.Node]user.NodeKeys, error) {
	var successLook bool
	var UID *id.User
	var u *user.User
	var err error

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

	if nick != "" {
		u.Nick = nick
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
// It sends a registration message and returns the registration hash
func (cl *Client) sendRegistrationMessage(registrationCode string,
	publicKeyRSA *rsa.PublicKey) ([]byte, error) {
	connected, err := cl.commManager.ConnectToPermissioning()
	defer cl.commManager.DisconnectFromPermissioning()
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't connect to permissioning to send registration message")
	}
	if !connected {
		return nil, errors.New("Didn't connect to permissioning to send registration message. Check the NDF")
	}
	regHash := make([]byte, 0)
	// Send registration code and public key to RegistrationServer
	response, err := cl.commManager.Comms.
		SendRegistrationMessage(io.ConnAddr(PermissioningAddrID),
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
	regHash = response.ClientSignedByServer.Signature
	// Disconnect from regServer here since it will not be needed
	return regHash, nil
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
	fmt.Println(data)
	// Sign DH pubkey
	rng := csprng.NewSystemRNG()
	signed, err := rsa.Sign(rng, privateKeyRSA, sha, data, opts)
	if err != nil {
		return nil, nil, err
	}

	// Send signed public key and salt for UserID to Server
	nonceResponse, err := cl.commManager.Comms.
		SendRequestNonceMessage(gwID,
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
	confirmResponse, err := cl.commManager.Comms.
		SendConfirmNonceMessage(gwID, msg)
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
	km.GenerateKeys(grp, userID, cl.session.GetKeyStore())

	// Create Receive KeyManager
	km = keyStore.NewManager(baseKey, privKeyCyclic,
		partnerPubKeyCyclic, partnerID, false,
		numKeys, keysTTL, params.NumRekeys)

	// Generate Receive Keys
	km.GenerateKeys(grp, userID, cl.session.GetKeyStore())

	// Create RekeyKeys and add to RekeyManager
	rkm := cl.session.GetRekeyManager()

	keys := &keyStore.RekeyKeys{
		CurrPrivKey: privKeyCyclic,
		CurrPubKey:  partnerPubKeyCyclic,
	}

	rkm.AddKeys(partnerID, keys)

	return nil
}
