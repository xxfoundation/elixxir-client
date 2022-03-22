///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package groupStore

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/xx_network/primitives/id"
	"sort"
	"strings"
)

// Error messages.
const (
	idUnmarshalErr = "failed to unmarshal member ID: %+v"
	dhKeyDecodeErr = "failed to decode member DH key: %+v"
)

// DhKeyList is a map of users to their DH key.
type DhKeyList map[id.ID]*cyclic.Int

// GenerateDhKeyList generates the DH key between the user and all group
// members.
func GenerateDhKeyList(userID *id.ID, privKey *cyclic.Int,
	members group.Membership, grp *cyclic.Group) DhKeyList {
	dkl := make(DhKeyList, len(members)-1)

	for _, m := range members {
		// Skip the group.member for the current user
		if !userID.Cmp(m.ID) {
			dkl.Add(privKey, m, grp)
		}
	}

	return dkl
}

// Add generates DH key between the user and the group member. The
func (dkl DhKeyList) Add(privKey *cyclic.Int, m group.Member, grp *cyclic.Group) {
	dkl[*m.ID] = diffieHellman.GenerateSessionKey(privKey, m.DhKey, grp)
}

// DeepCopy returns a copy of the DhKeyList.
func (dkl DhKeyList) DeepCopy() DhKeyList {
	newDkl := make(DhKeyList, len(dkl))
	for uid, key := range dkl {
		newDkl[uid] = key.DeepCopy()
	}
	return newDkl
}

// Serialize serializes the DhKeyList and returns the byte slice.
func (dkl DhKeyList) Serialize() []byte {
	buff := bytes.NewBuffer(nil)

	for uid, key := range dkl {
		// Write ID
		buff.Write(uid.Marshal())

		// Write DH key length
		b := make([]byte, 8)
		keyBytes := key.BinaryEncode()
		binary.LittleEndian.PutUint64(b, uint64(len(keyBytes)))
		buff.Write(b)

		// Write DH key
		buff.Write(keyBytes)
	}

	return buff.Bytes()
}

// DeserializeDhKeyList deserializes the bytes into a DhKeyList.
func DeserializeDhKeyList(data []byte) (DhKeyList, error) {
	if len(data) == 0 {
		return nil, nil
	}

	buff := bytes.NewBuffer(data)
	dkl := make(DhKeyList)

	const idLen = id.ArrIDLen
	for n := buff.Next(idLen); len(n) == idLen; n = buff.Next(idLen) {
		// Read and unmarshal ID
		uid, err := id.Unmarshal(n)
		if err != nil {
			return nil, errors.Errorf(idUnmarshalErr, err)
		}

		// get length of DH key
		keyLen := int(binary.LittleEndian.Uint64(buff.Next(8)))

		// Read and decode DH key
		key := &cyclic.Int{}
		err = key.BinaryDecode(buff.Next(keyLen))
		if err != nil {
			return nil, errors.Errorf(dhKeyDecodeErr, err)
		}

		dkl[*uid] = key
	}

	return dkl, nil
}

// GoString returns all the elements in the DhKeyList as text in sorted order.
// This functions satisfies the fmt.GoStringer interface.
func (dkl DhKeyList) GoString() string {
	str := make([]string, 0, len(dkl))

	unsorted := make([]struct {
		uid *id.ID
		key *cyclic.Int
	}, 0, len(dkl))

	for uid, key := range dkl {
		unsorted = append(unsorted, struct {
			uid *id.ID
			key *cyclic.Int
		}{uid: uid.DeepCopy(), key: key.DeepCopy()})
	}

	sort.Slice(unsorted, func(i, j int) bool {
		return bytes.Compare(unsorted[i].uid.Bytes(),
			unsorted[j].uid.Bytes()) == -1
	})

	for _, val := range unsorted {
		str = append(str, val.uid.String()+": "+val.key.Text(10))
	}

	return "{" + strings.Join(str, ", ") + "}"
}
