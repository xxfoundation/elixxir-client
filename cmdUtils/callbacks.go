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

func MakeAuthCallbacks(autoConfirm bool, params xxdk.E2EParams) *AuthCallbacks {
	return &AuthCallbacks{
		autoConfirm: autoConfirm,
		confCh:      make(chan *id.ID, 10),
		params:      params,
	}
}

func (a *AuthCallbacks) ReceiveConfirmation() *id.ID {
	return <-a.confCh
}

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
		if viper.GetBool(VerifySendFlag) { // Verify message sends were successful
			acceptChannelVerified(messenger, requestor.ID, a.params)
		} else {
			acceptChannel(messenger, requestor.ID)
		}

		a.confCh <- requestor.ID
	}

}

func (a *AuthCallbacks) Confirm(requestor contact.Contact,
	_ receptionID.EphemeralIdentity,
	_ rounds.Round, _ *xxdk.E2e) {
	jww.INFO.Printf("Channel Confirmed: %s", requestor.ID)
	a.confCh <- requestor.ID
}

func (a *AuthCallbacks) Reset(requestor contact.Contact,
	_ receptionID.EphemeralIdentity,
	_ rounds.Round, _ *xxdk.E2e) {
	msg := fmt.Sprintf("Authentication channel reset from: %s\n",
		requestor.ID)
	jww.INFO.Printf(msg)
	fmt.Printf(msg)
}

func RegisterMessageListener(client *xxdk.E2e) chan receive.Message {
	recvCh := make(chan receive.Message, 10000)
	listenerID := client.GetE2E().RegisterChannel("DefaultCLIReceiver",
		receive.AnyUser(), catalog.NoType, recvCh)
	jww.INFO.Printf("Message ListenerID: %v", listenerID)
	return recvCh
}
