////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package nodes

import (
	"io"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/gateway"
	"gitlab.com/elixxir/client/v4/stoppable"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/registration"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/chacha"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

// requestKey is a helper function which constructs a ClientKeyRequest message.
// This message is sent to the passed gateway. It will further handle the
// request from the gateway.
func requestKey(sender gateway.Sender, comms RegisterNodeCommsInterface,
	ngw network.NodeGateway, s session, r *registrar,
	rng io.Reader,
	stop *stoppable.Single) (*cyclic.Int, []byte, uint64, error) {

	// Generate a Diffie-Hellman keypair
	grp := r.session.GetCmixGroup()

	start := time.Now()
	prime := grp.GetPBytes()
	dhPrivBytes, err := csprng.GenerateInGroup(prime, 32, rng)
	if err != nil {
		return nil, nil, 0, err
	}
	dhPriv := grp.NewIntFromBytes(dhPrivBytes)
	dhPub := diffieHellman.GeneratePublicKey(dhPriv, grp)

	// Parse the ID into an id.ID object.
	gwId := ngw.Gateway.ID
	gatewayID, err := id.Unmarshal(gwId)
	if err != nil {
		jww.ERROR.Printf("registerWithNode failed to decode "+
			"gateway ID: %v", err)
		return nil, nil, 0, err
	}

	signedKeyReq, err := makeSignedKeyRequest(s, rng, gatewayID, dhPub)
	if err != nil {
		return nil, nil, 0, err
	}

	// Request nonce message from gateway
	jww.INFO.Printf("Register: Requesting client key from "+
		"gateway %s, setup took %s", gatewayID, time.Since(start))

	start = time.Now()
	result, err := sender.SendToAny(func(host *connect.Host) (interface{}, error) {
		startInternal := time.Now()
		keyResponse, err2 := comms.SendRequestClientKeyMessage(host, signedKeyReq)
		if err2 != nil {
			return nil, errors.WithMessagef(err2,
				"Register: Failed requesting client key from gateway %s", gatewayID.String())
		}
		if keyResponse.Error != "" {
			return nil, errors.WithMessage(err2,
				"requestKey: clientKeyResponse error")
		}
		jww.TRACE.Printf("just comm reg request took %s", time.Since(startInternal))

		return keyResponse, nil
	}, stop)
	jww.TRACE.Printf("full reg request took %s", time.Since(start))

	if err != nil {
		return nil, nil, 0, err
	}

	// Cast the response
	signedKeyResponse := result.(*pb.SignedKeyResponse)
	if signedKeyResponse.Error != "" {
		return nil, nil, 0, errors.New(signedKeyResponse.Error)
	}

	// Process the server's response
	return processRequestResponse(signedKeyResponse, ngw, grp, dhPriv)
}

// makeSignedKeyRequest is a helper function which constructs a
// pb.SignedClientKeyRequest to send to the node/gateway pair the
// user is trying to register with.
func makeSignedKeyRequest(s session, rng io.Reader,
	gwId *id.ID, dhPub *cyclic.Int) (*pb.SignedClientKeyRequest, error) {

	// Reconstruct client confirmation message
	userPubKeyRSA := s.GetTransmissionRSA().Public().MarshalPem()
	confirmation := &pb.ClientRegistrationConfirmation{
		RSAPubKey: string(userPubKeyRSA),
		Timestamp: s.GetRegistrationTimestamp().UnixNano(),
	}
	confirmationSerialized, err := proto.Marshal(confirmation)
	if err != nil {
		return nil, err
	}

	// Construct a key request message
	keyRequest := &pb.ClientKeyRequest{
		Salt: s.GetTransmissionSalt(),
		ClientTransmissionConfirmation: &pb.SignedRegistrationConfirmation{
			RegistrarSignature: &messages.RSASignature{
				Signature: s.GetTransmissionRegistrationValidationSignature()},
			ClientRegistrationConfirmation: confirmationSerialized,
		},
		ClientDHPubKey:        dhPub.Bytes(),
		RegistrationTimestamp: s.GetRegistrationTimestamp().UnixNano(),
		RequestTimestamp:      netTime.Now().UnixNano(),
	}

	// Serialize the reconstructed message
	serializedMessage, err := proto.Marshal(keyRequest)
	if err != nil {
		return nil, err
	}

	// Sign DH public key
	h := hash.CMixHash.New()
	h.Write(serializedMessage)
	data := h.Sum(nil)
	clientSig, err := signRegistrationRequest(rng, data, s.GetTransmissionRSA())
	if err != nil {
		return nil, err
	}

	// Construct signed key request message
	signedRequest := &pb.SignedClientKeyRequest{
		ClientKeyRequest:          serializedMessage,
		ClientKeyRequestSignature: &messages.RSASignature{Signature: clientSig},
		Target:                    gwId.Bytes(),
		UseSHA:                    useSHA(),
	}

	return signedRequest, nil
}

// processRequestResponse is a helper function which handles the server's
// key request response.
func processRequestResponse(signedKeyResponse *pb.SignedKeyResponse,
	ngw network.NodeGateway, grp *cyclic.Group,
	dhPrivKey *cyclic.Int) (*cyclic.Int, []byte, uint64, error) {
	h := hash.CMixHash.New()

	// Hash the response
	h.Reset()
	h.Write(signedKeyResponse.KeyResponse)
	hashedResponse := h.Sum(nil)

	// Verify the response signature
	err := verifyNodeSignature(ngw.Gateway.TlsCertificate, hashedResponse,
		signedKeyResponse.KeyResponseSignedByGateway.Signature)
	if err != nil {
		return nil, nil, 0,
			errors.Errorf("Could not verify nodes's signature: %v", err)
	}

	// Unmarshal the response
	keyResponse := &pb.ClientKeyResponse{}
	err = proto.Unmarshal(signedKeyResponse.KeyResponse, keyResponse)
	if err != nil {
		return nil, nil, 0,
			errors.WithMessagef(err, "Failed to unmarshal client key response")
	}

	// Convert Node DH Public key to a cyclic.Int
	nodeDHPub := grp.NewIntFromBytes(keyResponse.NodeDHPubKey)

	start := time.Now()
	// Construct the session key
	h.Reset()
	sessionKey := registration.GenerateBaseKey(grp,
		nodeDHPub, dhPrivKey, h)

	jww.TRACE.Printf("DH for reg took %s", time.Since(start))

	// Verify the HMAC
	jww.TRACE.Printf("[ClientKeyHMAC] Session Key Bytes: %+v", sessionKey.Bytes())
	jww.TRACE.Printf("[ClientKeyHMAC] EncryptedClientKey: %+v", keyResponse.EncryptedClientKey)
	jww.TRACE.Printf("[ClientKeyHMAC] EncryptedClientKeyHMAC: %+v", keyResponse.EncryptedClientKeyHMAC)

	if !registration.VerifyClientHMAC(sessionKey.Bytes(),
		keyResponse.EncryptedClientKey, hash.CMixHash.New,
		keyResponse.EncryptedClientKeyHMAC) {
		return nil, nil, 0, errors.New("Failed to verify client HMAC")
	}

	// Decrypt the client key
	clientKey, err := chacha.Decrypt(
		sessionKey.Bytes(), keyResponse.EncryptedClientKey)
	if err != nil {
		return nil, nil, 0,
			errors.WithMessagef(err, "Failed to decrypt client key")
	}

	// Construct the transmission key from the client key
	transmissionKey := grp.NewIntFromBytes(clientKey)

	// Use Cmix keypair to sign Server nonce
	return transmissionKey, keyResponse.KeyID, keyResponse.ValidUntil, nil
}
