package permissioning

import (
	"github.com/pkg/errors"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
)

// client.Comms should implement this interface
type RegistrationMessageSender interface {
	SendRegistrationMessage(host *connect.Host, message *pb.UserRegistration) (*pb.UserRegistrationConfirmation, error)
	GetHost(*id.ID) (*connect.Host, bool)
}

//Register registers the user with optional registration code
// Returns an error if registration fails.
func Register(comms RegistrationMessageSender, publicKey *rsa.PublicKey, registrationCode string) ([]byte, error) {
	// Send registration code and public key to RegistrationServer
	host, ok := comms.GetHost(&id.Permissioning)
	if !ok {
		return nil, errors.New("Failed to find permissioning host")
	}

	response, err := comms.
		SendRegistrationMessage(host,
			&pb.UserRegistration{
				RegistrationCode: registrationCode,
				ClientRSAPubKey:  string(rsa.CreatePublicKeyPem(publicKey)),
			})
	if err != nil {
		err = errors.Wrap(err, "sendRegistrationMessage: Unable to contact Registration Server!")
		return nil, err
	}
	if response.Error != "" {
		return nil, errors.Errorf("sendRegistrationMessage: error handling message: %s", response.Error)
	}
	return response.ClientSignedByServer.Signature, nil
}
