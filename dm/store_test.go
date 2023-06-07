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
	"gitlab.com/elixxir/client/v4/collective"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/ekv"
	"math/rand"
	"os"
	"reflect"
	"sort"
	"testing"
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
		user := dmUser{nil, statuses[i%len(statuses)], prng.Uint32()}
		elemName := marshalElementName(pubKey)
		expected[elemName] = &user
		err := us.set(pubKey, user.Status, user.Token)
		if err != nil {
			t.Errorf("Failed to set dmUser %s %v: %+v", elemName, user, err)
		}
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

// Unit test of userStore.update.
func Test_userStore_update(t *testing.T) {
	prng := rand.New(rand.NewSource(42889))
	us, expected, _, _ := newFilledUserStore(25, 3694, t)
	statuses := []userStatus{statusMute, statusNotifyAll, statusBlocked}

	for elemName, exp := range expected {
		exp.Status = statuses[prng.Intn(len(statuses))]
		if err := us.update(exp.PublicKey, exp.Status); err != nil {
			t.Errorf("Failed to update dmUser %s: %+v", elemName, err)
		} else if user, err := us.get(exp.PublicKey); err != nil {
			t.Errorf("Failed to get dmUser %s: %+v", elemName, err)
		} else if !reflect.DeepEqual(exp, user) {
			t.Errorf("Loaded unexpected dmUser %s.\nexpected: %v\nreceived: %v",
				elemName, exp, user)
		}
	}
}

// Unit test of userStore.get.
func Test_userStore_get(t *testing.T) {
	us, expected, _, _ := newFilledUserStore(25, 3694, t)

	for elemName, exp := range expected {
		user, err := us.get(exp.PublicKey)
		if err != nil {
			t.Errorf("Failed to get dmUser %s: %+v", elemName, err)
		} else if !reflect.DeepEqual(exp, user) {
			t.Errorf("Loaded unexpected dmUser %s.\nexpected: %v\nreceived: %v",
				elemName, exp, user)
		}
	}
}

// Unit test of userStore.delete.
func Test_userStore_delete(t *testing.T) {
	us, expected, _, kv := newFilledUserStore(25, 98957, t)

	for elemName, exp := range expected {
		err := us.delete(exp.PublicKey)
		if err != nil {
			t.Errorf("Error while deleting dmUser %s: %+v", elemName, err)
		}

		_, err = us.get(exp.PublicKey)
		if err == nil || kv.Exists(err) {
			t.Errorf("Unexpected error for user %s."+
				"\nexpected: %+v\nreceived: %+v", elemName, os.ErrNotExist, err)
		}
	}
}

// Unit test of userStore.getAll.
func Test_userStore_getAll(t *testing.T) {
	us, _, expected, _ := newFilledUserStore(25, 52889, t)

	users, err := us.getAll()
	if err != nil {
		t.Errorf("Failed to get all users: %+v", err)
	}

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

	err := us.iterate(init, add)
	if err != nil {
		t.Errorf("Failed to get all users: %+v", err)
	}

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

	t.Logf("%s", data)

	var b [2]uint32
	err = json.Unmarshal(data, &b)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v", b)
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
		user := dmUser{pubKey, statuses[i%len(statuses)], prng.Uint32()}
		elemName := marshalElementName(pubKey)
		userMap[elemName] = &user
		userList[i] = &user
		err := us.set(pubKey, user.Status, user.Token)
		if err != nil {
			t.Errorf("Failed to set dmUser %s %v: %+v", elemName, user, err)
		}
	}

	return us, userMap, userList, kv
}
