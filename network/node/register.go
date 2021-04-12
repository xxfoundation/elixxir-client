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
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"strconv"
	"time"
)

type RegisterNodeCommsInterface interface {
	SendRequestNonceMessage(host *connect.Host,
		message *pb.NonceRequest) (*pb.Nonce, error)
	SendConfirmNonceMessage(host *connect.Host,
		message *pb.RequestRegistrationConfirmation) (*pb.RegistrationConfirmation, error)
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

func registerNodes(sender *gateway.Sender, session *storage.Session, rngGen *fastRNG.StreamGenerator, comms RegisterNodeCommsInterface,
	stop *stoppable.Single, c chan network.NodeGateway) {
	u := session.User()
	regSignature := u.GetTransmissionRegistrationValidationSignature()
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
			err := registerWithNode(sender, comms, gw, regSignature, uci, cmix, rng)
			if err != nil {
				jww.ERROR.Printf("Failed to register node: %+v", err)
			}
		case <-t.C:
		}
	}
}

//registerWithNode serves as a helper for RegisterWithNodes
// It registers a user with a specific in the client's ndf.
func registerWithNode(sender *gateway.Sender, comms RegisterNodeCommsInterface, ngw network.NodeGateway, regSig []byte,
	uci *user.CryptographicIdentity, store *cmix.Store, rng csprng.Source) error {
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
		// Initialise blake2b hash for transmission keys and reception
		// keys
		transmissionHash, _ := hash.NewCMixHash()

		nonce, dhPub, err := requestNonce(sender, comms, gatewayID, regSig, uci, store, rng)
		if err != nil {
			return errors.Errorf("Failed to request nonce: %+v", err)
		}

		// Load server DH pubkey
		serverPubDH := store.GetGroup().NewIntFromBytes(dhPub)

		// Confirm received nonce
		jww.INFO.Println("Register: Confirming received nonce from node %s", nodeID.String())
		err = confirmNonce(sender, comms, uci.GetTransmissionID().Bytes(),
			nonce, uci.GetTransmissionRSA(), gatewayID)
		if err != nil {
			errMsg := fmt.Sprintf("Register: Unable to confirm nonce: %v", err)
			return errors.New(errMsg)
		}
		transmissionKey = registration.GenerateBaseKey(store.GetGroup(),
			serverPubDH, store.GetDHPrivateKey(), transmissionHash)
	}

	store.Add(nodeID, transmissionKey)

	jww.INFO.Printf("Completed registration with node %s", nodeID)

	return nil
}

func requestNonce(sender *gateway.Sender, comms RegisterNodeCommsInterface, gwId *id.ID, regHash []byte,
	uci *user.CryptographicIdentity, store *cmix.Store, rng csprng.Source) ([]byte, []byte, error) {
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
	jww.INFO.Printf("Register: Requesting nonce from gateway %v",
		gwId.Bytes())

	result, err := sender.SendToAny(func(host *connect.Host) (interface{}, error) {
		nonceResponse, err := comms.SendRequestNonceMessage(host,
			&pb.NonceRequest{
				Salt:            uci.GetTransmissionSalt(),
				ClientRSAPubKey: string(rsa.CreatePublicKeyPem(uci.GetTransmissionRSA().GetPublic())),
				ClientSignedByServer: &messages.RSASignature{
					Signature: regHash,
				},
				ClientDHPubKey: dhPub,
				RequestSignature: &messages.RSASignature{
					Signature: clientSig,
				},
				Target: gwId.Marshal(),
			})
		if err != nil {
			errMsg := fmt.Sprintf("Register: Failed requesting nonce from gateway: %+v", err)
			return nil, errors.New(errMsg)
		}
		if nonceResponse.Error != "" {
			err := errors.New(fmt.Sprintf("requestNonce: nonceResponse error: %s", nonceResponse.Error))
			return nil, err
		}
		return nonceResponse, nil
	})
	if err != nil {
		return nil, nil, err
	}
	nonceResponse := result.(*pb.Nonce)

	// Use Client keypair to sign Server nonce
	return nonceResponse.Nonce, nonceResponse.DHPubKey, nil
}

// confirmNonce is a helper for the Register function
// It signs a nonce and sends it for confirmation
// Returns nil if successful, error otherwise
func confirmNonce(sender *gateway.Sender, comms RegisterNodeCommsInterface, UID, nonce []byte,
	privateKeyRSA *rsa.PrivateKey, gwID *id.ID) error {
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
	})
	return err
}
