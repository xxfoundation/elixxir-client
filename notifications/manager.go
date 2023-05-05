package notifications

import (
	"encoding/json"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/client/v4/xxdk"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"golang.org/x/crypto/blake2b"
	"sync"
)

const (
	prefixConst             = "notificationsManager:%x"
	notificationsMap        = "notificationsRegistrations"
	notificationsMapVersion = 0
)

type manager struct {
	// internal notifications tracking data structures
	notifications map[id.ID]registration
	group         map[string]Group  // ordered by group for easy access
	callbacks     map[string]Update //update events
	mux           sync.Mutex

	// internal data defining notifications access
	transmissionRSA                             rsa.PrivateKey
	transmissionRegistrationValidationSignature []byte
	registrationTimestampNs                     int64
	registrationSalt                            []byte

	// external refrences
	comms *client.Comms
	rng   *fastRNG.StreamGenerator

	// notificationHost stores the host of the remote notifications server
	// will be nil if this device does not talk to notifications
	notificationHost *connect.Host

	local  versioned.KV
	remote versioned.KV
}

type registration struct {
	Group string
	Registered bool
	State
}

// NewOrLoadManager creates a new notifications manager for tracking and
// registering notifications.  Depends on the remote synchronization of the
// [collective.SyncKV].
// Will not register notifications with the remote if `allowRemoteRegistration`
// is false, which should be the case for web based instantiations
func NewOrLoadManager(identity xxdk.TransmissionIdentity, regSig []byte,
	kv versioned.KV, comms *client.Comms, rng *fastRNG.StreamGenerator,
	allowRemoteRegistration bool) Manger {

	var nbHost *connect.Host
	if allowRemoteRegistration{
		var exists bool
		nbHost, exists = comms.GetHost(&id.NotificationBot)
		if !exists {
			jww.FATAL.Panicf("Notification bot not registered, " +
				"notifications cannot be startedL")
		}
	}

	kvLocal, err := kv.Prefix(prefix(identity.RSAPrivate.Public()))
	if err != nil {
		jww.FATAL.Panicf("Notifications failed to prefix kv")
	}

	kvRemote, err := kvLocal.Prefix(versioned.StandardRemoteSyncPrefix)
	if err != nil {
		jww.FATAL.Panicf("Notifications failed to prefix kv")
	}

	m := &manager{
		transmissionRSA: identity.RSAPrivate,
		transmissionRegistrationValidationSignature: regSig,
		registrationTimestampNs:                     identity.RegistrationTimestamp,
		registrationSalt:                            identity.Salt,
		comms:                                       comms,
		rng:                                         rng,
		notificationHost:                            nbHost,
		local:                                       kvLocal,
		remote:                                      kvRemote,
	}

	// lock so that an update cannot run while we are loading the basic
	// notifications structure from disk into ram
	m.mux.Lock()
	m.remote.ListenOnRemoteMap(notificationsMap, notificationsMapVersion, m.mapUpdate)
	m.loadNotifications()
	m.mux.Unlock()

}


// mapUpdate is the listener function which is called whenever the notifications
// data is updated based upon a remote sync
func (m *manager) mapUpdate(mapName string, edits map[string]versioned.ElementEdit) {
	if mapName != notificationsMap {
		jww.ERROR.Printf("Got an update for the wrong map, "+
			"expected: %s, got: %s", notificationsMap, mapName)
		return
	}

	updates := make(groupChanges)

	m.mux.Lock()
	defer m.mux.Unlock()

	// process edits
	for elementName, edit := range edits {
		nID := &id.ID{}
		if err := nID.UnmarshalText([]byte(elementName)); err != nil {
			jww.WARN.Printf("Failed to unmarshal id in notification "+
				"update %s on operation %s , skipping: %+v", elementName,
				edit.Operation, err)
			continue
		}

		if edit.Operation == versioned.Deleted {
			// get the group and see if we have it locally
			localReg, exists := m.notifications[*nID]
			if !exists {
				// if we don't have it locally, skip
				continue
			}
			updates.AddDeletion(localReg.Group, nID)
			m.deleteNotificationRAM(nID)
			continue
		}

		newUpdate := registration{}
		if err := json.Unmarshal(edit.NewElement.Data, &newUpdate); err != nil {
			jww.WARN.Printf("Failed to unmarshal data in notification "+
				"update %s, skipping: %+v", elementName, err)
			continue
		}

		if edit.Operation == versioned.Created {
			updates.AddCreated(newUpdate.Group, nID)
		} else if edit.Operation == versioned.Updated {
			updates.AddEdit(newUpdate.Group, nID)
		} else {
			jww.WARN.Printf("Failed to handle notification update %s, "+
				"bad operation: %s, skipping", elementName, edit.Operation)
			continue
		}
		m.upsertNotificationRAM(nID, newUpdate)
		if newUpdate.Status && !newUpdate.Registered
	}

	//call callbacks
	for groupName, update := range updates {
		if cb, exists := m.callbacks[groupName]; exists {
			// can be nil if the last element was deleted
			group, _ := m.group[groupName]
			go cb(group, update.created, update.edit,
				update.deletion)
		}
	}
}

// loadNotifications loads the notifications from the local storage.
// does not take the lock and cannot run concurrently with the update function
func (m *manager) loadNotifications() {

	mapObj, err := m.remote.GetMap(notificationsMap, notificationsMapVersion)
	if err != nil {
		jww.WARN.Printf("Notifications map not found, creating from scratch: %+v", err)
		m.notifications = make(map[id.ID]registration)
		m.group = make(map[string]Group)
		return
	}

	for key, regObj := range mapObj {
		reg := registration{}

		if err = json.Unmarshal(regObj.Data, &reg); err != nil {
			jww.WARN.Printf("Failed to unmarshal notifications "+
				"registration for %s, skipping: %+v", key, err)
			continue
		}
		nID := &id.ID{}
		if err = nID.UnmarshalText([]byte(key)); err != nil {
			jww.WARN.Printf("Failed to unmarshal notifications "+
				"registration id for %s, skipping: %+v", key, err)
			continue
		}

		m.upsertNotificationRAM(nID, reg)
	}
}

func (m *manager) upsertNotificationRAM(nid *id.ID, reg registration) {
	m.notifications[*nid] = reg
	m.addToGroupRAM(nid, reg)
}

func (m *manager) addToGroupRAM(nID *id.ID, reg registration) {
	g, exists := m.group[reg.Group]
	if !exists {
		g = make(Group)
	}
	g[*nID] = reg.State
	m.group[reg.Group] = g
}

func (m *manager) deleteNotificationRAM(nid *id.ID) {
	reg, exists := m.notifications[*nid]
	if !exists {
		return
	}

	groupList := m.group[reg.Group]
	if len(groupList) == 1 {
		delete(m.group, reg.Group)
	}

	delete(groupList, *nid)
	m.group[reg.Group] = groupList

	delete(m.notifications, *nid)
}

// data structure to make map updates cleaner
type groupChange struct {
	created  []*id.ID
	edit     []*id.ID
	deletion []*id.ID
}

type groupChanges map[string]groupChange

func (gc *groupChanges) AddCreated(groupName string, nid *id.ID) {
	group := gc.get(groupName)
	group.created = append(group.created, nid)
	(*gc)[groupName] = group
}

func (gc *groupChanges) AddEdit(groupName string, nid *id.ID) {
	group := gc.get(groupName)
	group.edit = append(group.edit, nid)
	(*gc)[groupName] = group
}

func (gc *groupChanges) AddDeletion(groupName string, nid *id.ID) {
	group := gc.get(groupName)
	group.deletion = append(group.deletion, nid)
	(*gc)[groupName] = group
}

func (gc *groupChanges) get(groupName string) groupChange {
	if group, exists := (*gc)[groupName]; exists {
		return group
	} else {
		return groupChange{}
	}
}

func prefix(pubkey rsa.PublicKey) string {
	h, _ := blake2b.New256(nil)
	h.Write(pubkey.MarshalPem())
	return fmt.Sprintf(prefixConst, h.Sum(nil))
}
