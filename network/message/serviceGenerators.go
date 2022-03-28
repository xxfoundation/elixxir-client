package message

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/sih"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
)

// GetDefaultService is used to generate a default service. All identities will
// respond to their default service, but it lacks privacy because it uses the
// public ID as the key. Used for initial reach out in some protocols, otherwise
// should not be used.
func GetDefaultService(recipient *id.ID) Service {
	jww.WARN.Printf(
		"Generating Default Service for %s - may not be private", recipient)
	return Service{
		Identifier: recipient[:],
		Tag:        sih.Default,
		Source:     recipient[:],
	}
}

// GetRandomService is used to make a service for cMix sending when no service
// is needed. It fills the Identifier with random, bits in order to preserve
// privacy.
func GetRandomService(rng csprng.Source) Service {
	identifier := make([]byte, 32)
	if _, err := rng.Read(identifier); err != nil {
		jww.FATAL.Panicf("Failed to generate random data: %+v", err)
	}
	return Service{
		Identifier: identifier,
		Tag:        "Random",
		Source:     identifier,
	}
}
