package utility

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/knownRounds"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"testing"
)

// Tests happy path of NewCheckedRounds.
func TestNewCheckedRounds(t *testing.T) {
	// Set up expected value
	size := 10
	expectedCR := &CheckedRounds{
		rounds: knownRounds.NewKnownRound(size),
		kv:     versioned.NewKV(make(ekv.Memstore)),
		key:    "testKey",
	}

	// Create new CheckedRounds
	cr, err := NewCheckedRounds(expectedCR.kv, expectedCR.key, size)
	if err != nil {
		t.Errorf("NewCheckedRounds() returned an error."+
			"\n\texpected: %v\n\treceived: %v", nil, err)
	}

	if !reflect.DeepEqual(expectedCR, cr) {
		t.Errorf("NewCheckedRounds() returned an incorrect CheckedRounds."+
			"\n\texpected: %v\n\treceived: %v", expectedCR, cr)
	}
}

// Tests happy path of LoadCheckedRounds.
func TestLoadCheckedRounds(t *testing.T) {
	// Set up expected value
	size := 10
	expectedCR := &CheckedRounds{
		rounds: knownRounds.NewKnownRound(size),
		kv:     versioned.NewKV(make(ekv.Memstore)),
		key:    "testKey",
	}

	// Check rounds in the buffer and save the key value store
	for i := 0; i < (size * 64); i++ {
		if i%7 == 0 {
			expectedCR.rounds.Check(id.Round(i))
		}
	}
	err := expectedCR.save()
	if err != nil {
		t.Fatalf("Error saving CheckedRounds: %v", err)
	}

	cr, err := LoadCheckedRounds(expectedCR.kv, expectedCR.key, size)
	if err != nil {
		t.Errorf("LoadCheckedRounds() returned an error."+
			"\n\texpected: %v\n\treceived: %v", nil, err)
	}

	if !reflect.DeepEqual(expectedCR, cr) {
		t.Errorf("LoadCheckedRounds() returned an incorrect CheckedRounds."+
			"\n\texpected: %+v\n\treceived: %+v", expectedCR, cr)
	}
}

// Tests happy path of CheckedRounds.save().
func TestCheckedRounds_save(t *testing.T) {
	// Set up expected value
	size := 10
	expectedCR := &CheckedRounds{
		rounds: knownRounds.NewKnownRound(size),
		kv:     versioned.NewKV(make(ekv.Memstore)),
		key:    "testKey",
	}
	for i := 0; i < (size * 64); i++ {
		if i%7 == 0 {
			expectedCR.rounds.Check(id.Round(i))
		}
	}
	expectedData, err := expectedCR.rounds.Marshal()
	if err != nil {
		t.Fatalf("Marshal() returned an error: %v", err)
	}
	cr := &CheckedRounds{
		rounds: knownRounds.NewKnownRound(size),
		kv:     expectedCR.kv,
		key:    expectedCR.key,
	}

	err = expectedCR.save()
	if err != nil {
		t.Errorf("save() returned an error: %v", err)
	}

	obj, err := expectedCR.kv.Get(expectedCR.key)
	if err != nil {
		t.Errorf("Get() returned an error: %v", err)
	}

	if !reflect.DeepEqual(expectedData, obj.Data) {
		t.Errorf("save() did not save the correct CheckedRounds."+
			"\n\texpected: %+v\n\treceived: %+v", expectedData, cr)
	}
}

// Tests happy path of CheckedRounds.load().
func TestCheckedRounds_load(t *testing.T) {
	// Set up expected value
	size := 10
	expectedCR := &CheckedRounds{
		rounds: knownRounds.NewKnownRound(size),
		kv:     versioned.NewKV(make(ekv.Memstore)),
		key:    "testKey",
	}
	for i := 0; i < (size * 64); i++ {
		if i%7 == 0 {
			expectedCR.rounds.Check(id.Round(i))
		}
	}
	cr := &CheckedRounds{
		rounds: knownRounds.NewKnownRound(size),
		kv:     expectedCR.kv,
		key:    expectedCR.key,
	}

	err := expectedCR.save()
	if err != nil {
		t.Errorf("save() returned an error: %v", err)
	}

	err = cr.load()
	if err != nil {
		t.Errorf("load() returned an error: %v", err)
	}

	if !reflect.DeepEqual(expectedCR, cr) {
		t.Errorf("load() did not produce the correct CheckedRounds."+
			"\n\texpected: %+v\n\treceived: %+v", expectedCR, cr)
	}
}

func TestCheckedRounds_Smoke(t *testing.T) {
	cr, err := NewCheckedRounds(versioned.NewKV(make(ekv.Memstore)), "testKey", 10)
	if err != nil {
		t.Fatalf("Failed to create new CheckedRounds: %v", err)
	}

	if cr.Checked(10) {
		t.Errorf("Checked() on round ID %d did not return the expected value."+
			"\n\texpected: %v\n\treceived: %v", 10, false, cr.Checked(10))
	}

	cr.Check(10)

	if !cr.Checked(10) {
		t.Errorf("Checked() on round ID %d did not return the expected value."+
			"\n\texpected: %v\n\treceived: %v", 10, true, cr.Checked(10))
	}

	cr.Forward(20)

	if !cr.Checked(15) {
		t.Errorf("Checked() on round ID %d did not return the expected value."+
			"\n\texpected: %v\n\treceived: %v", 15, true, cr.Checked(15))
	}

	roundCheck := func(id id.Round) bool {
		return id%2 == 1
	}
	cr.RangeUnchecked(30, roundCheck)

	newCR := &CheckedRounds{
		rounds: knownRounds.NewKnownRound(10),
		kv:     cr.kv,
		key:    cr.key,
	}

	err = newCR.load()
	if err != nil {
		t.Errorf("load() returned an error: %v", err)
	}

	if !reflect.DeepEqual(cr, newCR) {
		t.Errorf("load() did not produce the correct CheckedRounds."+
			"\n\texpected: %+v\n\treceived: %+v", cr, newCR)
	}

	// TODO: Test CheckedRounds.RangeUncheckedMasked
}
