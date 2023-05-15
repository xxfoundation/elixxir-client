package notifications

import (
	"bytes"
	"encoding/json"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	pb "gitlab.com/elixxir/comms/mixmessages"
	notifCrypto "gitlab.com/elixxir/crypto/notifications"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"time"
)

func (m *manager) Set(toBeNotifiedOn *id.ID, group string, metadata []byte, status NotificationState) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	currentReg, exists := m.notifications[*toBeNotifiedOn]
	if exists {
		if currentReg.Status == status &&
			bytes.Equal(metadata, currentReg.Metadata) {
			return nil
		}
	}

	// register with remote
	if status == Push && (!exists || exists && currentReg.Status != Push) {
		if err := m.registerNotification(toBeNotifiedOn); err != nil {
			return err
		}
	} else if status != Push {
		if err := m.unregisterNotification(toBeNotifiedOn); err != nil {
			return err
		}
	}

	ts := netTime.Now()

	reg := registration{
		Group: group,
		State: State{
			Metadata: copyBytes(metadata),
			Status:   status,
		},
	}

	err := m.storeRegistration(toBeNotifiedOn, reg, ts)
	if err != nil {
		return err
	}

	// update in ram storage
	m.upsertNotificationUnsafeRAM(toBeNotifiedOn, reg)

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

	if r.Status == Push {
		if err := m.unregisterNotification(toBeNotifiedOn); err != nil {
			return err
		}
	}

	elementName := makeElementName(toBeNotifiedOn)

	_, err := m.remote.DeleteMapElement(notificationsMap, elementName,
		notificationsMapVersion)
	m.deleteNotificationUnsafeRAM(toBeNotifiedOn)
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
func (m *manager) registerNotification(nid *id.ID) error {
	iid, err := ephemeral.GetIntermediaryId(nid)
	if err != nil {
		return err
	}

	ts := netTime.Now().UTC()

	stream := m.rng.GetStream()
	sig, err := notifCrypto.SignIdentity(m.transmissionRSA, iid, ts,
		notifCrypto.RegisterTrackedIDTag, stream)
	stream.Close()
	if err != nil {
		return err
	}

	_, err = m.comms.RegisterTrackedID(m.notificationHost,
		&pb.TrackedIntermediaryIDRequest{
			TrackedIntermediaryID: iid,
			TransmissionRSAPem:    m.transmissionRSAPubPem,
			RequestTimestamp:      ts.UnixNano(),
			Signature:             sig,
		})

	return err
}

// unregisterNotification unregisters to receive notifications on the given
// id from remote.
func (m *manager) unregisterNotification(nid *id.ID) error {
	iid, err := ephemeral.GetIntermediaryId(nid)
	if err != nil {
		return err
	}

	ts := netTime.Now().UTC()

	stream := m.rng.GetStream()
	sig, err := notifCrypto.SignIdentity(m.transmissionRSA, iid, ts,
		notifCrypto.UnregisterTrackedIDTag, stream)
	stream.Close()
	if err != nil {
		return err
	}

	_, err = m.comms.UnregisterTrackedID(m.notificationHost,
		&pb.TrackedIntermediaryIDRequest{
			TrackedIntermediaryID: iid,
			TransmissionRSAPem:    m.transmissionRSAPubPem,
			RequestTimestamp:      ts.UnixNano(),
			Signature:             sig,
		})

	return err
}

func copyBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
