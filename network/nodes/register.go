///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package nodes

import (
	"crypto/sha256"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage/user"
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
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/tls"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"gitlab.com/xx_network/primitives/netTime"
	"strconv"
	"sync"
	"time"
)

func registerNodes(r *registrar, stop *stoppable.Single, inProgress, attempts *sync.Map) {
	u := r.session.User()
	regSignature := u.GetTransmissionRegistrationValidationSignature()
	// Timestamp in which user has registered with registration
	regTimestamp := u.GetRegistrationTimestamp().UnixNano()
	uci := u.GetCryptographicIdentity()

	interval := time.Duration(500) * time.Millisecond
	t := time.NewTicker(interval)
	for {
		select {
		case <-stop.Quit():
			t.Stop()
			stop.ToStopped()
			return
		case gw := <-r.c:
			rng := r.rng.GetStream()
			nidStr := fmt.Sprintf("%x", gw.Node.ID)
			nid, err := gw.Node.GetNodeId()
			if err != nil {
				jww.WARN.Printf("Could not process node ID for registration: %s", err)
				continue
			}

			if r.Has(nid) {
				jww.INFO.Printf("not registering node %s, already registered", nid)
			}

			if _, operating := inProgress.LoadOrStore(nidStr, struct{}{}); operating {
				continue
			}

			//keep track of how many times this has been attempted
			numAttempts := uint(1)
			if nunAttemptsInterface, hasValue := attempts.LoadOrStore(nidStr, numAttempts); hasValue {
				numAttempts = nunAttemptsInterface.(uint)
				attempts.Store(nidStr, numAttempts+1)
			}

			// No need to register with stale nodes
			if isStale := gw.Node.Status == ndf.Stale; isStale {
				jww.DEBUG.Printf("Skipping registration with stale nodes %s", nidStr)
				continue
			}
			err = registerWithNode(r.sender, r.comms, gw, regSignature,
				regTimestamp, uci, r, rng, stop)
			inProgress.Delete(nidStr)
			if err != nil {
				jww.ERROR.Printf("Failed to register nodes: %+v", err)
				//if we have not reached the attempt limit for this gateway, send it back into the channel to retry
				if numAttempts < maxAttempts {
					go func() {
						//delay the send for a backoff
						time.Sleep(delayTable[numAttempts-1])
						r.c <- gw
					}()
				}
			}
			rng.Close()
		case <-t.C:
		}
	}
}

//registerWithNode serves as a helper for RegisterWithNodes
// It registers a user with a specific in the client's ndf.
func registerWithNode(sender *gateway.Sender, comms RegisterNodeCommsInterface,
	ngw network.NodeGateway, regSig []byte, registrationTimestampNano int64,
	uci *user.CryptographicIdentity, r *registrar, rng csprng.Source,
	stop *stoppable.Single) error {

	nodeID, err := ngw.Node.GetNodeId()
	if err != nil {
		jww.ERROR.Println("registerWithNode() failed to decode nodeId")
		return err
	}

	if r.Has(nodeID) {
		return nil
	}

	jww.INFO.Printf("registerWithNode() begin registration with nodes: %s", nodeID)

	var transmissionKey *cyclic.Int
	var validUntil uint64
	var keyId []byte
	// TODO: should move this to a precanned user initialization
	if uci.IsPrecanned() {
		userNum := int(uci.GetTransmissionID().Bytes()[7])
		h := sha256.New()
		h.Reset()
		h.Write([]byte(strconv.Itoa(4000 + userNum)))

		transmissionKey = r.session.GetCmixGroup().NewIntFromBytes(h.Sum(nil))
		jww.INFO.Printf("transmissionKey: %v", transmissionKey.Bytes())
	} else {
		// Request key from server
		transmissionKey, keyId, validUntil, err = requestKey(sender, comms, ngw, regSig,
			registrationTimestampNano, uci, r, rng, stop)

		if err != nil {
			return errors.Errorf("Failed to request key: %+v", err)
		}

	}

	r.add(nodeID, transmissionKey, validUntil, keyId)

	jww.INFO.Printf("Completed registration with nodes %s", nodeID)

	return nil
}

func requestKey(sender *gateway.Sender, comms RegisterNodeCommsInterface,
	ngw network.NodeGateway, regSig []byte, registrationTimestampNano int64,
	uci *user.CryptographicIdentity, r *registrar, rng csprng.Source,
	stop *stoppable.Single) (*cyclic.Int, []byte, uint64, error) {

	grp := r.session.GetCmixGroup()

	// FIXME: Why 256 bits? -- this is spec but not explained, it has
	// to do with optimizing operations on one side and still preserves
	// decent security -- cite this.
	dhPrivBytes, err := csprng.GenerateInGroup(grp.GetPBytes(), 256, rng)
	if err != nil {
		return nil, nil, 0, err
	}

	dhPriv := grp.NewIntFromBytes(dhPrivBytes)

	dhPub := diffieHellman.GeneratePublicKey(dhPriv, grp)

	// Reconstruct client confirmation message
	userPubKeyRSA := rsa.CreatePublicKeyPem(uci.GetTransmissionRSA().GetPublic())
	confirmation := &pb.ClientRegistrationConfirmation{RSAPubKey: string(userPubKeyRSA), Timestamp: registrationTimestampNano}
	confirmationSerialized, err := proto.Marshal(confirmation)
	if err != nil {
		return nil, nil, 0, err
	}

	keyRequest := &pb.ClientKeyRequest{
		Salt: uci.GetTransmissionSalt(),
		ClientTransmissionConfirmation: &pb.SignedRegistrationConfirmation{
			RegistrarSignature:             &messages.RSASignature{Signature: regSig},
			ClientRegistrationConfirmation: confirmationSerialized,
		},
		ClientDHPubKey:        dhPub.Bytes(),
		RegistrationTimestamp: registrationTimestampNano,
		RequestTimestamp:      netTime.Now().UnixNano(),
	}

	serializedMessage, err := proto.Marshal(keyRequest)
	if err != nil {
		return nil, nil, 0, err
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
		return nil, nil, 0, err
	}

	gwid := ngw.Gateway.ID
	gatewayID, err := id.Unmarshal(gwid)
	if err != nil {
		jww.ERROR.Println("registerWithNode() failed to decode gatewayID")
		return nil, nil, 0, err
	}

	// Request nonce message from gateway
	jww.INFO.Printf("Register: Requesting client key from gateway %v", gatewayID.String())

	result, err := sender.SendToAny(func(host *connect.Host) (interface{}, error) {
		keyResponse, err := comms.SendRequestClientKeyMessage(host,
			&pb.SignedClientKeyRequest{
				ClientKeyRequest:          serializedMessage,
				ClientKeyRequestSignature: &messages.RSASignature{Signature: clientSig},
				Target:                    gatewayID.Bytes(),
			})
		if err != nil {
			return nil, errors.WithMessage(err, "Register: Failed requesting client key from gateway")
		}
		if keyResponse.Error != "" {
			return nil, errors.WithMessage(err, "requestKey: clientKeyResponse error")
		}
		return keyResponse, nil
	}, stop)

	if err != nil {
		return nil, nil, 0, err
	}

	signedKeyResponse := result.(*pb.SignedKeyResponse)
	if signedKeyResponse.Error != "" {
		return nil, nil, 0, errors.New(signedKeyResponse.Error)
	}

	// Hash the response
	h.Reset()
	h.Write(signedKeyResponse.KeyResponse)
	hashedResponse := h.Sum(nil)

	// Load nodes certificate
	gatewayCert, err := tls.LoadCertificate(ngw.Gateway.TlsCertificate)
	if err != nil {
		return nil, nil, 0, errors.WithMessagef(err, "Unable to load nodes's certificate")
	}

	// Extract public key
	nodePubKey, err := tls.ExtractPublicKey(gatewayCert)
	if err != nil {
		return nil, nil, 0, errors.WithMessagef(err, "Unable to load nodes's public key")
	}

	// Verify the response signature
	err = rsa.Verify(nodePubKey, opts.Hash, hashedResponse,
		signedKeyResponse.KeyResponseSignedByGateway.Signature, opts)
	if err != nil {
		return nil, nil, 0, errors.WithMessagef(err, "Could not verify nodes's signature")
	}

	// Unmarshal the response
	keyResponse := &pb.ClientKeyResponse{}
	err = proto.Unmarshal(signedKeyResponse.KeyResponse, keyResponse)
	if err != nil {
		return nil, nil, 0, errors.WithMessagef(err, "Failed to unmarshal client key response")
	}

	h.Reset()

	// Convert Node DH Public key to a cyclic.Int
	nodeDHPub := grp.NewIntFromBytes(keyResponse.NodeDHPubKey)

	// Construct the session key
	sessionKey := registration.GenerateBaseKey(grp,
		nodeDHPub, dhPriv, h)

	// Verify the HMAC
	h.Reset()
	if !registration.VerifyClientHMAC(sessionKey.Bytes(), keyResponse.EncryptedClientKey,
		opts.Hash.New, keyResponse.EncryptedClientKeyHMAC) {
		return nil, nil, 0, errors.New("Failed to verify client HMAC")
	}

	// Decrypt the client key
	clientKey, err := chacha.Decrypt(sessionKey.Bytes(), keyResponse.EncryptedClientKey)
	if err != nil {
		return nil, nil, 0, errors.WithMessagef(err, "Failed to decrypt client key")
	}

	// Construct the transmission key from the client key
	transmissionKey := grp.NewIntFromBytes(clientKey)

	// Use Client keypair to sign Server nonce
	return transmissionKey, keyResponse.KeyID, keyResponse.ValidUntil, nil
}
