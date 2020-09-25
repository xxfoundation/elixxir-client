package node

import (
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/cmix"
	"gitlab.com/elixxir/client/storage/user"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/registration"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

type RegisterNodeCommsInterface interface {
	GetHost(hostId *id.ID) (*connect.Host, bool)
	SendRequestNonceMessage(host *connect.Host,
		message *pb.NonceRequest) (*pb.Nonce, error)
	SendConfirmNonceMessage(host *connect.Host,
		message *pb.RequestRegistrationConfirmation) (*pb.RegistrationConfirmation, error)
}

func StartRegistration(instance *network.Instance, session *storage.Session, rngGen *fastRNG.StreamGenerator, comms RegisterNodeCommsInterface,
	c chan network.NodeGateway) stoppable.Stoppable {
	stop := stoppable.NewSingle("NodeRegistration")

	instance.SetAddGatewayChan(c)

	go registerNodes(session, rngGen, comms, stop, c)

	return stop
}

func registerNodes(session *storage.Session, rngGen *fastRNG.StreamGenerator, comms RegisterNodeCommsInterface,
	stop *stoppable.Single, c chan network.NodeGateway) {
	u := session.User()
	regSignature := u.GetRegistrationValidationSignature()
	uci := u.GetCryptographicIdentity()
	cmix := session.Cmix()

	rng := rngGen.GetStream()
	interval := time.Duration(500) * time.Millisecond
	t := time.NewTicker(interval)
	for true {
		select {
		case <-stop.Quit():
			t.Stop()
			return
		case gw := <-c:
			err := registerWithNode(comms, gw, regSignature, uci, cmix, rng)
			if err != nil {
				jww.ERROR.Printf("Failed to register node: %+v", err)
			}
		case <-t.C:
		}
	}

}

//registerWithNode serves as a helper for RegisterWithNodes
// It registers a user with a specific in the client's ndf.
func registerWithNode(comms RegisterNodeCommsInterface, ngw network.NodeGateway, regSig []byte,
	uci *user.CryptographicIdentity, store *cmix.Store, rng csprng.Source) error {
	gw := ngw.Gateway
	gatewayID, err := id.Unmarshal(gw.ID)
	if err != nil {
		return err
	}

	nodeID := gatewayID.DeepCopy()
	nodeID.SetType(id.Node)

	if store.IsRegistered(nodeID) {
		return nil
	}

	var transmissionKey *cyclic.Int
	// TODO: should move this to a precanned user initialization
	if uci.IsPrecanned() {
		userNum := binary.BigEndian.Uint64(uci.GetUserID().Bytes())
		h := sha256.New()
		h.Reset()
		h.Write([]byte(string(40000 + userNum)))

		transmissionKey = store.GetGroup().NewIntFromBytes(h.Sum(nil))
	} else {
		// Initialise blake2b hash for transmission keys and sha256 for reception
		// keys
		transmissionHash, _ := hash.NewCMixHash()

		nonce, dhPub, err := requestNonce(comms, gatewayID, regSig, uci, store, rng)
		if err != nil {
			return errors.Errorf("Failed to request nonce: %+v", err)
		}

		// Load server DH pubkey
		serverPubDH := store.GetGroup().NewIntFromBytes(dhPub)

		// Confirm received nonce
		jww.INFO.Println("Register: Confirming received nonce")
		err = confirmNonce(comms, uci.GetUserID().Bytes(),
			nonce, uci.GetRSA(), gatewayID)
		if err != nil {
			errMsg := fmt.Sprintf("Register: Unable to confirm nonce: %v", err)
			return errors.New(errMsg)
		}
		transmissionKey = registration.GenerateBaseKey(store.GetGroup(),
			serverPubDH, store.GetDHPrivateKey(), transmissionHash)
	}

	store.Add(nodeID, transmissionKey)

	return nil
}

func requestNonce(comms RegisterNodeCommsInterface, gwId *id.ID, regHash []byte,
	uci *user.CryptographicIdentity, store *cmix.Store, rng csprng.Source) ([]byte, []byte, error) {
	dhPub := store.GetDHPublicKey().Bytes()
	sha := crypto.SHA256
	opts := rsa.NewDefaultOptions()
	opts.Hash = sha
	h := sha.New()
	h.Write(dhPub)
	data := h.Sum(nil)

	// Sign DH pubkey
	clientSig, err := rsa.Sign(rng, uci.GetRSA(), sha, data, opts)
	if err != nil {
		return nil, nil, err
	}

	// Request nonce message from gateway
	jww.INFO.Printf("Register: Requesting nonce from gateway %v",
		gwId.Bytes())

	host, ok := comms.GetHost(gwId)
	if !ok {
		return nil, nil, errors.Errorf("Failed to find host with ID %s", gwId.String())
	}
	nonceResponse, err := comms.SendRequestNonceMessage(host,
		&pb.NonceRequest{
			Salt:            uci.GetSalt(),
			ClientRSAPubKey: string(rsa.CreatePublicKeyPem(uci.GetRSA().GetPublic())),
			ClientSignedByServer: &messages.RSASignature{
				Signature: regHash,
			},
			ClientDHPubKey: dhPub,
			RequestSignature: &messages.RSASignature{
				Signature: clientSig,
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
func confirmNonce(comms RegisterNodeCommsInterface, UID, nonce []byte,
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
		jww.ERROR.Printf(
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
