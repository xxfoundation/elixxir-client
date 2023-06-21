package notifications

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/collective"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/client/v4/xxdk"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"golang.org/x/crypto/blake2b"
	"sync"
)

const (
	prefixConst             = "notificationsManager:%x"
	notificationsMap        = "notificationsRegistrations"
	notificationsMapVersion = 0
	tokenStorageKey         = "tokenStorageKey"
	tokenStorageVersion     = 0
	maxStateKey             = "maxStateKey"
	maxStateKetVersion      = 0
)

type manager struct {
	// internal notifications tracking data structures
	notifications map[id.ID]registration
	group         map[string]Group  // ordered by group for easy access
	callbacks     map[string]Update //update events
	mux           sync.RWMutex

	// internal data defining notifications access
	transmissionRSA                             rsa.PrivateKey
	transmissionRSAPubPem                       []byte
	transmissionRegistrationValidationSignature []byte
	registrationTimestampNs                     int64
	transmissionSalt                            []byte

	// external refrences
	comms Comms
	rng   *fastRNG.StreamGenerator

	// notificationHost stores the host of the remote notifications server
	// will be nil if this device does not talk to notifications
	notificationHost *connect.Host

	token tokenReg

	local  versioned.KV
	remote versioned.KV

	maxState       NotificationState
	initialization bool
}

type registration struct {
	Group string
	State
}

type tokenReg struct {
	Token string `json:"token"`
	App   string `json:"app"`
}

// NewOrLoadManager creates a new notifications manager for tracking and
// registering notifications.  Depends on the remote synchronization of the
// [collective.SyncKV].
// Will not register notifications with the remote if `allowRemoteRegistration`
// is false, which should be the case for web based instantiations
func NewOrLoadManager(identity xxdk.TransmissionIdentity, regSig []byte,
	kv versioned.KV, comms Comms, rng *fastRNG.StreamGenerator) Manager {

	var nbHost *connect.Host

	var exists bool
	nbHost, exists = comms.GetHost(&id.NotificationBot)
	if !exists {
		jww.FATAL.Panicf("Notification bot not registered, " +
			"notifications cannot be startedL")
	}

	kvLocal, err := kv.Prefix(prefix(identity.RSAPrivate.Public()))
	if err != nil {
		jww.FATAL.Panicf("Notifications failed to prefix kv")
	}

	kvRemote, err := kvLocal.Prefix(collective.StandardRemoteSyncPrefix)
	if err != nil {
		jww.FATAL.Panicf("Notifications failed to prefix kv")
	}

	m := &manager{
		transmissionRSA:                             identity.RSAPrivate,
		transmissionRSAPubPem:                       identity.RSAPrivate.Public().MarshalPem(),
		transmissionRegistrationValidationSignature: regSig,
		registrationTimestampNs:                     identity.RegistrationTimestamp,
		transmissionSalt:                            identity.Salt,
		comms:                                       comms,
		rng:                                         rng,
		notificationHost:                            nbHost,
		local:                                       kvLocal,
		remote:                                      kvRemote,
		callbacks:                                   make(map[string]Update),
		notifications:                               make(map[id.ID]registration),
		group:                                       make(map[string]Group),
		maxState:                                    Push,
		initialization:                              true,
	}

	// lock so that an update cannot run while we are loading the basic
	// notifications structure from disk into ram

	err = m.remote.ListenOnRemoteKey(maxStateKey,
		maxStateKetVersion, m.maxStateUpdate, false)
	if err != nil && ekv.Exists(err) {
		jww.FATAL.Panicf("Could not load notifications state key: %+v", err)
	}
	err = m.remote.ListenOnRemoteMap(notificationsMap,
		notificationsMapVersion, m.mapUpdate, false)
	if err != nil {
		jww.FATAL.Panicf("Could not load notifications map: %+v", err)
	}
	m.mux.Lock()
	m.loadTokenUnsafe()
	m.initialization = false
	m.mux.Unlock()

	return m
}

func (m *manager) RegisterUpdateCallback(group string, nu Update) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.callbacks[group] = nu
	if g, ok := m.group[group]; ok {
		nu(g, nil, nil, nil, m.maxState)
	}
}

// mapUpdate is the listener function which is called whenever the notifications
// data is updated based upon a remote sync
func (m *manager) mapUpdate(edits map[string]versioned.ElementEdit) {

	updates := make(groupChanges)

	m.mux.Lock()
	defer m.mux.Unlock()

	// process edits
	for elementName, edit := range edits {
		nID, err := getIDFromElementName(elementName)
		if err != nil {
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
			m.deleteNotificationUnsafeRAM(nID)
			continue
		}

		newUpdate := registration{}
		if err := json.Unmarshal(edit.NewElement.Data, &newUpdate); err != nil {
			jww.WARN.Printf("Failed to unmarshal data in notification "+
				"update %s, skipping: %+v", elementName, err)
			continue
		}

		if edit.Operation == versioned.Created ||
			edit.Operation == versioned.Loaded {
			updates.AddCreated(newUpdate.Group, nID)
		} else if edit.Operation == versioned.Updated {
			updates.AddEdit(newUpdate.Group, nID)
		} else {
			jww.WARN.Printf("Failed to handle notification update %s, "+
				"bad operation: %s, skipping", elementName, edit.Operation)
			continue
		}
		m.upsertNotificationUnsafeRAM(nID, newUpdate)
	}

	//call callbacks
	for groupName, update := range updates {
		if cb, exists := m.callbacks[groupName]; exists {
			// can be nil if the last element was deleted
			group, _ := m.group[groupName]
			go cb(group.DeepCopy(), update.created, update.edit,
				update.deletion, m.maxState)
		}
	}
}

// loadNotificationsUnsafe loads the notifications from the local storage.
// does not take the lock and cannot run concurrently with the update function
// must be called under the lock
func (m *manager) loadNotificationsUnsafe(mapObj map[string]*versioned.Object) {

	for key, regObj := range mapObj {
		reg := registration{}

		if err := json.Unmarshal(regObj.Data, &reg); err != nil {
			jww.WARN.Printf("Failed to unmarshal notifications "+
				"registration for %s, skipping: %+v", key, err)
			continue
		}
		nID, err := getIDFromElementName(key)
		if err != nil {
			jww.WARN.Printf("Failed to unmarshal notifications "+
				"registration id for %s, skipping: %+v", key, err)
			continue
		}

		m.upsertNotificationUnsafeRAM(nID, reg)
	}
}

func (m *manager) maxStateUpdate(old, new *versioned.Object, op versioned.KeyOperation) {
	if op == versioned.Deleted {
		jww.FATAL.Panicf("Notifications max state key cannot be deleted")
	}

	m.mux.Lock()
	defer m.mux.Unlock()

	if err := json.Unmarshal(new.Data, &m.maxState); err != nil {
		jww.WARN.Printf("failed to unmarshal %s, ignoring: %+v",
			maxStateKey, err)
		return
	}
	if !m.initialization {
		for g := range m.callbacks {
			cb := m.callbacks[g]
			go cb(m.group[g].DeepCopy(), nil, nil, nil, m.maxState)
		}
	} else {
		jww.DEBUG.Printf("Skipping callback on masStateUpdate to %s, "+
			"in initialization", m.maxState)
	}

}

func (m *manager) loadMaxStateUnsafe(obj *versioned.Object) {
	err := json.Unmarshal(obj.Data, &m.maxState)
	if err != nil {
		jww.WARN.Printf("failed to unmarshal %s, defaulting to %s. "+
			"This should occur only on first run : %+v", maxStateKey, Push, err)
		m.setMaxStateUnsafe(Push)
	}
}

func (m *manager) setMaxStateUnsafe(max NotificationState) {

	b, err := json.Marshal(&max)
	if err != nil {
		jww.FATAL.Panicf("Failed to set max notifications sate to %s:"+
			" %+v", max, err)
	}

	err = m.remote.Set(maxStateKey, &versioned.Object{
		Version:   maxStateKetVersion,
		Timestamp: netTime.Now(),
		Data:      b,
	})
	if err != nil {
		jww.FATAL.Panicf("Failed to set max notifications sate to %s:"+
			" %+v", max, err)
	}

	m.maxState = max
}

// upsertNotificationUnsafeRAM adds the given notification registration to the
// in ram storage, both to notification and groups
// must be called under the lock
func (m *manager) upsertNotificationUnsafeRAM(nid *id.ID, reg registration) {
	m.notifications[*nid] = reg
	m.addToGroupUnsafeRAM(nid, reg)
}

// addToGroupUnsafeRAM adds the given notification registration to the
// groups in ram storage
// must be called under the lock
func (m *manager) addToGroupUnsafeRAM(nID *id.ID, reg registration) {
	g, exists := m.group[reg.Group]
	if !exists {
		g = make(Group)
	}
	g[*nID] = reg.State
	m.group[reg.Group] = g
}

// deleteNotificationUnsafeRAM removes the given notification registration from
// the in ram storage, both to notification and groups
// must be called under the lock
// returns the group becasue it may be nesseary
func (m *manager) deleteNotificationUnsafeRAM(nid *id.ID) string {
	reg, exists := m.notifications[*nid]
	if !exists {
		return ""
	}

	groupList := m.group[reg.Group]
	if len(groupList) == 1 {
		delete(m.group, reg.Group)
	} else {
		delete(groupList, *nid)
		m.group[reg.Group] = groupList
	}

	delete(m.notifications, *nid)

	return reg.Group
}

// setTokenUnsafe sets the token in ram and on disk, locally only. Returns true
// if the token was not net before
// must be called under the lock
func (m *manager) setTokenUnsafe(token, app string) bool {
	setBefore := m.token.Token != ""
	m.token = tokenReg{
		Token: token,
		App:   app,
	}
	tokenBytes, err := json.Marshal(m.token)
	if err != nil {
		jww.FATAL.Panicf("Failed to marshal Token to disk to %s: %+v",
			token, err)
	}

	err = m.local.Set(tokenStorageKey, &versioned.Object{
		Version:   tokenStorageVersion,
		Timestamp: netTime.Now(),
		Data:      tokenBytes,
	})
	if err != nil {
		jww.FATAL.Panicf("Failed to set Token on disk to %s: %+v", token, err)
	}
	return setBefore
}

// deleteTokenUnsafe deletes the token from ram and disk locally.
// returns true if it existed
// must be called under the lock
func (m *manager) deleteTokenUnsafe() bool {
	setBefore := m.token.Token != ""
	if setBefore {
		err := m.local.Delete(tokenStorageKey, tokenStorageVersion)
		if err != nil {
			jww.FATAL.Panicf("Failed to delete Token on disk to %s: %+v",
				m.token.Token, err)
		}
	}
	m.token = tokenReg{}
	return setBefore
}

// loadTokenUnsafe loads the token from disk, setting it to empty if it cannot be
// found
// must be called under the lock
func (m *manager) loadTokenUnsafe() {
	tokenObj, err := m.local.Get(tokenStorageKey, tokenStorageVersion)
	if err != nil {
		if ekv.Exists(err) {
			jww.FATAL.Panicf("Error received from ekv on loading "+
				"Token: %+v", err)
		} else {
			// no token has been registered
			jww.DEBUG.Printf("No token found on disk, assuming we have" +
				"not registered")
			return
		}
	}

	if err = json.Unmarshal(tokenObj.Data, &m.token); err != nil {
		jww.WARN.Printf("Failed to unmarshal token from disk, operating as if no token is present: %+v", err)
	}

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

func makeElementName(nid *id.ID) string {
	return base64.StdEncoding.EncodeToString(nid[:])
}

func getIDFromElementName(elementName string) (*id.ID, error) {
	b, err := base64.StdEncoding.DecodeString(elementName)
	if err != nil {
		return nil, err
	}
	return id.Unmarshal(b)
}
