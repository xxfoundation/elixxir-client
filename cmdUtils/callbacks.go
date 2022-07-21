package cmd

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/primitives/id"
)

// AuthCallbacks implements the auth.Callbacks interface.
type AuthCallbacks struct {
	autoConfirm bool
	confCh      chan *id.ID
	params      xxdk.E2EParams
}

// MakeAuthCallbacks is a constructor for AuthCallbacks/
func MakeAuthCallbacks(autoConfirm bool, params xxdk.E2EParams) *AuthCallbacks {
	return &AuthCallbacks{
		autoConfirm: autoConfirm,
		confCh:      make(chan *id.ID, 10),
		params:      params,
	}
}

// GetConfirmationChan is a getter of the private AuthCallbacks.confChan.
func (a *AuthCallbacks) GetConfirmationChan() chan *id.ID {
	return a.confCh
}

// Request will be called when a catalog.Request message is processed.
// This is an example implementation for CLI and CI/CD integration purposes.
// This will send a confirmation message back to the original sender.
func (a *AuthCallbacks) Request(requestor contact.Contact,
	receptionID receptionID.EphemeralIdentity,
	round rounds.Round, messenger *xxdk.E2e) {
	msg := fmt.Sprintf("Authentication channel request from: %s\n",
		requestor.ID)
	jww.INFO.Printf(msg)
	fmt.Printf(msg)
	if a.autoConfirm {
		jww.INFO.Printf("Channel Request: %s",
			requestor.ID)
		for {
			recipientContact, err := messenger.GetAuth().GetReceivedRequest(
				requestor.ID)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
			rid, err := messenger.GetAuth().Confirm(
				recipientContact)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}

			// Verify message sends were successful
			if viper.GetBool(VerifySendFlag) {
				if !VerifySendSuccess(messenger, a.params.Base, []id.Round{rid},
					requestor.ID, nil) {
					continue
				}
			}
			break
		}
		a.confCh <- requestor.ID
	}

}

// Confirm will be called when a catalog.Confirm message is processed.
// This is an example implementation for CLI and CI/CD integration purposes.
// This will send a confirmation signal through the AuthCallbacks
// confirmation channel.
func (a *AuthCallbacks) Confirm(requestor contact.Contact,
	_ receptionID.EphemeralIdentity,
	_ rounds.Round, _ *xxdk.E2e) {
	jww.INFO.Printf("Channel Confirmed: %s", requestor.ID)
	a.confCh <- requestor.ID
}

// Reset will be called when a catalog.Reset message is processed.
// This is an example implementation for CLI and CI/CD integration purposes.
// This will simply print a message that it received a message from the requestor
// to a log and stdout.
func (a *AuthCallbacks) Reset(requestor contact.Contact,
	_ receptionID.EphemeralIdentity,
	_ rounds.Round, _ *xxdk.E2e) {
	msg := fmt.Sprintf("Authentication channel reset from: %s\n",
		requestor.ID)
	jww.INFO.Printf(msg)
	fmt.Printf(msg)
}

// RegisterMessageListener registers a simple message listener to the messenger.
func RegisterMessageListener(messenger *xxdk.E2e) chan receive.Message {
	recvCh := make(chan receive.Message, 10000)
	listenerID := messenger.GetE2E().RegisterChannel("DefaultCLIReceiver",
		receive.AnyUser(), catalog.NoType, recvCh)
	jww.INFO.Printf("Message ListenerID: %v", listenerID)
	return recvCh
}
