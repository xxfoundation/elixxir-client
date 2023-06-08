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

// partnerStore contains the DM remote KV storage
type partnerStore struct {
	remote versioned.KV
	mux    sync.Mutex
}

// newPartnerStore initialises a new partnerStore.
func newPartnerStore(kv versioned.KV) (*partnerStore, error) {
	remote, err := kv.Prefix(collective.StandardRemoteSyncPrefix)
	if err != nil {
		return nil, err
	}

	err = remote.ListenOnRemoteMap(dmMapName, dmMapVersion, nil, false)
	if err != nil && remote.Exists(err) {
		return nil, errors.Wrap(err, "failed to load and listen to remote "+
			"updates on DM partner storage")
	}

	return &partnerStore{remote: kv}, nil
}

// dmPartner stores information for each partner the current user has a DM
// conversation with.
type dmPartner struct {
	// PublicKey is the partner's public key. It is not included in JSON.
	PublicKey ed25519.PublicKey `json:"-"`

	// Status indicates the notification status for the partner or if they are
	// blocked.
	Status partnerStatus `json:"s"`
}

// String prints the dmPartner in a human-readable form for logging and
// debugging. This function adheres to the fmt.Stringer interface.
func (dmu *dmPartner) String() string {
	fields := []string{
		hex.EncodeToString(dmu.PublicKey),
		strconv.Itoa(int(dmu.Status)),
	}
	return "{" + strings.Join(fields, " ") + "}"
}

// partnerStatus represents the notification status or blocked status of the DM
// partner.
type partnerStatus uint32

const (
	statusMute      partnerStatus = 10
	statusNotifyAll partnerStatus = 20
	statusBlocked   partnerStatus = 50

	// defaultStatus is set when adding a new partner or resetting a status.
	defaultStatus = statusNotifyAll
)

// set saves the dmPartner info to storage keyed on the Ed25519 public key.
func (ps *partnerStore) set(pubKey ed25519.PublicKey, status partnerStatus) {
	ps.mux.Lock()
	defer ps.mux.Unlock()
	ps.setUnsafe(pubKey, status)
}

func (ps *partnerStore) setUnsafe(
	pubKey ed25519.PublicKey, status partnerStatus) {
	elemName := marshalElementName(pubKey)
	data, err := json.Marshal(dmPartner{
		Status: status,
	})
	if err != nil {
		jww.FATAL.Panicf("[DM] Failed to JSON marshal partner %X for storage: %+v",
			pubKey, err)
	}

	obj := &versioned.Object{
		Version:   dmStoreVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	err = ps.remote.StoreMapElement(dmMapName, elemName, obj, dmMapVersion)
	if err != nil {
		jww.FATAL.Panicf("[DM] Failed to set partner %X: %+v", pubKey, err)
	}
}

// get returns the dmPartner from storage. Returns false if the partner does not
// exist.
func (ps *partnerStore) get(
	pubKey ed25519.PublicKey) (partner *dmPartner, exists bool) {
	elemName := marshalElementName(pubKey)
	ps.mux.Lock()
	obj, err := ps.remote.GetMapElement(dmMapName, elemName, dmMapVersion)
	ps.mux.Unlock()
	if err != nil {
		if ps.remote.Exists(err) {
			jww.FATAL.Panicf("[DM] Failed to load partner %X from storage: %+v",
				pubKey, err)
		} else {
			return nil, false
		}
	}

	partner = &dmPartner{}
	if err = json.Unmarshal(obj.Data, partner); err != nil {
		jww.FATAL.Panicf("[DM] Failed to JSON unmarshal partner %X from "+
			"storage: %+v", pubKey, err)
	}
	partner.PublicKey = pubKey

	return partner, true
}

// getOrSet returns the dmPartner from storage. If the partner does not exist,
// then it is added with the default status and returned.
// TODO: test
func (ps *partnerStore) getOrSet(pubKey ed25519.PublicKey) *dmPartner {
	elemName := marshalElementName(pubKey)
	ps.mux.Lock()
	defer ps.mux.Unlock()
	obj, err := ps.remote.GetMapElement(dmMapName, elemName, dmMapVersion)
	if err != nil {
		if ps.remote.Exists(err) {
			jww.FATAL.Panicf("[DM] Failed to load partner %X from storage: %+v",
				pubKey, err)
		}

		ps.setUnsafe(pubKey, defaultStatus)
		return &dmPartner{
			PublicKey: pubKey,
			Status:    defaultStatus,
		}
	}

	var partner dmPartner
	if err = json.Unmarshal(obj.Data, &partner); err != nil {
		jww.FATAL.Panicf("[DM] Failed to JSON unmarshal partner %X from "+
			"storage: %+v", pubKey, err)
	}
	partner.PublicKey = pubKey

	return &partner
}

// delete removes the dmPartner for the public key from storage.
func (ps *partnerStore) delete(pubKey ed25519.PublicKey) {
	elemName := marshalElementName(pubKey)
	ps.mux.Lock()
	_, err := ps.remote.DeleteMapElement(dmMapName, elemName, dmMapVersion)
	ps.mux.Unlock()
	if err != nil {
		jww.FATAL.Panicf("[DM] Failed to delete partner %X from storage: %+v",
			pubKey, err)
	}
}

// getAll returns a list of all partner in storage.
func (ps *partnerStore) getAll() []*dmPartner {
	var partners []*dmPartner
	ps.iterate(func(n int) { partners = make([]*dmPartner, 0, n) },
		func(partner *dmPartner) { partners = append(partners, partner) })
	return partners
}

// iterate loops through all partners in storage, unmarshalls each one, and
// passes it into the add function. Before add is called, init is called with
// the total number of partners storage. Init can be nil.
func (ps *partnerStore) iterate(
	init func(n int), add func(partner *dmPartner)) {
	ps.mux.Lock()
	partnerMap, err := ps.remote.GetMap(dmMapName, dmMapVersion)
	ps.mux.Unlock()
	if err != nil {
		jww.FATAL.Panicf("[DM] Failed to load map %s from storage: %+v",
			dmMapName, err)
	}

	if init != nil {
		init(len(partnerMap))
	}
	for elemName, obj := range partnerMap {
		var partner dmPartner
		partner.PublicKey, err = unmarshalElementName(elemName)
		if err != nil {
			jww.ERROR.Printf("[DM] Failed to parse element name %s: %+v",
				elemName, err)
			continue
		}
		if err = json.Unmarshal(obj.Data, &partner); err != nil {
			jww.ERROR.Printf("[DM] Failed to parse partner for element name "+
				"%q: %+v", elemName, err)
			continue
		}

		add(&partner)
	}
}

// elementEdit describes a single edit in the partnerStore KV storage.
type elementEdit struct {
	old       *dmPartner
	new       *dmPartner
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
func (ps *partnerStore) listen(cb func(edits []elementEdit)) error {
	ps.mux.Lock()
	defer ps.mux.Unlock()
	return ps.remote.ListenOnRemoteMap(dmMapName, dmMapVersion,
		func(edits map[string]versioned.ElementEdit) {
			partnerEdits := make([]elementEdit, 0, len(edits))
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
						jww.ERROR.Printf("[DM] Failed to parse old partner for "+
							"element name %q: %+v", elemName, err)
						continue
					} else {
						e.old.PublicKey = pubKey
					}
				}

				if len(edit.NewElement.Data) > 0 {
					err = json.Unmarshal(edit.NewElement.Data, &e.new)
					if err != nil {
						jww.ERROR.Printf("[DM] Failed to parse new partner for "+
							"element name %q: %+v", elemName, err)
						continue
					} else {
						e.new.PublicKey = pubKey
					}
				}

				partnerEdits = append(partnerEdits, e)
			}
			cb(partnerEdits)
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
