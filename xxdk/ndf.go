///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package xxdk

import (
	"encoding/base64"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/client"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/crypto/tls"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"io/ioutil"
	"net/http"
)

// DownloadNdfFromGateway will download an NDF from a gateway on the cMix network.
// It will take the given address and certificate and send a request to a gateway
// for an NDF over HTTP/2 using the xx network's gRPC implementation.
// This returns a JSON marshalled version of the NDF.
func DownloadNdfFromGateway(address string, cert []byte) (
	[]byte, error) {
	// Establish parameters for gRPC
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false

	// Construct client's gRPC comms object
	comms, err := client.NewClientComms(nil, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	// Construct a host off of the gateway to connect to
	host, err := connect.NewHost(&id.TempGateway, address,
		cert, params)
	if err != nil {
		return nil, err
	}

	// Construct a Poll message with dummy data.
	// All that's needed is the NDF
	dummyID := ephemeral.ReservedIDs[0]
	pollMsg := &pb.GatewayPoll{
		Partial: &pb.NDFHash{
			Hash: nil,
		},
		LastUpdate:    uint64(0),
		ReceptionID:   dummyID[:],
		ClientVersion: []byte(SEMVER),
	}

	// Send poll request and receive response containing NDF
	resp, err := comms.SendPoll(host, pollMsg)
	if err != nil {
		return nil, err
	}

	return resp.PartialNDF.Ndf, nil
}

// DownloadAndVerifySignedNdfWithUrl retrieves the NDF from a specified URL.
// The NDF is processed into a protobuf containing a signature that is verified
// using the cert string passed in. The NDF is returned as marshaled byte data
// that may be used to start a client.
func DownloadAndVerifySignedNdfWithUrl(url, cert string) ([]byte, error) {
	// Build a request for the file
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.WithMessagef(
			err, "Failed to retrieve NDF from %s", url)
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			jww.ERROR.Printf("Failed to close http response body: %+v", err)
		}
	}()

	// Download contents of the file
	signedNdfEncoded, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.WithMessage(
			err, "Failed to read signed NDF response request")
	}

	// Process the download NDF and return the marshaled NDF
	return processAndVerifySignedNdf(signedNdfEncoded, cert)
}

// processAndVerifySignedNdf is a helper function that parses the downloaded NDF
// into a protobuf containing a signature. The signature is verified using the
// passed in cert. Upon successful parsing and verification, the NDF is returned
// as byte data.
func processAndVerifySignedNdf(signedNdfEncoded []byte, cert string) ([]byte, error) {
	// Base64 decode the signed NDF
	signedNdfMarshaled, err := base64.StdEncoding.DecodeString(
		string(signedNdfEncoded))
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to decode signed NDF")
	}

	// Unmarshal the signed NDF
	signedNdfMsg := &pb.NDF{}
	err = proto.Unmarshal(signedNdfMarshaled, signedNdfMsg)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to unmarshal signed NDF into protobuf")
	}

	// Load the certificate from it's PEM contents
	schedulingCert, err := tls.LoadCertificate(cert)
	if err != nil {
		return nil, errors.WithMessagef(err,
			"Failed to parse scheduling cert (%s)", cert)
	}

	// Extract the public key from the cert
	schedulingPubKey, err := tls.ExtractPublicKey(schedulingCert)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to extract public key from cert")
	}

	// Verify signed NDF message
	err = signature.VerifyRsa(signedNdfMsg, schedulingPubKey)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to verify signed NDF message")
	}

	return signedNdfMsg.Ndf, nil
}
