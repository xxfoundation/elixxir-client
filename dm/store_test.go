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

// Unit test of partnerStore.set.
func Test_partnerStore_set(t *testing.T) {
	kv := collective.TestingKV(t, ekv.MakeMemstore(),
		collective.StandardPrefexs, collective.NewMockRemote())
	ps, err := newPartnerStore(kv)
	if err != nil {
		t.Fatal(err)
	}

	prng := rand.New(rand.NewSource(6784))
	const numPartners = 25
	expected := make(map[string]*dmPartner, numPartners)
	statuses := []partnerStatus{statusMute, statusNotifyAll, statusBlocked}
	for i := 0; i < numPartners; i++ {
		pubKey, _, _ := ed25519.GenerateKey(prng)
		partner := dmPartner{nil, statuses[i%len(statuses)]}
		elemName := marshalElementName(pubKey)
		expected[elemName] = &partner
		ps.set(pubKey, partner.Status)
	}

	for elemName, exp := range expected {
		var partner dmPartner
		obj, err := kv.GetMapElement(dmMapName, elemName, dmStoreVersion)
		if err != nil {
			t.Errorf("Failed to get dmPartner %s: %+v", elemName, err)
		} else if err = json.Unmarshal(obj.Data, &partner); err != nil {
			t.Errorf("Failed to JSON unmarshal dmPartner %s: %+v", elemName, err)
		} else if !reflect.DeepEqual(exp, &partner) {
			t.Errorf("Loaded unexpected dmPartner %s.\nexpected: %v\nreceived: %v",
				elemName, exp, &partner)
		}
	}
}

// Unit test of partnerStore.get.
func Test_partnerStore_get(t *testing.T) {
	ps, expected, _, _ := newFilledPartnerStore(25, 3694, t)

	for elemName, exp := range expected {
		partner, exist := ps.get(exp.PublicKey)
		if !exist {
			t.Errorf("Partner %s does not exist", elemName)
		} else if !reflect.DeepEqual(exp, partner) {
			t.Errorf("Loaded unexpected dmPartner %s.\nexpected: %v\nreceived: %v",
				elemName, exp, partner)
		}
	}
}

// Unit test of partnerStore.delete.
func Test_partnerStore_delete(t *testing.T) {
	ps, expected, _, _ := newFilledPartnerStore(25, 98957, t)

	for elemName, exp := range expected {
		ps.delete(exp.PublicKey)

		_, exists := ps.get(exp.PublicKey)
		if exists {
			t.Errorf("Partner %s not deleted.", elemName)
		}
	}
}

// Unit test of partnerStore.getAll.
func Test_partnerStore_getAll(t *testing.T) {
	ps, _, expected, _ := newFilledPartnerStore(25, 52889, t)

	partners := ps.getAll()

	sort.SliceStable(expected, func(i, j int) bool {
		return bytes.Compare(expected[i].PublicKey, expected[j].PublicKey) == -1
	})
	sort.SliceStable(partners, func(i, j int) bool {
		return bytes.Compare(partners[i].PublicKey, partners[j].PublicKey) == -1
	})

	if !reflect.DeepEqual(expected, partners) {
		t.Errorf("List of all partners does not match expected."+
			"\nexpected: %v\nreceived: %v", expected, partners)
	}
}

// Tests that partnerStore.iterate calls init with the correct size and that add
// is called for all stored partners.
func Test_partnerStore_iterate(t *testing.T) {
	ps, _, expected, _ := newFilledPartnerStore(25, 33482, t)

	var partners []*dmPartner
	init := func(n int) { partners = make([]*dmPartner, 0, n) }
	add := func(partner *dmPartner) { partners = append(partners, partner) }

	ps.iterate(init, add)

	sort.SliceStable(expected, func(i, j int) bool {
		return bytes.Compare(expected[i].PublicKey, expected[j].PublicKey) == -1
	})
	sort.SliceStable(partners, func(i, j int) bool {
		return bytes.Compare(partners[i].PublicKey, partners[j].PublicKey) == -1
	})

	if !reflect.DeepEqual(expected, partners) {
		t.Errorf("List of all partners does not match expected."+
			"\nexpected: %v\nreceived: %v", expected, partners)
	}
}

// Tests that partnerStore.iterate calls init with the correct size and that add
// is called for all stored partners.
func Test_partnerStore_listen(t *testing.T) {

	kv := collective.TestingKV(t, ekv.MakeMemstore(),
		collective.StandardPrefexs, collective.NewMockRemote())
	prng := rand.New(rand.NewSource(33482))
	ps, err := newPartnerStore(kv)
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
				new:       &dmPartner{pubKey1, statusMute},
				operation: versioned.Loaded,
			}, {
			old:       nil,
			new:       &dmPartner{pubKey2, statusNotifyAll},
			operation: versioned.Loaded,
		}, {
			old:       nil,
			new:       &dmPartner{pubKey3, statusBlocked},
			operation: versioned.Loaded,
		},
		}, {
			{
				old:       &dmPartner{pubKey1, statusMute},
				new:       &dmPartner{pubKey1, statusNotifyAll},
				operation: versioned.Updated,
			},
		}, {
			{
				old:       &dmPartner{pubKey2, statusNotifyAll},
				new:       nil,
				operation: versioned.Deleted,
			},
		}, {
			{
				old:       nil,
				new:       &dmPartner{pubKey4, statusNotifyAll},
				operation: versioned.Created,
			},
		},
	}

	for _, edit := range expectedEdits[0] {
		ps.set(edit.new.PublicKey, edit.new.Status)
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
		err = ps.listen(cb)
		if err != nil {
			t.Errorf("Failed to add listener: %+v", err)
		}
	}()

	testChan <- struct{}{}

	for _, edits := range expectedEdits[1:] {
		for _, edit := range edits {
			switch edit.operation {
			case versioned.Created:
				ps.set(edit.new.PublicKey, edit.new.Status)
			case versioned.Updated:
				ps.set(edit.new.PublicKey, edit.new.Status)
			case versioned.Deleted:
				ps.delete(edit.old.PublicKey)
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

// newFilledPartnerStore creates a new partnerStore and fills it with randomly
// generated partners.
func newFilledPartnerStore(numPartners int, seed int64, t testing.TB) (
	*partnerStore, map[string]*dmPartner, []*dmPartner, versioned.KV) {
	kv := collective.TestingKV(t, ekv.MakeMemstore(),
		collective.StandardPrefexs, collective.NewMockRemote())
	prng := rand.New(rand.NewSource(seed))
	ps, err := newPartnerStore(kv)
	if err != nil {
		t.Fatal(err)
	}
	partnerMap := make(map[string]*dmPartner, numPartners)
	partnerList := make([]*dmPartner, numPartners)
	statuses := []partnerStatus{statusMute, statusNotifyAll, statusBlocked}
	for i := 0; i < numPartners; i++ {
		pubKey, _, _ := ed25519.GenerateKey(prng)
		partner := dmPartner{pubKey, statuses[i%len(statuses)]}
		elemName := marshalElementName(pubKey)
		partnerMap[elemName] = &partner
		partnerList[i] = &partner
		ps.set(pubKey, partner.Status)
	}

	return ps, partnerMap, partnerList, kv
}
