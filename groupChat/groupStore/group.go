////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package groupStore

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

// Storage values.
const (
	// Key that is prepended to group ID to create a unique key to identify a
	// Group in storage.
	groupStorageKey   = "GroupChat/"
	groupStoreVersion = 0
)

// Error messages.
const (
	kvGetGroupErr = "failed to get group %s from storage: %+v"
	membershipErr = "failed to deserialize member list: %+v"
	dhKeyListErr  = "failed to deserialize DH key list: %+v"
)

// Group contains the membership list, the cryptographic information, and the
// identifying information of a group chat.
type Group struct {
	Name        []byte            // Name of the group set by the user
	ID          *id.ID            // Group ID
	Key         group.Key         // Group key
	IdPreimage  group.IdPreimage  // 256-bit randomly generated value
	KeyPreimage group.KeyPreimage // 256-bit randomly generated value
	InitMessage []byte            // The original invite message
	Created     time.Time         // Timestamp of when the group was created
	Members     group.Membership  // Sorted list of members in group
	DhKeys      DhKeyList         // List of shared DH keys
}

// NewGroup creates a new Group from copies of the given data.
func NewGroup(name []byte, groupID *id.ID, groupKey group.Key,
	idPreimage group.IdPreimage, keyPreimage group.KeyPreimage,
	initMessage []byte, created time.Time, members group.Membership,
	dhKeys DhKeyList) Group {
	g := Group{
		Name:        make([]byte, len(name)),
		ID:          groupID.DeepCopy(),
		Key:         groupKey,
		IdPreimage:  idPreimage,
		KeyPreimage: keyPreimage,
		InitMessage: make([]byte, len(initMessage)),
		Created:     created.Round(0),
		Members:     members.DeepCopy(),
		DhKeys:      dhKeys,
	}

	copy(g.Name, name)
	copy(g.InitMessage, initMessage)

	return g
}

// DeepCopy returns a copy of the Group.
func (g Group) DeepCopy() Group {
	newGrp := Group{
		Name:        make([]byte, len(g.Name)),
		ID:          g.ID.DeepCopy(),
		Key:         g.Key,
		IdPreimage:  g.IdPreimage,
		KeyPreimage: g.KeyPreimage,
		InitMessage: make([]byte, len(g.InitMessage)),
		Created:     g.Created,
		Members:     g.Members.DeepCopy(),
		DhKeys:      make(map[id.ID]*cyclic.Int, len(g.Members)-1),
	}

	copy(newGrp.Name, g.Name)
	copy(newGrp.InitMessage, g.InitMessage)

	for uid, key := range g.DhKeys {
		newGrp.DhKeys[uid] = key.DeepCopy()
	}

	return newGrp
}

// store saves an individual Group to storage keying on the group ID.
func (g Group) store(kv versioned.KV) error {
	obj := &versioned.Object{
		Version:   groupStoreVersion,
		Timestamp: netTime.Now(),
		Data:      g.Serialize(),
	}

	return kv.Set(groupStoreKey(g.ID), obj)
}

// loadGroup returns the group with the corresponding ID from storage.
func loadGroup(groupID *id.ID, kv versioned.KV) (Group, error) {
	obj, err := kv.Get(groupStoreKey(groupID), groupStoreVersion)
	if err != nil {
		return Group{}, errors.Errorf(kvGetGroupErr, groupID, err)
	}

	return DeserializeGroup(obj.Data)
}

// removeGroup deletes the given group from storage.
func removeGroup(groupID *id.ID, kv versioned.KV) error {
	return kv.Delete(groupStoreKey(groupID), groupStoreVersion)
}

// Serialize serializes the Group and returns the byte slice. The serialized
// data follows the following format.
// +----------+----------+----------+----------+------------+-------------+-----------------+-------------+---------+-------------+----------+----------+
// | Name len |   Name   |    ID    |    Key   | IdPreimage | KeyPreimage | InitMessage len | InitMessage | Created | Members len | Members  |  DhKeys  |
// | 8 bytes  | variable | 33 bytes | 32 bytes |  32 bytes  |  32 bytes   |     8 bytes     |  variable   | 8 bytes |   8 bytes   | variable | variable |
// +----------+----------+----------+----------+------------+-------------+-----------------+-------------+---------+-------------+----------+----------+
func (g Group) Serialize() []byte {
	buff := bytes.NewBuffer(nil)

	// Write length of name and name
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(len(g.Name)))
	buff.Write(b)
	buff.Write(g.Name)

	// Write group ID
	if g.ID != nil {
		buff.Write(g.ID.Marshal())
	} else {
		buff.Write(make([]byte, id.ArrIDLen))
	}

	// Write group key and preimages
	buff.Write(g.Key[:])
	buff.Write(g.IdPreimage[:])
	buff.Write(g.KeyPreimage[:])

	// Write length of InitMessage and InitMessage
	b = make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(len(g.InitMessage)))
	buff.Write(b)
	buff.Write(g.InitMessage)

	// Write created timestamp as Unix nanoseconds
	b = make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(g.Created.UnixNano()))
	buff.Write(b)

	// Write length of group membership and group membership
	b = make([]byte, 8)
	memberBytes := g.Members.Serialize()
	binary.LittleEndian.PutUint64(b, uint64(len(memberBytes)))
	buff.Write(b)
	buff.Write(memberBytes)

	// Write DH key list
	buff.Write(g.DhKeys.Serialize())

	return buff.Bytes()
}

// DeserializeGroup deserializes the bytes into a Group.
func DeserializeGroup(data []byte) (Group, error) {
	buff := bytes.NewBuffer(data)
	var g Group
	var err error

	// get name
	nameLen := binary.LittleEndian.Uint64(buff.Next(8))
	if nameLen > 0 {
		g.Name = buff.Next(int(nameLen))
	}

	// get group ID
	var groupID id.ID
	copy(groupID[:], buff.Next(id.ArrIDLen))
	if groupID == [id.ArrIDLen]byte{} {
		g.ID = nil
	} else {
		g.ID = &groupID
	}

	// get group key and preimages
	copy(g.Key[:], buff.Next(group.KeyLen))
	copy(g.IdPreimage[:], buff.Next(group.IdPreimageLen))
	copy(g.KeyPreimage[:], buff.Next(group.KeyPreimageLen))

	// get InitMessage
	initMessageLength := binary.LittleEndian.Uint64(buff.Next(8))
	if initMessageLength > 0 {
		g.InitMessage = buff.Next(int(initMessageLength))
	}

	// get created timestamp
	createdNano := int64(binary.LittleEndian.Uint64(buff.Next(8)))
	if createdNano == (time.Time{}).UnixNano() {
		g.Created = time.Time{}
	} else {
		g.Created = time.Unix(0, createdNano)
	}

	// get member list
	membersLength := binary.LittleEndian.Uint64(buff.Next(8))
	g.Members, err = group.DeserializeMembership(buff.Next(int(membersLength)))
	if err != nil {
		return Group{}, errors.Errorf(membershipErr, err)
	}

	// get DH key list
	g.DhKeys, err = DeserializeDhKeyList(buff.Bytes())
	if err != nil {
		return Group{}, errors.Errorf(dhKeyListErr, err)
	}

	return g, err
}

// groupStoreKey generates a unique key to save and load a Group to/from
// storage.
func groupStoreKey(groupID *id.ID) string {
	return groupStorageKey + groupID.String()
}

// GoString returns all the Group's fields as text. This functions satisfies the
// fmt.GoStringer interface.
func (g Group) GoString() string {
	idString := "<nil>"
	if g.ID != nil {
		idString = g.ID.String()
	}

	str := []string{
		"Name:" + fmt.Sprintf("%q", g.Name),
		"ID:" + idString,
		"Key:" + g.Key.String(),
		"IdPreimage:" + g.IdPreimage.String(),
		"KeyPreimage:" + g.KeyPreimage.String(),
		"InitMessage:" + fmt.Sprintf("%q", g.InitMessage),
		"Created:" + g.Created.String(),
		"Members:" + g.Members.String(),
		"DhKeys:" + g.DhKeys.GoString(),
	}

	return "{" + strings.Join(str, ", ") + "}"
}
