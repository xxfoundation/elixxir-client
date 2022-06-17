///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package xxdk

import (
	"encoding/base64"
	"github.com/pkg/errors"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/crypto/tls"
	"google.golang.org/protobuf/proto"
	"io/ioutil"
	"net/http"
)

// DownloadAndVerifySignedNdfWithUrl retrieves the NDF from a specified URL.
// The NDF is processed into a protobuf containing a signature which
// is verified using the cert string passed in. The NDF is returned as marshaled
// byte data which may be used to start a client.
func DownloadAndVerifySignedNdfWithUrl(url, cert string) ([]byte, error) {
	// Build a request for the file
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to retrieve "+
			"NDF from %s", url)
	}
	defer resp.Body.Close()

	// Download contents of the file
	signedNdfEncoded, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to read signed "+
			"NDF response request")
	}

	// Process the download NDF and return the marshaled NDF
	return processAndVerifySignedNdf(signedNdfEncoded, cert)
}

// processAndVerifySignedNdf is a helper function which parses the downloaded NDF
// into a protobuf containing a signature. The signature is verified using the
// passed in cert. Upon successful parsing and verification, the NDF is
// returned as byte data.
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
		return nil, errors.WithMessage(err, "Failed to unmarshal "+
			"signed NDF into protobuf")
	}

	// Load the certificate from it's PEM contents
	schedulingCert, err := tls.LoadCertificate(cert)
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to parse scheduling cert (%s)", cert)
	}

	// Extract the public key from the cert
	schedulingPubKey, err := tls.ExtractPublicKey(schedulingCert)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to extract public key from cert")
	}

	// Verify signed NDF message
	err = signature.VerifyRsa(signedNdfMsg, schedulingPubKey)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to verify signed NDF message")
	}

	return signedNdfMsg.Ndf, nil
}
