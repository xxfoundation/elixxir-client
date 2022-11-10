////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package clientVersion

import (
	"gitlab.com/elixxir/client/v5/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/version"
	"gitlab.com/xx_network/primitives/netTime"
	"reflect"
	"strings"
	"testing"
)

// Happy path.
func TestNewStore(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	expected := &Store{
		version: version.New(42, 43, "44"),
		kv:      kv.Prefix(prefix),
	}

	test, err := NewStore(expected.version, kv)
	if err != nil {
		t.Errorf("NewStore() returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expected, test) {
		t.Errorf("NewStore() failed to return the expected Store."+
			"\nexpected: %+v\nreceived: %+v", expected, test)
	}
}

// Happy path.
func TestLoadStore(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	ver := version.New(1, 2, "3A")

	expected := &Store{
		version: ver,
		kv:      kv.Prefix(prefix),
	}
	err := expected.save()
	if err != nil {
		t.Fatalf("Failed to save Store: %+v", err)
	}

	test, err := LoadStore(kv)
	if err != nil {
		t.Errorf("LoadStore() returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expected, test) {
		t.Errorf("LoadStore() failed to return the expected Store."+
			"\nexpected: %+v\nreceived: %+v", expected, test)
	}
}

// Error path: an error is returned when the loaded Store has an invalid version
// that fails to be parsed.
func TestLoadStore_ParseVersionError(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	obj := versioned.Object{
		Version:   storeVersion,
		Timestamp: netTime.Now(),
		Data:      []byte("invalid version"),
	}

	err := kv.Prefix(prefix).Set(storeKey, &obj)
	if err != nil {
		t.Fatalf("Failed to save Store: %+v", err)
	}

	_, err = LoadStore(kv)
	if err == nil || !strings.Contains(err.Error(), "failed to parse client version") {
		t.Errorf("LoadStore() did not return an error when the client version "+
			"is invalid: %+v", err)
	}
}

// Happy path.
func TestStore_Get(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	expected := version.New(1, 2, "3A")

	s := &Store{
		version: expected,
		kv:      kv.Prefix(prefix),
	}

	test := s.Get()
	if !reflect.DeepEqual(expected, test) {
		t.Errorf("get() failed to return the expected version."+
			"\nexpected: %s\nreceived: %s", &expected, &test)
	}
}

// Happy path.
func TestStore_CheckUpdateRequired(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	storedVersion := version.New(1, 2, "3")
	newVersion := version.New(2, 3, "4")
	s, err := NewStore(storedVersion, kv)
	if err != nil {
		t.Fatalf("Failed to generate a new Store: %+v", err)
	}

	updateRequired, oldVersion, err := s.CheckUpdateRequired(newVersion)
	if err != nil {
		t.Errorf("CheckUpdateRequired() returned an error: %+v", err)
	}

	if !updateRequired {
		t.Errorf("CheckUpdateRequired() did not indicate that an update is "+
			"required when the new Version (%s) is newer than the stored"+
			"version (%s)", &newVersion, &storedVersion)
	}

	if !version.Equal(storedVersion, oldVersion) {
		t.Errorf("CheckUpdateRequired() did return the expected old Version."+
			"\nexpected: %s\nreceived: %s", &storedVersion, &oldVersion)
	}
}

// Happy path: the new version is equal to the stored version.
func TestStore_CheckUpdateRequired_EqualVersions(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	storedVersion := version.New(2, 3, "3")
	newVersion := version.New(2, 3, "4")
	s, err := NewStore(storedVersion, kv)
	if err != nil {
		t.Fatalf("Failed to generate a new Store: %+v", err)
	}

	updateRequired, oldVersion, err := s.CheckUpdateRequired(newVersion)
	if err != nil {
		t.Errorf("CheckUpdateRequired() returned an error: %+v", err)
	}

	if updateRequired {
		t.Errorf("CheckUpdateRequired() did not indicate that an update is required "+
			"when the new Version (%s) is equal to the stored version (%s)",
			&newVersion, &storedVersion)
	}

	if !version.Equal(storedVersion, oldVersion) {
		t.Errorf("CheckUpdateRequired() did return the expected old Version."+
			"\nexpected: %s\nreceived: %s", &storedVersion, &oldVersion)
	}
}

// Error path: new version is older than stored version.
func TestStore_CheckUpdateRequired_NewVersionTooOldError(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	storedVersion := version.New(2, 3, "4")
	newVersion := version.New(1, 2, "3")
	s, err := NewStore(storedVersion, kv)
	if err != nil {
		t.Fatalf("Failed to generate a new Store: %+v", err)
	}

	updateRequired, oldVersion, err := s.CheckUpdateRequired(newVersion)
	if err == nil || !strings.Contains(err.Error(), "older than stored version") {
		t.Errorf("CheckUpdateRequired() did not return an error when the new version "+
			"is older than the stored version: %+v", err)
	}

	if updateRequired {
		t.Errorf("CheckUpdateRequired() indicated that an update is required when the "+
			"new Version (%s) is older than the stored version (%s)",
			&newVersion, &storedVersion)
	}

	if !version.Equal(storedVersion, oldVersion) {
		t.Errorf("CheckUpdateRequired() did return the expected old Version."+
			"\nexpected: %s\nreceived: %s", &storedVersion, &oldVersion)
	}
}

// Happy path.
func TestStore_update(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	ver1 := version.New(1, 2, "3A")
	ver2 := version.New(1, 5, "patch5")

	s := &Store{
		version: ver1,
		kv:      kv.Prefix(prefix),
	}

	err := s.update(ver2)
	if err != nil {
		t.Errorf("Update() returned an error: %+v", err)
	}

	if !reflect.DeepEqual(ver2, s.version) {
		t.Errorf("Update() did not set the correct version."+
			"\nexpected: %s\nreceived: %s", &ver2, &s.version)
	}
}

// Happy path.
func TestStore_save(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	ver := version.New(1, 2, "3A")

	s := &Store{
		version: ver,
		kv:      kv.Prefix(prefix),
	}

	err := s.save()
	if err != nil {
		t.Errorf("save() returned an error: %+v", err)
	}

	obj, err := s.kv.Get(storeKey, storeVersion)
	if err != nil {
		t.Errorf("Failed to load clientVersion store: %+v", err)
	}

	if ver.String() != string(obj.Data) {
		t.Errorf("Failed to get correct data from stored object."+
			"\nexpected: %s\nreceived: %s", ver.String(), obj.Data)
	}

	if storeVersion != obj.Version {
		t.Errorf("Failed to get correct version from stored object."+
			"\nexpected: %d\nreceived: %d", storeVersion, obj.Version)
	}

}
