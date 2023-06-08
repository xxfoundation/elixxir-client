////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/client/v4/collective"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/xx_network/primitives/netTime"
)

// Storage values.
const (
	dmMapName      = "dmMap"
	dmMapVersion   = 0
	dmStoreVersion = 0
)

// userStore contains the DM remote KV storage
type userStore struct {
	remote versioned.KV
	mux    sync.Mutex
}

// newUserStore initialises a new userStore.
func newUserStore(kv versioned.KV) (*userStore, error) {
	remote, err := kv.Prefix(collective.StandardRemoteSyncPrefix)
	if err != nil {
		return nil, err
	}

	err = remote.ListenOnRemoteMap(dmMapName, dmMapVersion, nil, false)
	if err != nil && remote.Exists(err) {
		return nil, errors.Wrap(err, "failed to load and listen to remote "+
			"updates on dm user storage")
	}

	return &userStore{remote: kv}, nil
}

// dmUser stores information for each user the current user has a DM
// conversation with.
type dmUser struct {
	// PublicKey is the user's public key. It is not included in JSON.
	PublicKey ed25519.PublicKey `json:"-"`

	// Status indicates the notification status for the user or if they are
	// blocked.
	Status userStatus `json:"s"`
}

// String prints the dmUser in a human-readable form for logging and debugging.
// This function adheres to the fmt.Stringer interface.
func (dmu *dmUser) String() string {
	fields := []string{
		hex.EncodeToString(dmu.PublicKey),
		strconv.Itoa(int(dmu.Status)),
	}
	return "{" + strings.Join(fields, " ") + "}"
}

// userStatus represents the notification status or blocked status of the user.
type userStatus uint32

const (
	statusMute      userStatus = 10
	statusNotifyAll userStatus = 20
	statusBlocked   userStatus = 50

	// defaultStatus is set when adding a new user or resetting a status.
	defaultStatus = statusNotifyAll
)

// set saves the dmUser info to storage keyed on the Ed25519 public key.
func (us *userStore) set(pubKey ed25519.PublicKey, status userStatus) {
	us.mux.Lock()
	defer us.mux.Unlock()
	us.setUnsafe(pubKey, status)
}

func (us *userStore) setUnsafe(pubKey ed25519.PublicKey, status userStatus) {
	elemName := marshalElementName(pubKey)
	data, err := json.Marshal(dmUser{
		Status: status,
	})
	if err != nil {
		jww.FATAL.Panicf("[DM] Failed to JSON marshal user %X for storage: %+v",
			pubKey, err)
	}

	obj := &versioned.Object{
		Version:   dmStoreVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	err = us.remote.StoreMapElement(dmMapName, elemName, obj, dmMapVersion)
	if err != nil {
		jww.FATAL.Panicf("[DM] Failed to set user %X: %+v", pubKey, err)
	}
}

// get returns the dmUser from storage. Returns false if the user does not
// exist.
func (us *userStore) get(pubKey ed25519.PublicKey) (user *dmUser, exists bool) {
	elemName := marshalElementName(pubKey)
	us.mux.Lock()
	obj, err := us.remote.GetMapElement(dmMapName, elemName, dmMapVersion)
	us.mux.Unlock()
	if err != nil {
		if us.remote.Exists(err) {
			jww.FATAL.Panicf("[DM] Failed to load user %X from storage: %+v",
				pubKey, err)
		} else {
			return nil, false
		}
	}

	user = &dmUser{}
	if err = json.Unmarshal(obj.Data, user); err != nil {
		jww.FATAL.Panicf("[DM] Failed to JSON unmarshal user %X from storage: %+v",
			pubKey, err)
	}
	user.PublicKey = pubKey

	return user, true
}

// getOrSet returns the dmUser from storage. If the user does not exist, then it
// is added with the default status and returned.
// TODO: test
func (us *userStore) getOrSet(pubKey ed25519.PublicKey) *dmUser {
	elemName := marshalElementName(pubKey)
	us.mux.Lock()
	defer us.mux.Unlock()
	obj, err := us.remote.GetMapElement(dmMapName, elemName, dmMapVersion)
	if err != nil {
		if us.remote.Exists(err) {
			jww.FATAL.Panicf("[DM] Failed to load user %X from storage: %+v",
				pubKey, err)
		}

		us.setUnsafe(pubKey, defaultStatus)
		return &dmUser{
			PublicKey: pubKey,
			Status:    defaultStatus,
		}
	}

	var user dmUser
	if err = json.Unmarshal(obj.Data, &user); err != nil {
		jww.FATAL.Panicf("[DM] Failed to JSON unmarshal user %X from storage: %+v",
			pubKey, err)
	}
	user.PublicKey = pubKey

	return &user
}

// delete removes the dmUser for the public key from storage.
func (us *userStore) delete(pubKey ed25519.PublicKey) {
	elemName := marshalElementName(pubKey)
	us.mux.Lock()
	_, err := us.remote.DeleteMapElement(dmMapName, elemName, dmMapVersion)
	us.mux.Unlock()
	if err != nil {
		jww.FATAL.Panicf("[DM] Failed to delete user %X from storage: %+v",
			pubKey, err)
	}
}

// getAll returns a list of all users in storage.
func (us *userStore) getAll() []*dmUser {
	var users []*dmUser
	us.iterate(func(n int) { users = make([]*dmUser, 0, n) },
		func(user *dmUser) { users = append(users, user) })
	return users
}

// iterate loops through all users in storage, unmarshalls each one, and passes
// it into the add function. Before add is called, init is called with the total
// number of users storage. Init can be nil.
func (us *userStore) iterate(init func(n int), add func(user *dmUser)) {
	us.mux.Lock()
	userMap, err := us.remote.GetMap(dmMapName, dmMapVersion)
	us.mux.Unlock()
	if err != nil {
		jww.FATAL.Panicf("[DM] Failed to load map %s from storage: %+v",
			dmMapName, err)
	}

	if init != nil {
		init(len(userMap))
	}
	for elemName, obj := range userMap {
		var user dmUser
		user.PublicKey, err = unmarshalElementName(elemName)
		if err != nil {
			jww.ERROR.Printf("[DM] Failed to parse element name %s: %+v",
				elemName, err)
			continue
		}
		if err = json.Unmarshal(obj.Data, &user); err != nil {
			jww.ERROR.Printf("[DM] Failed to parse user for element name %q: %+v",
				elemName, err)
			continue
		}

		add(&user)
	}
}

// elementEdit describes a single edit in the userStore KV storage.
type elementEdit struct {
	old       *dmUser
	new       *dmUser
	operation versioned.KeyOperation
}


// String prints the elementEdit in a human-readable form for logging and
// debugging. This function adheres to the fmt.Stringer interface.
func (ee elementEdit) String() string {
	fields := []string{
		"old:" + fmt.Sprint(ee.old),
		"new:" + fmt.Sprint(ee.new),
		"operation:" + ee.operation.String(),
	}

	return "{" + strings.Join(fields, " ") + "}"
}

// listen is called when the map or map elements are updated remotely or
// locally. The public key will never change between an old and new pair in the
// elementEdit list.
func (us *userStore) listen(cb func(edits []elementEdit)) error {
	us.mux.Lock()
	defer us.mux.Unlock()
	return us.remote.ListenOnRemoteMap(dmMapName, dmMapVersion,
		func(edits map[string]versioned.ElementEdit) {
			userEdits := make([]elementEdit, 0, len(edits))
			for elemName, edit := range edits {
				pubKey, err := unmarshalElementName(elemName)
				if err != nil {
					jww.ERROR.Printf("[DM] Failed to parse element name %s: %+v",
						elemName, err)
					continue
				}

				e := elementEdit{
					operation: edit.Operation,
				}

				if len(edit.OldElement.Data) > 0 {
					err = json.Unmarshal(edit.OldElement.Data, &e.old)
					if err != nil {
						jww.ERROR.Printf("[DM] Failed to parse old user for "+
							"element name %q: %+v", elemName, err)
						continue
					} else {
						e.old.PublicKey = pubKey
					}
				}

				if len(edit.NewElement.Data) > 0 {
					err = json.Unmarshal(edit.NewElement.Data, &e.new)
					if err != nil {
						jww.ERROR.Printf("[DM] Failed to parse new user for "+
							"element name %q: %+v", elemName, err)
						continue
					} else {
						e.new.PublicKey = pubKey
					}
				}

				userEdits = append(userEdits, e)
			}
			cb(userEdits)
		}, true)
}

// marshalElementName marshals the [ed25519.PublicKey] into a string for use as
// the element name in storage.
//
// base64.RawStdEncoding is used instead of base64.StdEncoding because it
// excludes unneeded padding that can save a few extra bytes of storage.
func marshalElementName(pubKey ed25519.PublicKey) string {
	return base64.RawStdEncoding.EncodeToString(pubKey)
}

// unmarshalElementName marshals the marshalled elementName into a
// [ed25519.PublicKey].
//
// Note the usage of base64.RawStdEncoding (refer to marshalElementName for more
// info).
func unmarshalElementName(elementName string) (ed25519.PublicKey, error) {
	return base64.RawStdEncoding.DecodeString(elementName)
}
