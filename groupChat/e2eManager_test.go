package groupChat

import (
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix/message"
	clientE2E "gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	sessionImport "gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"testing"
	"time"
)

// testE2eManager is a test implementation of NetworkManager interface.
type testE2eManager struct {
	e2eMessages []testE2eMessage
	partners    map[id.ID]partner.Manager
	errSkip     int
	sendErr     int
	dhPubKey    *cyclic.Int
	grp         *cyclic.Group
	sync.RWMutex
}

type testE2eMessage struct {
	Recipient *id.ID
	Payload   []byte
}

func (tnm *testE2eManager) AddPartner(partnerID *id.ID, partnerPubKey,
	myPrivKey *cyclic.Int, _ *sidh.PublicKey, _ *sidh.PrivateKey,
	_, _ sessionImport.Params) (partner.Manager, error) {

	testPartner := partner.NewTestManager(partnerID, partnerPubKey, myPrivKey, &testing.T{})
	tnm.partners[*partnerID] = testPartner
	return testPartner, nil
}

func (tnm *testE2eManager) GetPartner(partnerID *id.ID) (partner.Manager, error) {
	if p, ok := tnm.partners[*partnerID]; ok {
		return p, nil
	}
	return nil, errors.New("Unable to find partner")
}

func (tnm *testE2eManager) GetHistoricalDHPubkey() *cyclic.Int {
	return tnm.dhPubKey
}

func (tnm *testE2eManager) GetHistoricalDHPrivkey() *cyclic.Int {
	return tnm.dhPubKey
}

func (tnm *testE2eManager) GetE2eMsg(i int) testE2eMessage {
	tnm.RLock()
	defer tnm.RUnlock()
	return tnm.e2eMessages[i]
}

func (tnm *testE2eManager) SendE2E(_ catalog.MessageType, recipient *id.ID,
	payload []byte, _ clientE2E.Params) (clientE2E.SendReport, error) {
	tnm.Lock()
	defer tnm.Unlock()

	tnm.errSkip++
	if tnm.sendErr == 1 {
		return clientE2E.SendReport{}, errors.New("SendE2E error")
	} else if tnm.sendErr == 2 && tnm.errSkip%2 == 0 {
		return clientE2E.SendReport{}, errors.New("SendE2E error")
	}

	tnm.e2eMessages = append(tnm.e2eMessages, testE2eMessage{
		Recipient: recipient,
		Payload:   payload,
	})

	return clientE2E.SendReport{RoundList: []id.Round{0, 1, 2, 3}}, nil
}

func (*testE2eManager) RegisterListener(*id.ID, catalog.MessageType, receive.Listener) receive.ListenerID {
	return receive.ListenerID{}
}

func (*testE2eManager) AddService(string, message.Processor) error {
	return nil
}

/////////////////////////////////////////////////////////////////////////////////////
// Unused & unimplemented methods of the test object ////////////////////////////////
/////////////////////////////////////////////////////////////////////////////////////

func (tnm *testE2eManager) DeletePartner(partnerId *id.ID) error {
	// TODO implement me
	panic("implement me")
}

func (tnm *testE2eManager) DeletePartnerNotify(partnerId *id.ID, params clientE2E.Params) error {
	// TODO implement me
	panic("implement me")
}

func (*testE2eManager) GetDefaultHistoricalDHPubkey() *cyclic.Int {
	panic("implement me")
}

func (*testE2eManager) GetDefaultHistoricalDHPrivkey() *cyclic.Int {
	panic("implement me")
}

func (tnm *testE2eManager) StartProcesses() (stoppable.Stoppable, error) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testE2eManager) RegisterFunc(name string, senderID *id.ID, messageType catalog.MessageType, newListener receive.ListenerFunc) receive.ListenerID {
	//TODO implement me
	panic("implement me")
}

func (tnm *testE2eManager) RegisterChannel(name string, senderID *id.ID, messageType catalog.MessageType, newListener chan receive.Message) receive.ListenerID {
	//TODO implement me
	panic("implement me")
}

func (tnm *testE2eManager) Unregister(listenerID receive.ListenerID) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testE2eManager) UnregisterUserListeners(userID *id.ID) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testE2eManager) RegisterCallbacks(callbacks clientE2E.Callbacks) {
	// TODO implement me
	panic("implement me")
}

func (tnm *testE2eManager) AddPartnerCallbacks(partnerID *id.ID, cb clientE2E.Callbacks) {
	// TODO implement me
	panic("implement me")
}

func (tnm *testE2eManager) DeletePartnerCallbacks(partnerID *id.ID) {
	// TODO implement me
	panic("implement me")
}

func (tnm *testE2eManager) GetAllPartnerIDs() []*id.ID {
	//TODO implement me
	panic("implement me")
}

func (tnm *testE2eManager) HasAuthenticatedChannel(partner *id.ID) bool {
	//TODO implement me
	panic("implement me")
}

func (tnm *testE2eManager) RemoveService(tag string) error {
	//TODO implement me
	panic("implement me")
}

func (tnm *testE2eManager) SendUnsafe(mt catalog.MessageType, recipient *id.ID, payload []byte, params clientE2E.Params) ([]id.Round, time.Time, error) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testE2eManager) EnableUnsafeReception() {
	//TODO implement me
	panic("implement me")
}

func (tnm *testE2eManager) GetGroup() *cyclic.Group {
	//TODO implement me
	panic("implement me")
}

func (tnm *testE2eManager) GetReceptionID() *id.ID {
	//TODO implement me
	panic("implement me")
}

func (tnm *testE2eManager) FirstPartitionSize() uint {
	//TODO implement me
	panic("implement me")
}

func (tnm *testE2eManager) SecondPartitionSize() uint {
	//TODO implement me
	panic("implement me")
}

func (tnm *testE2eManager) PartitionSize(payloadIndex uint) uint {
	//TODO implement me
	panic("implement me")
}

func (tnm *testE2eManager) PayloadSize() uint {
	//TODO implement me
	panic("implement me")
}
