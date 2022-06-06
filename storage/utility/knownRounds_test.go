///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package utility

/*
// Tests happy path of NewKnownRounds.
func TestNewKnownRounds(t *testing.T) {
	// Set up expected value
	size := 10
	rootKv := versioned.NewKV(ekv.MakeMemstore())
	expectedKR := &KnownRounds{
		rounds: knownRounds.NewKnownRound(size),
		kv:     rootKv.Prefix(knownRoundsPrefix),
		key:    "testKey",
	}

	// Create new KnownRounds
	k := knownRounds.NewKnownRound(size)
	kr, err := NewKnownRounds(rootKv, expectedKR.key, k)
	if err != nil {
		t.Errorf("NewKnownRounds() returned an error."+
			"\n\texpected: %v\n\treceived: %v", nil, err)
	}

	if !reflect.DeepEqual(expectedKR, kr) {
		t.Errorf("NewKnownRounds() returned an incorrect KnownRounds."+
			"\n\texpected: %+v\n\treceived: %+v", expectedKR, kr)
	}
}

// Tests happy path of LoadKnownRounds.
func TestLoadKnownRounds(t *testing.T) {
	// Set up expected value
	size := 10
	rootKv := versioned.NewKV(ekv.MakeMemstore())
	expectedKR := &KnownRounds{
		rounds: knownRounds.NewKnownRound(size),
		kv:     rootKv.Prefix(knownRoundsPrefix),
		key:    "testKey",
	}

	// Check rounds in the buffer and save the key value store
	expectedKR.rounds.Check(id.Round(0))
	for i := 0; i < (size * 64); i++ {
		if i%7 == 0 {
			expectedKR.rounds.Check(id.Round(i))
		}
	}
	err := expectedKR.save()
	if err != nil {
		t.Fatalf("Error saving KnownRounds: %v", err)
	}

	kr, err := LoadKnownRounds(rootKv, expectedKR.key, size)
	if err != nil {
		t.Errorf("LoadKnownRounds() returned an error."+
			"\n\texpected: %v\n\treceived: %+v", nil, err)
	}

	if !reflect.DeepEqual(expectedKR, kr) {
		t.Errorf("LoadKnownRounds() returned an incorrect KnownRounds."+
			"\n\texpected: %+v\n\treceived: %+v", expectedKR, kr)
		t.Errorf("%+v != \n%+v",
			expectedKR.rounds, kr.rounds)
	}
}

// Tests happy path of KnownRounds.save().
func TestKnownRounds_save(t *testing.T) {
	// Set up expected value
	size := 10
	expectedKR := &KnownRounds{
		rounds: knownRounds.NewKnownRound(size),
		kv:     versioned.NewKV(ekv.MakeMemstore()),
		key:    "testKey",
	}
	for i := 0; i < (size * 64); i++ {
		if i%7 == 0 {
			expectedKR.rounds.Check(id.Round(i))
		}
	}
	expectedData, err := expectedKR.rounds.Marshal()
	if err != nil {
		t.Fatalf("Marshal() returned an error: %v", err)
	}
	kr := &KnownRounds{
		rounds: knownRounds.NewKnownRound(size),
		kv:     expectedKR.kv,
		key:    expectedKR.key,
	}

	err = expectedKR.save()
	if err != nil {
		t.Errorf("save() returned an error: %v", err)
	}

	obj, err := expectedKR.kv.get(expectedKR.key)
	if err != nil {
		t.Errorf("get() returned an error: %v", err)
	}

	if !reflect.DeepEqual(expectedData, obj.Data) {
		t.Errorf("save() did not save the correct KnownRounds."+
			"\n\texpected: %+v\n\treceived: %+v", expectedData, kr)
	}
}

// // Tests happy path of KnownRounds.load().
// func TestKnownRounds_load(t *testing.T) {
// 	// Set up expected value
// 	size := 10
// 	expectedKR := &KnownRounds{
// 		rounds: knownRounds.NewKnownRound(size),
// 		kv:     versioned.NewKV(ekv.MakeMemstore()),
// 		key:    "testKey",
// 	}
// 	for i := 0; i < (size * 64); i++ {
// 		if i%7 == 0 {
// 			expectedKR.rounds.Check(id.Round(i))
// 		}
// 	}
// 	kr := &KnownRounds{
// 		rounds: knownRounds.NewKnownRound(size * 64),
// 		kv:     expectedKR.kv,
// 		key:    expectedKR.key,
// 	}

// 	err := expectedKR.save()
// 	if err != nil {
// 		t.Errorf("save() returned an error: %v", err)
// 	}

// 	err = kr.load()
// 	if err != nil {
// 		t.Errorf("load() returned an error: %v", err)
// 	}

// 	if !reflect.DeepEqual(expectedKR, kr) {
// 		t.Errorf("load() did not produce the correct KnownRounds."+
// 			"\n\texpected: %+v\n\treceived: %+v", expectedKR, kr)
// 	}
// }

func TestKnownRounds_Smoke(t *testing.T) {
	k := knownRounds.NewKnownRound(10)
	kr, err := NewKnownRounds(versioned.NewKV(ekv.MakeMemstore()), "testKey", k)
	if err != nil {
		t.Fatalf("Failed to create new KnownRounds: %v", err)
	}

	if kr.Checked(10) {
		t.Errorf("Checked() on round ID %d did not return the expected value."+
			"\n\texpected: %v\n\treceived: %v", 10, false, kr.Checked(10))
	}

	kr.Check(10)

	if !kr.Checked(10) {
		t.Errorf("Checked() on round ID %d did not return the expected value."+
			"\n\texpected: %v\n\treceived: %v", 10, true, kr.Checked(10))
	}

	roundCheck := func(id id.Round) bool {
		return id%2 == 1
	}
	kr.RangeUnchecked(30, roundCheck)

	newKR := &KnownRounds{
		rounds: knownRounds.NewKnownRound(10),
		kv:     kr.kv,
		key:    kr.key,
	}

	err = newKR.load()
	if err != nil {
		t.Errorf("load() returned an error: %v", err)
	}

	if !reflect.DeepEqual(kr, newKR) {
		t.Errorf("load() did not produce the correct KnownRounds."+
			"\n\texpected: %+v\n\treceived: %+v", kr, newKR)
	}

	mask := knownRounds.NewKnownRound(1)
	mask.Check(17)
	kr.RangeUncheckedMasked(mask, roundCheck, 15)

	kr.Forward(20)

	if !kr.Checked(15) {
		t.Errorf("Checked() on round ID %d did not return the expected value."+
			"\n\texpected: %v\n\treceived: %v", 15, true, kr.Checked(15))
	}
}*/
