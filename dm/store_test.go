////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"

	"gitlab.com/elixxir/client/v4/collective"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/ekv"
)

// Unit test of userStore.set.
func Test_userStore_set(t *testing.T) {
	kv := collective.TestingKV(t, ekv.MakeMemstore(),
		collective.StandardPrefexs, collective.NewMockRemote())
	us, err := newUserStore(kv)
	if err != nil {
		t.Fatal(err)
	}

	prng := rand.New(rand.NewSource(6784))
	const numUsers = 25
	expected := make(map[string]*dmUser, numUsers)
	statuses := []userStatus{statusMute, statusNotifyAll, statusBlocked}
	for i := 0; i < numUsers; i++ {
		pubKey, _, _ := ed25519.GenerateKey(prng)
		user := dmUser{nil, statuses[i%len(statuses)]}
		elemName := marshalElementName(pubKey)
		expected[elemName] = &user
		us.set(pubKey, user.Status)
	}

	for elemName, exp := range expected {
		var user dmUser
		obj, err := kv.GetMapElement(dmMapName, elemName, dmStoreVersion)
		if err != nil {
			t.Errorf("Failed to get dmUser %s: %+v", elemName, err)
		} else if err = json.Unmarshal(obj.Data, &user); err != nil {
			t.Errorf("Failed to JSON unmarshal dmUser %s: %+v", elemName, err)
		} else if !reflect.DeepEqual(exp, &user) {
			t.Errorf("Loaded unexpected dmUser %s.\nexpected: %v\nreceived: %v",
				elemName, exp, &user)
		}
	}
}

// Unit test of userStore.get.
func Test_userStore_get(t *testing.T) {
	us, expected, _, _ := newFilledUserStore(25, 3694, t)

	for elemName, exp := range expected {
		user, exist := us.get(exp.PublicKey)
		if !exist {
			t.Errorf("User %s does not exist", elemName)
		} else if !reflect.DeepEqual(exp, user) {
			t.Errorf("Loaded unexpected dmUser %s.\nexpected: %v\nreceived: %v",
				elemName, exp, user)
		}
	}
}

// Unit test of userStore.delete.
func Test_userStore_delete(t *testing.T) {
	us, expected, _, _ := newFilledUserStore(25, 98957, t)

	for elemName, exp := range expected {
		us.delete(exp.PublicKey)

		_, exists := us.get(exp.PublicKey)
		if exists {
			t.Errorf("User %s not deleted.", elemName)
		}
	}
}

// Unit test of userStore.getAll.
func Test_userStore_getAll(t *testing.T) {
	us, _, expected, _ := newFilledUserStore(25, 52889, t)

	users := us.getAll()

	sort.SliceStable(expected, func(i, j int) bool {
		return bytes.Compare(expected[i].PublicKey, expected[j].PublicKey) == -1
	})
	sort.SliceStable(users, func(i, j int) bool {
		return bytes.Compare(users[i].PublicKey, users[j].PublicKey) == -1
	})

	if !reflect.DeepEqual(expected, users) {
		t.Errorf("List of all users does not match expected."+
			"\nexpected: %v\nreceived: %v", expected, users)
	}
}

// Tests that userStore.iterate calls init with the correct size and that add is
// called for all stored users.
func Test_userStore_iterate(t *testing.T) {
	us, _, expected, _ := newFilledUserStore(25, 33482, t)

	var users []*dmUser
	init := func(n int) { users = make([]*dmUser, 0, n) }
	add := func(user *dmUser) { users = append(users, user) }

	us.iterate(init, add)

	sort.SliceStable(expected, func(i, j int) bool {
		return bytes.Compare(expected[i].PublicKey, expected[j].PublicKey) == -1
	})
	sort.SliceStable(users, func(i, j int) bool {
		return bytes.Compare(users[i].PublicKey, users[j].PublicKey) == -1
	})

	if !reflect.DeepEqual(expected, users) {
		t.Errorf("List of all users does not match expected."+
			"\nexpected: %v\nreceived: %v", expected, users)
	}
}

// Tests that userStore.iterate calls init with the correct size and that add is
// called for all stored users.
func Test_userStore_listen(t *testing.T) {

	kv := collective.TestingKV(t, ekv.MakeMemstore(),
		collective.StandardPrefexs, collective.NewMockRemote())
	prng := rand.New(rand.NewSource(33482))
	us, err := newUserStore(kv)
	if err != nil {
		t.Fatal(err)
	}

	pubKey1, _, _ := ed25519.GenerateKey(prng)
	pubKey2, _, _ := ed25519.GenerateKey(prng)
	pubKey3, _, _ := ed25519.GenerateKey(prng)
	pubKey4, _, _ := ed25519.GenerateKey(prng)

	expectedEdits := [][]elementEdit{
		{
			{
				old:       nil,
				new:       &dmUser{pubKey1, statusMute},
				operation: versioned.Loaded,
			}, {
				old:       nil,
				new:       &dmUser{pubKey2, statusNotifyAll},
				operation: versioned.Loaded,
			}, {
				old:       nil,
				new:       &dmUser{pubKey3, statusBlocked},
				operation: versioned.Loaded,
			},
		}, {
			{
				old:       &dmUser{pubKey1, statusMute},
				new:       &dmUser{pubKey1, statusNotifyAll},
				operation: versioned.Updated,
			},
		}, {
			{
				old:       &dmUser{pubKey2, statusNotifyAll},
				new:       nil,
				operation: versioned.Deleted,
			},
		}, {
			{
				old:       nil,
				new:       &dmUser{pubKey4, statusNotifyAll},
				operation: versioned.Created,
			},
		},
	}

	for _, edit := range expectedEdits[0] {
		us.set(edit.new.PublicKey, edit.new.Status)
	}

	var i int
	testChan := make(chan struct{})
	cb := func(edits []elementEdit) {
		sort.SliceStable(expectedEdits[i], func(x, y int) bool {
			xKey := fmt.Sprintf("%s%s", expectedEdits[i][x].old, expectedEdits[i][x].new)
			yKey := fmt.Sprintf("%s%s", expectedEdits[i][y].old, expectedEdits[i][y].new)
			return bytes.Compare([]byte(xKey), []byte(yKey)) == -1
		})
		sort.SliceStable(edits, func(x, y int) bool {
			xKey := fmt.Sprintf("%s%s", edits[x].old, edits[x].new)
			yKey := fmt.Sprintf("%s%s", edits[y].old, edits[y].new)
			return bytes.Compare([]byte(xKey), []byte(yKey)) == -1
		})

		if !reflect.DeepEqual(expectedEdits[i], edits) {
			t.Errorf("Unexpected edits (%d).\nexpected: %s\nreceived: %s",
				i, expectedEdits[i], edits)
		}
		i++
		<-testChan
	}
	go func() {
		err = us.listen(cb)
		if err != nil {
			t.Errorf("Failed to add listener: %+v", err)
		}
	}()

	testChan <- struct{}{}

	for _, edits := range expectedEdits[1:] {
		for _, edit := range edits {
			switch edit.operation {
			case versioned.Created:
				us.set(edit.new.PublicKey, edit.new.Status)
			case versioned.Updated:
				us.set(edit.new.PublicKey, edit.new.Status)
			case versioned.Deleted:
				us.delete(edit.old.PublicKey)
			}
		}
		testChan <- struct{}{}
	}
}

// Unit test of marshalElementName and unmarshalElementName.
func Test_marshalElementName_unmarshalElementName(t *testing.T) {
	prng := rand.New(rand.NewSource(84))

	for i := 0; i < 20; i++ {
		expected, _, _ := ed25519.GenerateKey(prng)

		elemName := marshalElementName(expected)
		pubkey, err := unmarshalElementName(elemName)
		if err != nil {
			t.Errorf("Failed to unmarshal element name for %X (%d): %+v",
				expected, i, err)
		} else if !reflect.DeepEqual(expected, pubkey) {
			t.Errorf("Unexpected pub key (%d).\nexpected: %X\nreceived: %x",
				i, expected, pubkey)
		}
	}

	a := [2]uint32{5}
	data, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}

	var b [2]uint32
	err = json.Unmarshal(data, &b)
	if err != nil {
		t.Fatal(err)
	}
}

// newFilledUserStore creates a new userStore and fills it with randomly
// generated users.
func newFilledUserStore(numUsers int, seed int64, t testing.TB) (
	*userStore, map[string]*dmUser, []*dmUser, versioned.KV) {
	kv := collective.TestingKV(t, ekv.MakeMemstore(),
		collective.StandardPrefexs, collective.NewMockRemote())
	prng := rand.New(rand.NewSource(seed))
	us, err := newUserStore(kv)
	if err != nil {
		t.Fatal(err)
	}
	userMap := make(map[string]*dmUser, numUsers)
	userList := make([]*dmUser, numUsers)
	statuses := []userStatus{statusMute, statusNotifyAll, statusBlocked}
	for i := 0; i < numUsers; i++ {
		pubKey, _, _ := ed25519.GenerateKey(prng)
		user := dmUser{pubKey, statuses[i%len(statuses)]}
		elemName := marshalElementName(pubKey)
		userMap[elemName] = &user
		userList[i] = &user
		us.set(pubKey, user.Status)
	}

	return us, userMap, userList, kv
}
