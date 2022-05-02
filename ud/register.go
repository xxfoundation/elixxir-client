package ud

import (
	"github.com/pkg/errors"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/factID"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
)

// register initiates registration with user discovery given a specified
// username. Provided a comms sub-interface to facilitate testing.
func (m *Manager) register(username string, rng csprng.Source,
	comm registerUserComms, udHost *connect.Host) error {

	var err error
	cryptoUser := m.user.PortableUserInfo()

	// Construct the user registration message
	msg := &pb.UDBUserRegistration{
		PermissioningSignature: m.user.GetReceptionRegistrationValidationSignature(),
		RSAPublicPem:           string(rsa.CreatePublicKeyPem(cryptoUser.ReceptionRSA.GetPublic())),
		IdentityRegistration: &pb.Identity{
			Username: username,
			DhPubKey: m.e2e.GetHistoricalDHPubkey().Bytes(),
			Salt:     cryptoUser.ReceptionSalt,
		},
		UID:       cryptoUser.ReceptionID.Marshal(),
		Timestamp: cryptoUser.RegistrationTimestamp,
	}

	// Sign the identity data and add to user registration message
	identityDigest := msg.IdentityRegistration.Digest()
	msg.IdentitySignature, err = rsa.Sign(rng, cryptoUser.ReceptionRSA,
		hash.CMixHash, identityDigest, nil)
	if err != nil {
		return errors.Errorf("Failed to sign user's IdentityRegistration: %+v", err)
	}

	// Create new username fact
	usernameFact, err := fact.NewFact(fact.Username, username)
	if err != nil {
		return errors.Errorf("Failed to create new username fact: %+v", err)
	}

	// Hash and sign fact
	hashedFact := factID.Fingerprint(usernameFact)
	signedFact, err := rsa.Sign(rng, cryptoUser.ReceptionRSA, hash.CMixHash, hashedFact, nil)

	// Add username fact register request to the user registration message
	msg.Frs = &pb.FactRegisterRequest{
		UID: cryptoUser.ReceptionID.Marshal(),
		Fact: &pb.Fact{
			Fact:     username,
			FactType: 0,
		},
		FactSig: signedFact,
	}

	// Register user with user discovery
	_, err = comm.SendRegisterUser(udHost, msg)
	return err
}
