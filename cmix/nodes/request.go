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
	"gitlab.com/elixxir/client/v4/stoppable"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/registration"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/chacha"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

// requestKeys is a helper function which constructs a
// SignedClientBatchKeyRequest message.  This message is sent via the passed
// gateway Sender. It will further handle the request from the gateway.
// Responses are sent to a channel for processing by worker threads
func requestKeys(ngws []network.NodeGateway, dhPub *cyclic.Int, s session, r *registrar,
	stop *stoppable.Single) (*pb.SignedBatchKeyResponse, error) {
	rng := r.rng.GetStream()
	defer rng.Close()

	start := time.Now()

	var gwIds []*id.ID
	for _, ngw := range ngws {
		// Parse the ID into an id.ID object.
		gwId := ngw.Gateway.ID
		gatewayID, err := id.Unmarshal(gwId)
		if err != nil {
			jww.ERROR.Printf("registerWithNode failed to decode "+
				"gateway ID: %v", err)
			return nil, err
		}
		gwIds = append(gwIds, gatewayID)
	}

	signedBatchKeyReq, err := makeSignedKeyRequest(s, rng, gwIds, dhPub)
	if err != nil {
		return nil, err
	}

	// Request nonce message from gateway
	jww.DEBUG.Printf("Register: Requesting client key from "+
		"gateways %+v, setup took %s", gwIds, time.Since(start))

	start = time.Now()
	result, err := r.sender.SendToAny(func(host *connect.Host) (interface{}, error) {
		startInternal := time.Now()
		keyResponse, err2 := r.comms.BatchNodeRegistration(host, signedBatchKeyReq)
		if err2 != nil {
			return nil, errors.WithMessagef(err2,
				"Register: Failed requesting client key from gateways %+v", gwIds)
		}
		jww.TRACE.Printf("just comm reg request took %s", time.Since(startInternal))

		return keyResponse, nil
	}, stop)
	jww.TRACE.Printf("full reg request took %s", time.Since(start))
	if err != nil {
		return nil, err
	}

	// Cast the response
	signedKeyResponses := result.(*pb.SignedBatchKeyResponse)
	if len(ngws) != len(signedKeyResponses.SignedKeys) {
		return nil, errors.Errorf("Should have received %d slots, only received %d", len(ngws), len(signedKeyResponses.SignedKeys))
	}

	return signedKeyResponses, nil
}

// makeSignedKeyRequest is a helper function which constructs a
// pb.SignedClientBatchKeyRequest to send to the node/gateway pairs the
// user is trying to register with.
func makeSignedKeyRequest(s session, rng io.Reader,
	targets []*id.ID, dhPub *cyclic.Int) (*pb.SignedClientBatchKeyRequest, error) {

	// Reconstruct client confirmation message
	userPubKeyRSA := rsa.CreatePublicKeyPem(s.GetTransmissionRSA().GetPublic())
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
	opts := rsa.NewDefaultOptions()
	opts.Hash = hash.CMixHash
	h := opts.Hash.New()
	h.Write(serializedMessage)
	data := h.Sum(nil)
	clientSig, err := rsa.Sign(rng, s.GetTransmissionRSA(), opts.Hash,
		data, opts)
	if err != nil {
		return nil, err
	}

	var targetBytes [][]byte
	for _, gwId := range targets {
		targetBytes = append(targetBytes, gwId.Bytes())
	}

	// Construct signed key request message
	signedRequest := &pb.SignedClientBatchKeyRequest{
		ClientKeyRequest:          serializedMessage,
		ClientKeyRequestSignature: &messages.RSASignature{Signature: clientSig},
		Targets:                   targetBytes,
		Timeout:                   250,
	}

	return signedRequest, nil
}

// processRequestResponse is a helper function which handles the server's
// key request response.
func processRequestResponse(signedKeyResponse *pb.SignedKeyResponse,
	ngw network.NodeGateway, grp *cyclic.Group,
	dhPrivKey *cyclic.Int) (*cyclic.Int, []byte, uint64, error) {
	// Define hashing algorithm
	opts := rsa.NewDefaultOptions()
	opts.Hash = hash.CMixHash
	h := opts.Hash.New()

	// Hash the response
	h.Reset()
	h.Write(signedKeyResponse.GetKeyResponse())
	hashedResponse := h.Sum(nil)

	// Verify the response signature
	err := verifyNodeSignature(ngw.Gateway.TlsCertificate, opts.Hash, hashedResponse,
		signedKeyResponse.GetKeyResponseSignedByGateway().GetSignature(), opts)
	if err != nil {
		return nil, nil, 0,
			errors.Errorf("Could not verify nodes's signature: %v", err)
	}

	// Unmarshal the response
	keyResponse := &pb.ClientKeyResponse{}
	err = proto.Unmarshal(signedKeyResponse.GetKeyResponse(), keyResponse)
	if err != nil {
		return nil, nil, 0,
			errors.WithMessagef(err, "Failed to unmarshal client key response")
	}

	// Convert Node DH Public key to a cyclic.Int
	nodeDHPub := grp.NewIntFromBytes(keyResponse.GetNodeDHPubKey())

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
		keyResponse.GetEncryptedClientKey(), opts.Hash.New,
		keyResponse.GetEncryptedClientKeyHMAC()) {
		return nil, nil, 0, errors.New("Failed to verify client HMAC")
	}

	// Decrypt the client key
	clientKey, err := chacha.Decrypt(
		sessionKey.Bytes(), keyResponse.GetEncryptedClientKey())
	if err != nil {
		return nil, nil, 0,
			errors.WithMessagef(err, "Failed to decrypt client key")
	}

	// Construct the transmission key from the client key
	transmissionKey := grp.NewIntFromBytes(clientKey)

	// Use Cmix keypair to sign Server nonce
	return transmissionKey, keyResponse.GetKeyID(), keyResponse.GetValidUntil(), nil
}
