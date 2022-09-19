package channels

import (
	"crypto/ed25519"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/channel"
	"io"
	"time"
)

// NewDummyNameService returns a dummy object adhering to the name service
// This neither produces valid signatures or validates passed signature
// is is for Development and Debugging purposes only
func NewDummyNameService(username string, rng io.Reader) (NameService, error) {
	jww.WARN.Printf("Creating a Dummy Name Service. This is for " +
		"development and debugging only. It does not produce valid " +
		"signatures or verify passed signatures. YOU SHOULD NEVER SEE THIS " +
		"MESSAGE IN PRODUCTION")

	dns := &dummyNameService{
		username: username,
		lease:    time.Now().Add(35 * 24 * time.Hour),
	}

	//generate the private key
	var err error
	dns.public, dns.private, err = ed25519.GenerateKey(rng)
	if err != nil {
		return nil, err
	}

	//generate a dummy user discover identity to produce a validation signature
	//just sign with our own key, it wont be evaluated anyhow
	dns.validationSig = channel.SignChannelLease(dns.public, dns.username,
		dns.lease, dns.private)

	return dns, nil
}

type dummyNameService struct {
	private       ed25519.PrivateKey
	public        ed25519.PublicKey
	username      string
	validationSig []byte
	lease         time.Time
}

func (dns *dummyNameService) GetUsername() string {
	return dns.username
}

func (dns *dummyNameService) GetChannelValidationSignature() ([]byte, time.Time) {
	jww.WARN.Printf("GetChannelValidationSignature called on Dummy Name " +
		"Service, dummy signature from a random key returned - identity not " +
		"proven. YOU SHOULD NEVER SEE THIS MESSAGE IN PRODUCTION")
	return dns.validationSig, dns.lease
}

func (dns *dummyNameService) GetChannelPubkey() ed25519.PublicKey {
	return dns.public
}

func (dns *dummyNameService) SignChannelMessage(message []byte) (signature []byte, err error) {
	jww.WARN.Printf("SignChannelMessage called on Dummy Name Service, " +
		"signature from a random key - identity not proven. YOU SHOULD " +
		"NEVER SEE THIS MESSAGE IN PRODUCTION")
	sig := ed25519.Sign(dns.private, message)
	return sig, nil
}

func (dns *dummyNameService) ValidateChannelMessage(username string, lease time.Time,
	pubKey ed25519.PublicKey, authorIDSignature []byte) bool {
	//ignore the authorIDSignature
	jww.WARN.Printf("ValidateChannelMessage called on Dummy Name Service, " +
		"no validation done - identity not validated. YOU SHOULD NEVER SEE " +
		"THIS MESSAGE IN PRODUCTION")
	return true
}
