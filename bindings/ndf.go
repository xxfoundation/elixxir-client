///////////////////////////////////////////////////////////////////////////////
// Copyright © 2021 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

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

// todo: populate with actual URL
// ndfUrl is a hardcoded url to a bucket containing the signed NDF message.
const ndfUrl = `elixxir.io`

// DownloadSignedNdf retrieves the NDF from a hardcoded bucket URL.
// The NDF returned requires further processing and verification
// before being used. Use ProcessSignedNdf to properly process
// the downloaded data returned.
// DO NOT USE THE RETURNED DATA TO START A CLIENT.
func DownloadSignedNdf() ([]byte, error) {
	// Build a request for the file
	resp, err := http.Get(ndfUrl)
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to retrieve "+
			"NDF from %s", ndfUrl)
	}
	defer resp.Body.Close()

	// Download contents of the file
	signedNdfEncoded, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to read signed "+
			"NDF response request")
	}

	return signedNdfEncoded, nil
}

// DownloadSignedNdfWithUrl retrieves the NDF from a specified URL.
// The NDF returned requires further processing and verification
// before being used. Use ProcessSignedNdf to properly process
// the downloaded data returned.
// DO NOT USE THE RETURNED DATA TO START A CLIENT.
func DownloadSignedNdfWithUrl(url string) ([]byte, error) {
	// Build a reqeust for the file
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

	return signedNdfEncoded, nil
}

// ProcessSignedNdf takes the downloaded NDF from either
// DownloadSignedNdf or DownloadSignedNdfWithUrl (signedNdfEncoded)
// and the scheduling certificate (cert). The downloaded NDF is parsed
// into a protobuf containing a signature. The signature is verified using the
// passed in cert. Upon successful parsing and verification, the NDF is
// returned as byte data, which may be used to start a client.
func ProcessSignedNdf(signedNdfEncoded []byte, cert string) ([]byte, error) {
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
