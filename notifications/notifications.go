package notifications

import (
	"bytes"
	"encoding/json"
	"errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	pb "gitlab.com/elixxir/comms/mixmessages"
	notifCrypto "gitlab.com/elixxir/crypto/notifications"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"time"
)

func (m *manager) Set(toBeNotifiedOn *id.ID, group string, metadata []byte,
	status NotificationState) error {
	if err := status.IsValid(); err != nil {
		return err
	}

	m.mux.Lock()
	defer m.mux.Unlock()

	currentReg, exists := m.notifications[*toBeNotifiedOn]
	if exists {
		if currentReg.Group != group {
			return errors.New("cannot change the group of a notification " +
				"registration")
		}
		if currentReg.Status == status &&
			bytes.Equal(metadata, currentReg.Metadata) {
			return nil
		}
	}

	reg := registration{
		Group:     group,
		Confirmed: false,
		State: State{
			Metadata: copyBytes(metadata),
			Status:   status,
		},
	}

	ts := netTime.Now()

	err := m.storeRegistration(toBeNotifiedOn, reg, ts)
	if err != nil {
		return err
	}

	// update in ram storage
	m.upsertNotificationUnsafeRAM(toBeNotifiedOn, reg)

	// register with remote
	if status == Push && (!exists || exists && currentReg.Status != Push) || status != Push {
		to := time.NewTimer(5 * time.Second)
		select {
		case m.regChan <- pendingRegistration{
			r:   reg,
			nid: toBeNotifiedOn,
		}:
			jww.DEBUG.Printf("Sent registration to handler")
		case <-to.C:
			return errors.New("timed out sending registration to handler")
		}
	}
	return nil
}

func (m *manager) Get(toBeNotifiedOn *id.ID) (NotificationState, []byte, string, bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	r, exist := m.notifications[*toBeNotifiedOn]
	if exist {
		return r.Status, copyBytes(r.Metadata), r.Group, true
	} else {
		return 255, nil, "", false
	}
}

func (m *manager) Delete(toBeNotifiedOn *id.ID) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	r, exist := m.notifications[*toBeNotifiedOn]
	if !exist {
		return nil
	}

	r.PendingDeletion = true

	ts := netTime.Now()

	err := m.storeRegistration(toBeNotifiedOn, r, ts)
	if err != nil {
		return err
	}

	// update in ram storage
	m.upsertNotificationUnsafeRAM(toBeNotifiedOn, r)
	to := time.NewTimer(5 * time.Second)
	select {
	case m.regChan <- pendingRegistration{
		r:   r,
		nid: toBeNotifiedOn,
	}:
	case <-to.C:
		return errors.New("timed out sending deletion to channel")
	}

	return err
}

func (m *manager) storeRegistration(nid *id.ID, reg registration,
	ts time.Time) error {

	ts = ts.UTC()
	regBytes, err := json.Marshal(&reg)
	if err != nil {
		return err
	}

	// update remote storage
	elementName := makeElementName(nid)
	err = m.remote.StoreMapElement(notificationsMap, elementName,
		&versioned.Object{
			Version:   notificationsMapVersion,
			Timestamp: ts,
			Data:      regBytes,
		}, notificationsMapVersion)
	return err
}

func (m *manager) GetGroup(group string) (Group, bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	g, exists := m.group[group]
	if !exists {
		return nil, false
	}
	return g.DeepCopy(), true
}

// registerNotification registers to receive notifications on the given
// id from remote.
func (m *manager) registerNotification(nids []*id.ID) error {
	iidLst := make([][]byte, len(nids))
	for i, nid := range nids {
		iid, err := ephemeral.GetIntermediaryId(nid)
		if err != nil {
			return err
		}
		iidLst[i] = iid
	}

	ts := netTime.Now().UTC()

	stream := m.rng.GetStream()
	sig, err := notifCrypto.SignIdentity(m.transmissionRSA, iidLst, ts,
		notifCrypto.RegisterTrackedIDTag, stream)
	stream.Close()
	if err != nil {
		return err
	}

	_, err = m.comms.RegisterTrackedID(m.notificationHost,
		&pb.RegisterTrackedIdRequest{
			Request: &pb.TrackedIntermediaryIdRequest{
				TrackedIntermediaryID: iidLst,
				TransmissionRsaPem:    m.transmissionRSAPubPem,
				RequestTimestamp:      ts.UnixNano(),
				Signature:             sig,
			},
			RegistrationTimestamp:       m.registrationTimestampNs,
			TransmissionRsaRegistrarSig: m.transmissionRegistrationValidationSignature,
		})

	return err
}

// unregisterNotification unregisters to receive notifications on the given
// id from remote.
func (m *manager) unregisterNotification(nids []*id.ID) error {
	iidLst := make([][]byte, len(nids))
	for i, nid := range nids {
		iid, err := ephemeral.GetIntermediaryId(nid)
		if err != nil {
			return err
		}
		iidLst[i] = iid
	}

	ts := netTime.Now().UTC()

	stream := m.rng.GetStream()
	sig, err := notifCrypto.SignIdentity(m.transmissionRSA, iidLst, ts,
		notifCrypto.UnregisterTrackedIDTag, stream)
	stream.Close()
	if err != nil {
		return err
	}

	_, err = m.comms.UnregisterTrackedID(m.notificationHost,
		&pb.UnregisterTrackedIdRequest{Request: &pb.TrackedIntermediaryIdRequest{
			TrackedIntermediaryID: iidLst,
			TransmissionRsaPem:    m.transmissionRSAPubPem,
			RequestTimestamp:      ts.UnixNano(),
			Signature:             sig,
		}})

	return err
}

func copyBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
