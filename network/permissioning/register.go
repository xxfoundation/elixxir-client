package permissioning

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/comms/client"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
)

//RegisterWithPermissioning registers the user and returns the User ID.
// Returns an error if registration fails.
func RegisterWithPermissioning(ctx context.Context, comms client.Comms, registrationCode string) error {
	instance := ctx.Manager.GetInstance()
	instance.GetPartialNdf()

	//Check the regState is in proper state for registration
	regState := ctx.Session.GetRegistrationStatus()
	if regState != user.KeyGenComplete {
		return errors.Errorf("Attempting to register before key generation!")
	}

	userData := ctx.Session.User()

	// Register with the permissioning server and generate user information
	regValidationSignature, err := sendRegistrationMessage(comms,
		registrationCode,
		userData.GetCryptographicIdentity().GetRSA().GetPublic())
	if err != nil {
		globals.Log.INFO.Printf(err.Error())
		return err
	}

	// update the session with the registration response
	userData.SetRegistrationValidationSignature(regValidationSignature)

	err = ctx.Session.ForwardRegistrationStatus(user.PermissioningComplete)
	if err != nil {
		return err
	}

	return nil
}

// sendRegistrationMessage is a helper for the Register function
// It sends a registration message and returns the registration signature
// Registration code can also be an empty string
// precondition: client comms must have added permissioning host
func sendRegistrationMessage(comms client.Comms, registrationCode string,
	publicKeyRSA *rsa.PublicKey) ([]byte, error) {

	// Send registration code and public key to RegistrationServer
	host, ok := comms.GetHost(&id.Permissioning)
	if !ok {
		return nil, errors.New("Failed to find permissioning host")
	}

	response, err := comms.
		SendRegistrationMessage(host,
			&pb.UserRegistration{
				RegistrationCode: registrationCode,
				ClientRSAPubKey:  string(rsa.CreatePublicKeyPem(publicKeyRSA)),
			})
	if err != nil {
		err = errors.Wrap(err, "sendRegistrationMessage: Unable to contact Registration Server!")
		return nil, err
	}
	if response.Error != "" {
		return nil, errors.Wrapf(err, "sendRegistrationMessage: error handling message: %s", response.Error)
	}
	return response.ClientSignedByServer.Signature, nil
}
