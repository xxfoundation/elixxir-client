package noobChannel

import (
	"errors"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/catalog"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/e2e"
	"gitlab.com/elixxir/client/v4/e2e/receive"
	"gitlab.com/elixxir/client/v4/single"
	"gitlab.com/elixxir/client/v4/xxdk"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/crypto/contact"
	"time"
)

type Manager interface {
	GetNoobChannel() (*cryptoBroadcast.Channel, error)
}

type manager struct {
	confirmAuthCh      chan struct{}
	e2eClient          *xxdk.E2e
	noobChannelContact contact.Contact
	noobChannelChannel chan []byte
	listenerId         receive.ListenerID
}

func Init(e2eClient *xxdk.E2e, noobChannelContact contact.Contact) (Manager, error) {
	// Create manager struct
	m := &manager{
		confirmAuthCh:      make(chan struct{}),
		noobChannelChannel: make(chan []byte),
		e2eClient:          e2eClient,
		noobChannelContact: noobChannelContact,
	}

	// Register listener and auth callbacks for noob channel contact
	m.e2eClient.GetAuth().AddPartnerCallback(m.noobChannelContact.ID, m)
	m.listenerId = m.e2eClient.GetE2E().RegisterListener(m.noobChannelContact.ID, catalog.XxMessage, m)

	// Add noob channel auth
	_, err := m.e2eClient.GetAuth().Request(m.noobChannelContact, m.e2eClient.GetReceptionIdentity().GetContact().Facts)
	if err != nil {
		return nil, err
	}

	// Wait for auth confirm from noob channel
	timeout := time.NewTimer(30 * time.Second)
	select {
	case <-m.confirmAuthCh:
	case <-timeout.C:
		return nil, errors.New("timed out")
	}
	if !m.e2eClient.GetE2E().HasAuthenticatedChannel(m.noobChannelContact.ID) {
		return nil, errors.New("no authenticated channel")
	}
	return m, nil
}

func (m *manager) GetNoobChannel() (*cryptoBroadcast.Channel, error) {
	// Send E2E message to the noob channel bot
	_, err := m.e2eClient.GetE2E().SendE2E(catalog.XxMessage, m.noobChannelContact.ID, []byte("payload"), e2e.GetDefaultParams())
	if err != nil {
		return nil, err
	}

	// Wait for response
	timeout := time.NewTimer(45 * time.Second)
	var newChannel *cryptoBroadcast.Channel
	select {
	case newChannelBytes := <-m.noobChannelChannel:
		newChannel, err = cryptoBroadcast.UnmarshalChannel(newChannelBytes)
		if err != nil {
			return nil, err
		}
	case <-timeout.C:
		return nil, errors.New("timed out")
	}

	return newChannel, nil
}

/* NoobChannel-specific auth callbacks */

func (m *manager) Request(partner contact.Contact, receptionID receptionID.EphemeralIdentity,
	round rounds.Round) {
	jww.DEBUG.Printf("NoobChannel manager received an auth request")
}
func (m *manager) Confirm(partner contact.Contact, receptionID receptionID.EphemeralIdentity,
	round rounds.Round) {
	jww.DEBUG.Printf("NoobChannel manager received an auth confirmation")
	m.confirmAuthCh <- struct{}{}
}
func (m *manager) Reset(partner contact.Contact, receptionID receptionID.EphemeralIdentity,
	round rounds.Round) {
	jww.DEBUG.Printf("NoobChannel manager received an auth reset request")
}

/* NoobChannel-specific listener callbacks */

func (m *manager) Hear(item receive.Message) {
	jww.DEBUG.Printf("NoobChannel manager received a message")
	fmt.Println("Heard a message")
	m.noobChannelChannel <- item.Payload
}
func (m *manager) Name() string {
	return "noob channel listener"
}

func (m *manager) Callback(payload []byte, receptionID receptionID.EphemeralIdentity,
	rounds []rounds.Round, err error) {
	jww.DEBUG.Printf("NoobChannel manager received a message")
	fmt.Println("Heard a message")
	m.noobChannelChannel <- payload
}

func GetNoobChannel(e2eClient *xxdk.E2e, noobChannelContact contact.Contact) (*cryptoBroadcast.Channel, error) {
	m := &manager{
		noobChannelChannel: make(chan []byte),
	}

	_, _, err := single.TransmitRequest(noobChannelContact, "noobChannel", []byte("hello"), m, single.GetDefaultRequestParams(), e2eClient.GetCmix(), e2eClient.GetRng().GetStream(), e2eClient.GetStorage().GetE2EGroup())
	if err != nil {
		return nil, err
	}
	// Wait for response
	timeout := time.NewTimer(45 * time.Second)
	var newChannel *cryptoBroadcast.Channel
	select {
	case newChannelBytes := <-m.noobChannelChannel:
		newChannel, err = cryptoBroadcast.UnmarshalChannel(newChannelBytes)
		if err != nil {
			return nil, err
		}
	case <-timeout.C:
		return nil, errors.New("timed out")
	}

	return newChannel, nil
}
