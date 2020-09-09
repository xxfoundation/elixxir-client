package node

import (
	"crypto"
	"crypto/rand"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/client/context/stoppable"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/cmix"
	"gitlab.com/elixxir/client/storage/user"
	"gitlab.com/elixxir/comms/client"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/registration"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"time"
)

func StartRegistration(ctx context.Context, comms client.Comms) stoppable.Stoppable {
	instance := ctx.Manager.GetInstance()

	c := make(chan ndf.Gateway, 100)
	instance.SetAddGatewayChan(c)

	u := ctx.Session.User()
	regSignature := u.GetRegistrationValidationSignature()
	userCryptographicIdentity := u.GetCryptographicIdentity()
	ctx.Session.Cmix()

	stop := stoppable.NewSingle("NodeRegistration")

	go func() {
		for true {
			select {
			case <-stop.Quit():
				return
			case gw := <-c:
				err := registerWithNode(comms, gw, regSignature, userCryptographicIdentity,
					ctx.Session.Cmix(), ctx.Session)
				if err != nil {
					jwalterweatherman.ERROR.Printf("Failed")
				}
			default:
				time.Sleep(0.5)
			}
		}
	}()

	return stop

}

//registerWithNode serves as a helper for RegisterWithNodes
// It registers a user with a specific in the client's ndf.
func registerWithNode(comms client.Comms, gw ndf.Gateway, regHash []byte,
	userCryptographicIdentity *user.CryptographicIdentity, store *cmix.Store, session *storage.Session) error {

	gatewayID, err := id.Unmarshal(gw.ID)
	if err != nil {
		return err
	}

	// Initialise blake2b hash for transmission keys and sha256 for reception
	// keys
	transmissionHash, _ := hash.NewCMixHash()

	nonce, dhPub, err := requestNonce(comms, gatewayID, regHash, userCryptographicIdentity, store)

	// Load server DH pubkey
	serverPubDH := store.GetGroup().NewIntFromBytes(dhPub)

	// Confirm received nonce
	globals.Log.INFO.Println("Register: Confirming received nonce")
	err = confirmNonce(comms, userCryptographicIdentity.GetUserID().Bytes(),
		nonce, userCryptographicIdentity.GetRSA(), gatewayID)
	if err != nil {
		errMsg := fmt.Sprintf("Register: Unable to confirm nonce: %v", err)
		return errors.New(errMsg)
	}

	nodeID := gatewayID.DeepCopy()
	nodeID.SetType(id.Node)
	transmissionKey := registration.GenerateBaseKey(store.GetGroup(),
		serverPubDH, store.GetDHPrivateKey(), transmissionHash)
	session.Cmix().Add(nodeID, transmissionKey)

	return nil
}

func requestNonce(comms client.Comms, gwId *id.ID, regHash []byte,
	userCryptographicIdentity *user.CryptographicIdentity, store *cmix.Store) ([]byte, []byte, error) {
	dhPub := store.GetDHPublicKey().Bytes()
	sha := crypto.SHA256
	opts := rsa.NewDefaultOptions()
	opts.Hash = sha
	h := sha.New()
	h.Write(dhPub)
	data := h.Sum(nil)

	// Sign DH pubkey
	rng := csprng.NewSystemRNG()
	signed, err := rsa.Sign(rng, userCryptographicIdentity.GetRSA(), sha, data, opts)
	if err != nil {
		return nil, nil, err
	}

	// Request nonce message from gateway
	globals.Log.INFO.Printf("Register: Requesting nonce from gateway %v",
		gwId.Bytes())

	host, ok := comms.GetHost(gwId)
	if !ok {
		return nil, nil, errors.Errorf("Failed to find host with ID %s", gwId.String())
	}
	nonceResponse, err := comms.SendRequestNonceMessage(host,
		&pb.NonceRequest{
			Salt:            userCryptographicIdentity.GetSalt(),
			ClientRSAPubKey: string(rsa.CreatePublicKeyPem(userCryptographicIdentity.GetRSA().GetPublic())),
			ClientSignedByServer: &messages.RSASignature{
				Signature: regHash,
			},
			ClientDHPubKey: dhPub,
			RequestSignature: &messages.RSASignature{
				Signature: signed,
			},
		})

	if err != nil {
		errMsg := fmt.Sprintf("Register: Failed requesting nonce from gateway: %+v", err)
		return nil, nil, errors.New(errMsg)
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
func confirmNonce(comms client.Comms, UID, nonce []byte,
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

	host, ok := comms.GetHost(gwID)
	if !ok {
		return errors.Errorf("Failed to find host with ID %s", gwID.String())
	}
	confirmResponse, err := comms.SendConfirmNonceMessage(host, msg)
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
