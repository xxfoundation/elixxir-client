///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package node

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/cmix"
	"gitlab.com/elixxir/client/storage/user"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/registration"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/chacha"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/tls"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"strconv"
	"time"
)

type RegisterNodeCommsInterface interface {
	SendRequestClientKeyMessage(host *connect.Host,
		message *pb.SignedClientKeyRequest) (*pb.SignedKeyResponse, error)

	// ---------------------- Start of deprecated fields ----------- //
	// TODO: Remove once RequestClientKey is properly tested
	SendRequestNonceMessage(host *connect.Host, message *pb.NonceRequest) (*pb.Nonce, error)
	SendConfirmNonceMessage(host *connect.Host,
		message *pb.RequestRegistrationConfirmation) (*pb.RegistrationConfirmation, error)
	// ---------------------- End of deprecated fields ----------- //

}

func StartRegistration(sender *gateway.Sender, session *storage.Session, rngGen *fastRNG.StreamGenerator, comms RegisterNodeCommsInterface,
	c chan network.NodeGateway, numParallel uint) stoppable.Stoppable {

	multi := stoppable.NewMulti("NodeRegistrations")

	for i := uint(0); i < numParallel; i++ {
		stop := stoppable.NewSingle(fmt.Sprintf("NodeRegistration %d", i))

		go registerNodes(sender, session, rngGen, comms, stop, c)
		multi.Add(stop)
	}

	return multi
}

func registerNodes(sender *gateway.Sender, session *storage.Session,
	rngGen *fastRNG.StreamGenerator, comms RegisterNodeCommsInterface,
	stop *stoppable.Single, c chan network.NodeGateway) {
	u := session.User()
	regSignature := u.GetTransmissionRegistrationValidationSignature()
	// Timestamp in which user has registered with registration
	regTimestamp := u.GetRegistrationTimestamp().UnixNano()
	uci := u.GetCryptographicIdentity()
	cmix := session.Cmix()

	rng := rngGen.GetStream()
	interval := time.Duration(500) * time.Millisecond
	t := time.NewTicker(interval)
	for {
		select {
		case <-stop.Quit():
			t.Stop()
			stop.ToStopped()
			return
		case gw := <-c:
			err := registerWithNode(sender, comms, gw, regSignature,
				regTimestamp, uci, cmix, rng, stop)
			if err != nil {
				jww.ERROR.Printf("Failed to register node: %+v", err)
			}
		case <-t.C:
		}
	}
}

//registerWithNode serves as a helper for RegisterWithNodes
// It registers a user with a specific in the client's ndf.
func registerWithNode(sender *gateway.Sender, comms RegisterNodeCommsInterface,
	ngw network.NodeGateway, regSig []byte, registrationTimestampNano int64,
	uci *user.CryptographicIdentity, store *cmix.Store, rng csprng.Source,
	stop *stoppable.Single) error {

	nodeID, err := ngw.Node.GetNodeId()
	if err != nil {
		jww.ERROR.Println("registerWithNode() failed to decode nodeId")
		return err
	}

	gwid := ngw.Gateway.ID
	gatewayID, err := id.Unmarshal(gwid)
	if err != nil {
		jww.ERROR.Println("registerWithNode() failed to decode gatewayID")
		return err
	}

	if store.IsRegistered(nodeID) {
		return nil
	}

	jww.INFO.Printf("registerWithNode() begin registration with node: %s", nodeID)

	var transmissionKey *cyclic.Int
	// TODO: should move this to a precanned user initialization
	if uci.IsPrecanned() {
		userNum := int(uci.GetTransmissionID().Bytes()[7])
		h := sha256.New()
		h.Reset()
		h.Write([]byte(strconv.Itoa(4000 + userNum)))

		transmissionKey = store.GetGroup().NewIntFromBytes(h.Sum(nil))
		jww.INFO.Printf("transmissionKey: %v", transmissionKey.Bytes())
	} else {
		// Request key from server
		signedKeyResponse, err := requestKey(sender, comms, gatewayID, regSig,
			registrationTimestampNano, uci, store, rng, stop)

		if err != nil {
			// TODO: remove old codepath when new registration path is properly tested
			jww.WARN.Printf("Databaseless registration failed, attempting soon "+
				"to be deprecated code path. Error: %v", err)
			transmissionKey, err = registerDepreciated(sender, comms, regSig,
				registrationTimestampNano, uci, store, rng, stop, gatewayID, nodeID)
			return errors.Errorf("Failed to request nonce: %+v", err)
		}

		// Hash the response
		opts := rsa.NewDefaultOptions()
		h := opts.Hash.New()
		h.Write(signedKeyResponse.KeyResponse)
		hashedResponse := h.Sum(nil)

		// Load node certificate
		nodeCert, err := tls.LoadCertificate(ngw.Node.TlsCertificate)
		if err != nil {
			return errors.WithMessagef(err, "Unable to load node's certificate")
		}

		// Extract public key
		nodePubKey, err := tls.ExtractPublicKey(nodeCert)
		if err != nil {
			return errors.WithMessagef(err, "Unable to load node's public key")
		}

		// Verify the response signature
		err = rsa.Verify(nodePubKey, opts.Hash, hashedResponse,
			signedKeyResponse.KeyResponseSignedByNode.Signature, opts)
		if err != nil {
			return errors.WithMessagef(err, "Could not verify node's signature")
		}

		// Unmarshal the response
		keyResponse := &pb.ClientKeyResponse{}
		err = proto.Unmarshal(signedKeyResponse.KeyResponse, keyResponse)
		if err != nil {
			return errors.WithMessagef(err, "Failed to unmarshal client key response")
		}

		h.Reset()

		// Convert Node DH Public key to a cyclic.Int
		grp := store.GetGroup()
		nodeDHPub := grp.NewIntFromBytes(keyResponse.NodeDHPubKey)

		// Construct the session key
		sessionKey := registration.GenerateBaseKey(grp,
			nodeDHPub, store.GetDHPrivateKey(), h)



		// Verify the HMAC
		h.Reset()
		if !registration.VerifyClientHMAC(sessionKey.Bytes(), keyResponse.EncryptedClientKey,
			h, keyResponse.EncryptedClientKeyHMAC) {
			return errors.WithMessagef(err, "Failed to verify client HMAC")
		}

		// Decrypt the client key
		clientKey, err := chacha.Decrypt(sessionKey.Bytes(), keyResponse.EncryptedClientKey)
		if err != nil {
			return errors.WithMessagef(err, "Failed to decrypt client key")
		}

		// Construct the transmission key from the client key
		transmissionKey = store.GetGroup().NewIntFromBytes(clientKey)
	}

	store.Add(nodeID, transmissionKey)

	jww.INFO.Printf("Completed registration with node %s", nodeID)

	return nil
}

func requestKey(sender *gateway.Sender, comms RegisterNodeCommsInterface, gwId *id.ID,
	regSig []byte, registrationTimestampNano int64, uci *user.CryptographicIdentity,
	store *cmix.Store, rng csprng.Source, stop *stoppable.Single) (keyResponse *pb.SignedKeyResponse, err error) {

	dhPub := store.GetDHPublicKey().Bytes()

	keyRequest := &pb.ClientKeyRequest{
		Salt: uci.GetTransmissionSalt(),
		ClientTransmissionConfirmation: &pb.SignedRegistrationConfirmation{
			RegistrarSignature: &messages.RSASignature{Signature: regSig},
		},
		ClientDHPubKey:        dhPub,
		RegistrationTimestamp: registrationTimestampNano,
		RequestTimestamp:      netTime.Now().UnixNano(),
	}

	serializedMessage, err := proto.Marshal(keyRequest)
	if err != nil {
		return nil, err
	}

	opts := rsa.NewDefaultOptions()
	opts.Hash = hash.CMixHash
	h := opts.Hash.New()
	h.Write(serializedMessage)
	data := h.Sum(nil)

	// Sign DH pubkey
	clientSig, err := rsa.Sign(rng, uci.GetTransmissionRSA(), opts.Hash,
		data, opts)
	if err != nil {
		return nil, err
	}

	// Request nonce message from gateway
	jww.INFO.Printf("Register: Requesting nonce from gateway %v", gwId.String())

	result, err := sender.SendToAny(func(host *connect.Host) (interface{}, error) {
		nonceResponse, err := comms.SendRequestClientKeyMessage(host,
			&pb.SignedClientKeyRequest{
				ClientKeyRequest:          serializedMessage,
				ClientKeyRequestSignature: &messages.RSASignature{Signature: clientSig},
				Target:                    gwId.Bytes(),
			})
		if err != nil {
			return nil, errors.WithMessage(err, "Register: Failed requesting nonce from gateway")
		}
		if nonceResponse.Error != "" {
			return nil, errors.WithMessage(err, "requestKey: nonceResponse error")
		}
		return nonceResponse, nil
	}, stop)

	if err != nil {
		return nil, err
	}

	response := result.(*pb.SignedKeyResponse)
	if response.Error != "" {
		return nil, errors.New(response.Error)
	}

	// Use Client keypair to sign Server nonce
	return response, nil
}

// ---------------------- Start of deprecated fields ----------- //

// registerDepreciated is a DEPRECATED codepath that registers a user via
// the request/confirmNonce codepath. This is left for backward compatibility
// and will be removed.
// TODO: Remove this once RequestClientKey is properly tested
func registerDepreciated(sender *gateway.Sender, comms RegisterNodeCommsInterface,
	regSig []byte, registrationTimestampNano int64, uci *user.CryptographicIdentity,
	store *cmix.Store, rng csprng.Source, stop *stoppable.Single,
	gatewayID, nodeID *id.ID) (*cyclic.Int, error) {

	jww.WARN.Printf("DEPRECATED: Registering using soon to be deprecated code path")

	transmissionHash, _ := hash.NewCMixHash()

	// Register nonce
	nonce, dhPub, err := requestNonce(sender, comms, gatewayID, regSig,
		registrationTimestampNano, uci, store, rng, stop)
	if err != nil {
		return nil, err
	}

	// Load server DH pubkey
	serverPubDH := store.GetGroup().NewIntFromBytes(dhPub)

	// Confirm received nonce
	jww.INFO.Printf("Register: Confirming received nonce from node %s", nodeID.String())
	err = confirmNonce(sender, comms, uci.GetTransmissionID().Bytes(),
		nonce, uci.GetTransmissionRSA(), gatewayID, stop)

	if err != nil {
		errMsg := fmt.Sprintf("Register: Unable to confirm nonce: %v", err)
		return nil, errors.New(errMsg)
	}
	transmissionKey := registration.GenerateBaseKey(store.GetGroup(),
		serverPubDH, store.GetDHPrivateKey(), transmissionHash)

	return transmissionKey, err
}

// WARNING DEPRECATED: requestNonce will soon be deprecated and removed. This will only
// be used for testing with backwards compatibility.
// TODO: Remove this once RequestClientKey is properly tested
func requestNonce(sender *gateway.Sender, comms RegisterNodeCommsInterface, gwId *id.ID,
	regSig []byte, registrationTimestampNano int64, uci *user.CryptographicIdentity,
	store *cmix.Store, rng csprng.Source, stop *stoppable.Single) ([]byte, []byte, error) {

	jww.WARN.Printf("DEPRECATED: Registering with a soon to be deprecated function")

	dhPub := store.GetDHPublicKey().Bytes()
	opts := rsa.NewDefaultOptions()
	opts.Hash = hash.CMixHash
	h, _ := hash.NewCMixHash()
	h.Write(dhPub)
	data := h.Sum(nil)

	// Sign DH pubkey
	clientSig, err := rsa.Sign(rng, uci.GetTransmissionRSA(), opts.Hash,
		data, opts)
	if err != nil {
		return nil, nil, err
	}

	// Request nonce message from gateway
	jww.INFO.Printf("Register: Requesting nonce from gateway %v", gwId.String())

	result, err := sender.SendToAny(func(host *connect.Host) (interface{}, error) {
		nonceResponse, err := comms.SendRequestNonceMessage(host,
			&pb.NonceRequest{
				Salt:            uci.GetTransmissionSalt(),
				ClientRSAPubKey: string(rsa.CreatePublicKeyPem(uci.GetTransmissionRSA().GetPublic())),
				ClientSignedByServer: &messages.RSASignature{
					Signature: regSig,
				},
				ClientDHPubKey: dhPub,
				RequestSignature: &messages.RSASignature{
					Signature: clientSig,
				},
				Target: gwId.Marshal(),
				// Timestamp in which user has registered with registration
				TimeStamp: registrationTimestampNano,
			})
		if err != nil {
			return nil, errors.WithMessage(err, "Register: Failed requesting nonce from gateway")
		}
		if nonceResponse.Error != "" {
			return nil, errors.WithMessage(err, "requestNonce: nonceResponse error")
		}
		return nonceResponse, nil
	}, stop)

	if err != nil {
		return nil, nil, err
	}

	nonceResponse := result.(*pb.Nonce)

	// Use Client keypair to sign Server nonce
	return nonceResponse.Nonce, nonceResponse.DHPubKey, nil
}

// WARNING DEPRECATED: confirmNonce will soon be deprecated and removed. This will only
// be used for testing with backwards compatibility.
// confirmNonce is a helper for the Register function
// It signs a nonce and sends it for confirmation
// Returns nil if successful, error otherwise
// TODO: Remove this once RequestClientKey is properly tested
func confirmNonce(sender *gateway.Sender, comms RegisterNodeCommsInterface, UID,
	nonce []byte, privateKeyRSA *rsa.PrivateKey, gwID *id.ID,
	stop *stoppable.Single) error {
	jww.WARN.Printf("DEPRECATED: ConfirmNonce is a soon to be deprecated function")

	opts := rsa.NewDefaultOptions()
	opts.Hash = hash.CMixHash
	h, _ := hash.NewCMixHash()
	h.Write(nonce)
	// Hash the ID of the node we are sending to
	nodeId := gwID.DeepCopy()
	nodeId.SetType(id.Node)
	h.Write(nodeId.Bytes())
	data := h.Sum(nil)

	// Hash nonce & sign
	sig, err := rsa.Sign(rand.Reader, privateKeyRSA, opts.Hash, data, opts)
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
		Target: gwID.Marshal(),
	}

	_, err = sender.SendToAny(func(host *connect.Host) (interface{}, error) {
		confirmResponse, err := comms.SendConfirmNonceMessage(host, msg)
		if err != nil {
			return nil, err
		} else if confirmResponse.Error != "" {
			err := errors.New(fmt.Sprintf(
				"confirmNonce: Error confirming nonce: %s", confirmResponse.Error))
			return nil, err
		}
		return confirmResponse, nil
	}, stop)

	return err
}

// ---------------------- End of deprecated fields ----------- //
