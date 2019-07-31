package keyStore

import (
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"testing"
	"time"
)

// Helper function to compare E2E Keys
func E2EKeyCmp(a, b *E2EKey) bool {
	if a.GetManager() != b.GetManager() {
		return false
	}
	if a.GetOuterType() != b.GetOuterType() {
		return false
	}
	if a.GetKey().Cmp(b.GetKey()) != 0 {
		return false
	}
	return true
}

// Test KeyStack creation and push/pop
func TestKeyStack(t *testing.T) {
	ks := NewKeyStack()
	grp := cyclic.NewGroup(large.NewInt(107),
		large.NewInt(2),
		large.NewInt(5))
	expectedKeys := make([]*E2EKey, 100)

	for i := 0; i < 100; i++ {
		key := new(E2EKey)
		key.outer = parse.E2E
		key.key = grp.NewInt(int64(i + 2))
		key.manager = nil
		expectedKeys[99-i] = key
		ks.Push(key)
	}

	for i := 0; i < 100; i++ {
		actual := ks.Pop()
		if !E2EKeyCmp(actual, expectedKeys[i]) {
			t.Errorf("Pop'd key doesn't match with expected")
		}
	}
}

// Test that KeyStack panics on pop if empty
func TestKeyStack_Panic(t *testing.T) {
	ks := NewKeyStack()
	grp := cyclic.NewGroup(large.NewInt(107),
		large.NewInt(2),
		large.NewInt(5))
	expectedKeys := make([]*E2EKey, 10)

	for i := 0; i < 10; i++ {
		key := new(E2EKey)
		key.outer = parse.E2E
		key.key = grp.NewInt(int64(i + 2))
		key.manager = nil
		expectedKeys[9-i] = key
		ks.Push(key)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Pop should panic when stack is empty")
		}
	}()

	for i := 0; i < 11; i++ {
		actual := ks.Pop()
		if !E2EKeyCmp(actual, expectedKeys[i]) {
			t.Errorf("Pop'd key doesn't match with expected")
		}
	}
}

// Test that delete correctly empties stack
func TestKeyStack_Delete(t *testing.T) {
	ks := NewKeyStack()
	grp := cyclic.NewGroup(large.NewInt(107),
		large.NewInt(2),
		large.NewInt(5))
	expectedKeys := make([]*E2EKey, 100)

	for i := 0; i < 100; i++ {
		key := new(E2EKey)
		key.outer = parse.E2E
		key.key = grp.NewInt(int64(i + 2))
		key.manager = nil
		expectedKeys[99-i] = key
		ks.Push(key)
	}

	for i := 0; i < 50; i++ {
		actual := ks.Pop()
		if !E2EKeyCmp(actual, expectedKeys[i]) {
			t.Errorf("Pop'd key doesn't match with expected")
		}
	}

	ks.Delete()

	k4 := ks.Pop()
	if k4 != nil {
		t.Errorf("Pop should return nil when stack is empty")
	}
}

// Test concurrent access
func TestKeyStack_Concurrent(t *testing.T) {
	ks := NewKeyStack()
	grp := cyclic.NewGroup(large.NewInt(107),
		large.NewInt(2),
		large.NewInt(5))
	expectedKeys := make([]*E2EKey, 100)

	for i := 0; i < 100; i++ {
		key := new(E2EKey)
		key.outer = parse.E2E
		key.key = grp.NewInt(int64(i + 2))
		key.manager = nil
		expectedKeys[99-i] = key
		ks.Push(key)
	}

	for i := 0; i < 100; i++ {
		go func() {
			ks.Pop()
		}()
	}

	// wait for goroutines
	time.Sleep(500 * time.Millisecond)

	k4 := ks.Pop()
	if k4 != nil {
		t.Errorf("Pop should return nil when stack is empty")
	}
}
