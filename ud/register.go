package ud

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/factID"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/signature/rsa"
)

type registerUserComms interface {
	SendRegisterUser(*connect.Host, *pb.UDBUserRegistration) (*messages.Ack, error)
}

// Register registers a user with user discovery. Will return an error if the
// network signatures are malformed or if the username is taken. Usernames cannot
// be changed after registration at this time. Will fail if the user is already
// registered.
// Identity does not go over cmix, it occurs over normal communications
func (m *Manager) Register(username string) error {
	jww.INFO.Printf("ud.Register(%s)", username)
	return m.register(username, m.comms)
}

// register registers a user with user discovery with a specified comm for
// easier testing.
func (m *Manager) register(username string, comm registerUserComms) error {
	if m.IsRegistered() {
		return errors.New("cannot register client with User Discovery: " +
			"client is already registered")
	}

	var err error
	user := m.storage.User()
	cryptoUser := m.storage.User().GetCryptographicIdentity()
	rng := m.rng.GetStream()

	// Construct the user registration message
	msg := &pb.UDBUserRegistration{
		PermissioningSignature: user.GetReceptionRegistrationValidationSignature(),
		RSAPublicPem:           string(rsa.CreatePublicKeyPem(cryptoUser.GetReceptionRSA().GetPublic())),
		IdentityRegistration: &pb.Identity{
			Username: username,
			DhPubKey: m.storage.E2e().GetDHPublicKey().Bytes(),
			Salt:     cryptoUser.GetReceptionSalt(),
		},
		UID:       cryptoUser.GetReceptionID().Marshal(),
		Timestamp: user.GetRegistrationTimestamp().UnixNano(),
	}

	// Sign the identity data and add to user registration message
	identityDigest := msg.IdentityRegistration.Digest()
	msg.IdentitySignature, err = rsa.Sign(rng, cryptoUser.GetReceptionRSA(),
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
	signedFact, err := rsa.Sign(rng, cryptoUser.GetReceptionRSA(), hash.CMixHash, hashedFact, nil)

	// Add username fact register request to the user registration message
	msg.Frs = &pb.FactRegisterRequest{
		UID: cryptoUser.GetReceptionID().Marshal(),
		Fact: &pb.Fact{
			Fact:     username,
			FactType: 0,
		},
		FactSig: signedFact,
	}

	// Get UD host
	host, err := m.getHost()
	if err != nil {
		return err
	}

	// Register user with user discovery
	_, err = comm.SendRegisterUser(host, msg)
	if err != nil {
		return err
	}

	err = m.setRegistered()
	if m.client != nil {
		m.client.ReportEvent(1, "UserDiscovery", "Registration",
			fmt.Sprintf("User Registered with UD: %+v",
				user))
	}

	// Store username
	err = m.storage.GetUd().StoreUsername(usernameFact)
	if err != nil {
		return err
	}

	return err
}
