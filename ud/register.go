package ud

import (
	"fmt"
	"github.com/pkg/errors"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/factID"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
)

// register initiates registration with user discovery given a specified
// username. Provided a comms sub-interface to facilitate testing.
func (m *Manager) register(username string, networkSignature []byte,
	rng csprng.Source, comm registerUserComms) error {

	// Retrieve data used for registration
	identity := m.user.GetReceptionIdentity()
	privKey, err := identity.GetRSAPrivatePem()
	if err != nil {
		return err
	}
	grp, err := identity.GetGroup()
	if err != nil {
		return err
	}
	dhKeyPriv, err := identity.GetDHKeyPrivate()
	if err != nil {
		return err
	}
	dhKeyPub := diffieHellman.GeneratePublicKey(dhKeyPriv, grp)

	// Construct the user registration message
	msg := &pb.UDBUserRegistration{
		PermissioningSignature: networkSignature,
		RSAPublicPem:           string(rsa.CreatePublicKeyPem(privKey.GetPublic())),
		IdentityRegistration: &pb.Identity{
			Username: username,
			DhPubKey: dhKeyPub.Bytes(),
			Salt:     identity.Salt,
		},
		UID:       identity.ID.Marshal(),
		Timestamp: m.user.GetTransmissionIdentity().RegistrationTimestamp,
	}

	// Sign the identity data and add to user registration message
	identityDigest := msg.IdentityRegistration.Digest()
	msg.IdentitySignature, err = rsa.Sign(rng, privKey,
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
	signedFact, err := rsa.Sign(rng, privKey, hash.CMixHash, hashedFact, nil)
	if err != nil {
		return errors.Errorf("Failed to sign fact: %v", err)
	}

	// Add username fact register request to the user registration message
	msg.Frs = &pb.FactRegisterRequest{
		UID: identity.ID.Marshal(),
		Fact: &pb.Fact{
			Fact:     username,
			FactType: 0,
		},
		FactSig: signedFact,
	}

	// Register user with user discovery
	_, err = comm.SendRegisterUser(m.ud.host, msg)
	if err != nil {
		return err
	}

	// Set storage to registered
	if err = setRegistered(m.getKv()); err != nil && m.getEventReporter() != nil {
		m.getEventReporter().Report(1, "UserDiscovery", "Registration",
			fmt.Sprintf("User Registered with UD: %+v",
				username))
	}

	return err
}
