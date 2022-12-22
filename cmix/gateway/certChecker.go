////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package gateway

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

const (
	certCheckerPrefix     = "GwCertChecker"
	keyTemplate           = "GatewayCertificate-%s"
	certCheckerStorageVer = uint64(1)
)

// CertCheckerCommInterface is an interface for client comms to be used in cert checker
type CertCheckerCommInterface interface {
	GetGatewayTLSCertificate(host *connect.Host,
		message *pb.RequestGatewayCert) (*pb.GatewayCertificate, error)
}

// certChecker stores verified certificates and handles verification checking
type certChecker struct {
	kv    *versioned.KV
	comms CertCheckerCommInterface
}

// newCertChecker initializes a certChecker object
func newCertChecker(comms CertCheckerCommInterface, kv *versioned.KV) *certChecker {
	return &certChecker{
		kv:    kv.Prefix(certCheckerPrefix),
		comms: comms,
	}
}

// CheckRemoteCertificate attempts to verify the tls certificate for a given host
func (cc *certChecker) CheckRemoteCertificate(gwHost *connect.Host) error {
	if !gwHost.IsWeb() {
		jww.WARN.Printf("remote certificate verification is only " +
			"implemented for web connections")
		return nil
	}
	// Request signed certificate from the gateway
	gwTlsCertResp, err := cc.comms.GetGatewayTLSCertificate(gwHost, &pb.RequestGatewayCert{})
	if err != nil {
		return err
	}
	declaredRemoteCert := gwTlsCertResp.GetCertificate()
	remoteCertSignature := gwTlsCertResp.GetSignature()
	declaredFingerprint := md5.Sum(declaredRemoteCert)

	// Get remote certificate used for connection from the host object
	usedRemoteCert, err := gwHost.GetRemoteCertificate()
	if err != nil {
		return err
	}
	usedFingerprint := md5.Sum(usedRemoteCert.Raw)

	// If the fingerprints of the used & declared certs do not match, return an error
	if usedFingerprint != declaredFingerprint {
		return errors.Errorf("Declared & used remote certificates "+
			"do not match\n\tDeclared: %+v\n\tUsed: %+v\n",
			declaredFingerprint, usedFingerprint)
	}

	// Check if we have already verified this certificate for this host
	storedFingerprint, err := cc.loadGatewayCertificateFingerprint(gwHost)
	if err == nil {
		if bytes.Compare(storedFingerprint, usedFingerprint[:]) == 0 {
			return nil
		}
	}

	// Verify received signature
	err = verifyRemoteCertificate(usedRemoteCert.Raw, remoteCertSignature, gwHost)
	if err != nil {
		return err
	}

	// Store checked certificate fingerprint
	return cc.storeGatewayCertificateFingerprint(usedFingerprint[:], gwHost)
}

// verifyRemoteCertificate verifies the RSA signature of a gateway on its tls certificate
func verifyRemoteCertificate(cert, sig []byte, gwHost *connect.Host) error {
	h, err := hash.NewCMixHash()
	if err != nil {
		return err
	}
	h.Write(cert)
	return rsa.Verify(gwHost.GetPubKey(), hash.CMixHash, h.Sum(nil), sig, rsa.NewDefaultOptions())
}

// loadGatewayCertificateFingerprint retrieves the stored certificate
// fingerprint for a given gateway, or returns an error if not found
func (cc *certChecker) loadGatewayCertificateFingerprint(gwHost *connect.Host) ([]byte, error) {
	key := getKey(gwHost.GetId())
	obj, err := cc.kv.Get(key, certCheckerStorageVer)
	if err != nil {
		return nil, err
	}
	return obj.Data, err
}

// storeGatewayCertificateFingerprint stores the certificate fingerprint for a given gateway
func (cc *certChecker) storeGatewayCertificateFingerprint(fingerprint []byte, gwHost *connect.Host) error {
	key := getKey(gwHost.GetId())
	return cc.kv.Set(key, &versioned.Object{
		Version:   certCheckerStorageVer,
		Timestamp: time.Now(),
		Data:      fingerprint,
	})
}

// getKey is a helper function to generate the key for a gateway certificate fingerprint
func getKey(id *id.ID) string {
	return fmt.Sprintf(keyTemplate, id.String())
}
