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

	// Token is the unique DM token for the user.
	Token uint32 `json:"t"`
}

// String prints the dmUser in a human-readable form for logging and debugging.
// This function adheres to the fmt.Stringer interface.
func (dmu *dmUser) String() string {
	fields := []string{
		hex.EncodeToString(dmu.PublicKey),
		strconv.Itoa(int(dmu.Status)),
		strconv.Itoa(int(dmu.Token)),
	}
	return "{" + strings.Join(fields, " ") + "}"
}

// userStatus represents the notification status or blocked status of the user.
type userStatus uint32

const (
	statusMute      userStatus = 10
	statusNotifyAll userStatus = 20
	statusBlocked   userStatus = 50
)

// set saves the dmUser info to storage keyed on the Ed25519 public key.
func (us *userStore) set(pubKey ed25519.PublicKey, status userStatus, token uint32) error {
	elemName := marshalElementName(pubKey)
	data, err := json.Marshal(dmUser{
		Status: status,
		Token:  token,
	})
	if err != nil {
		return err
	}

	us.mux.Lock()
	defer us.mux.Unlock()
	return us.remote.StoreMapElement(dmMapName, elemName, &versioned.Object{
		Version:   dmStoreVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}, dmMapVersion)
}

// update updates the status for the dmUser keyed on the Ed25519 public key.
func (us *userStore) update(pubKey ed25519.PublicKey, status userStatus) error {
	elemName := marshalElementName(pubKey)
	us.mux.Lock()
	defer us.mux.Unlock()
	obj, err := us.remote.GetMapElement(dmMapName, elemName, dmMapVersion)
	if err != nil {
		return err
	}

	var user dmUser
	if err = json.Unmarshal(obj.Data, &user); err != nil {
		return err
	}
	user.Status = status

	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	return us.remote.StoreMapElement(dmMapName, elemName, &versioned.Object{
		Version:   dmStoreVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}, dmMapVersion)
}

// get returns the dmUser from storage.
func (us *userStore) get(pubKey ed25519.PublicKey) (*dmUser, error) {
	elemName := marshalElementName(pubKey)
	us.mux.Lock()
	obj, err := us.remote.GetMapElement(dmMapName, elemName, dmMapVersion)
	us.mux.Unlock()
	if err != nil {
		return nil, err
	}

	var user dmUser
	if err = json.Unmarshal(obj.Data, &user); err != nil {
		return nil, err
	}
	user.PublicKey = pubKey

	return &user, nil
}

// delete removes the dmUser for the public key from storage.
func (us *userStore) delete(pubKey ed25519.PublicKey) error {
	elemName := marshalElementName(pubKey)
	us.mux.Lock()
	_, err := us.remote.DeleteMapElement(dmMapName, elemName, dmMapVersion)
	us.mux.Unlock()
	if err != nil {
		return err
	}

	return nil
}

// getAll returns a list of all users in storage.
func (us *userStore) getAll() ([]*dmUser, error) {
	var users []*dmUser
	init := func(n int) { users = make([]*dmUser, 0, n) }
	add := func(user *dmUser) { users = append(users, user) }
	return users, us.iterate(init, add)
}

// iterate loops through all users in storage, unmarshalls each one, and passes
// it into the add function. Before add is called, init is called with the total
// number of users storage.
func (us *userStore) iterate(init func(n int), add func(user *dmUser)) error {
	us.mux.Lock()
	userMap, err := us.remote.GetMap(dmMapName, dmMapVersion)
	us.mux.Unlock()
	if err != nil {
		return err
	}

	init(len(userMap))
	for elemName, elem := range userMap {
		var user dmUser
		if err = json.Unmarshal(elem.Data, &user); err != nil {
			jww.ERROR.Printf("[DM] Failed to parse user for element name %q: %+v",
				elemName, err)
			continue
		}

		user.PublicKey, err = unmarshalElementName(elemName)
		if err != nil {
			jww.ERROR.Printf("[DM] Failed to parse element name %s for user "+
				"%+v: %+v", elemName, elem, err)
			continue
		}

		add(&user)
	}

	return nil
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
