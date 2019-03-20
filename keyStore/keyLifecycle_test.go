package keyStore

import (
	"fmt"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

var kl *KeyLifecycle
var arrSize int
var privateKey *cyclic.Int
var partner *id.User
var sendKeys, sendReKeys []E2EKey
var ttl uint16

func init() {
	rand.Seed(42)

	// Generate KeyLifeCycle with random values
	privateKey = cyclic.NewInt(rand.Int63())
	partner = id.NewUserFromUint(rand.Uint64(), &(testing.T{}))
	kl = GenerateKeyLifecycle(privateKey, partner)

	// Generate two random arrays of E2EKeys
	arrSize = 20
	sendKeys = make([]E2EKey, arrSize)
	sendReKeys = make([]E2EKey, arrSize)

	for i := 0; i < arrSize; i++ {
		sendKeys[i] = E2EKey{kl, cyclic.NewInt(rand.Int63()), format.OuterType(rand.Intn(6))}
		sendReKeys[i] = E2EKey{kl, cyclic.NewInt(rand.Int63()), format.OuterType(rand.Intn(6))}
	}

	// Set time to live
	ttl = 19
}

// Tests that an empty KeyLifecycle is generated with all the correct initial
// values.
func TestGenerateKeyLifecycle(t *testing.T) {
	if kl.privateKey != privateKey {
		t.Errorf("GenerateKeyLifecycle() did not set the correct privateKey\n\treceived: %v\n\texpected: %v", kl.privateKey, privateKey)
	}

	if kl.partner != partner {
		t.Errorf("GenerateKeyLifecycle() did not set the correct partner\n\treceived: %v\n\texpected: %v", kl.partner, partner)
	}

	if kl.count != 0 {
		t.Errorf("GenerateKeyLifecycle() did not set the count to zero\n\treceived: %v\n\texpected: %v", kl.count, 0)
	}

	if kl.state != KEYING {
		t.Errorf("GenerateKeyLifecycle() did not set the state to KEYING\n\treceived: %v\n\texpected: %v", kl.state, KEYING)
	}

	if kl.sendKeys.Len() != 0 {
		t.Errorf("GenerateKeyLifecycle() did not create an empty stack for sendKeys\n\treceived: %v\n\texpected: %v", kl.sendKeys.Len(), 0)
	}

	if kl.sendReKeys.Len() != 0 {
		t.Errorf("GenerateKeyLifecycle() did not create an empty stack for sendReKeys\n\treceived: %v\n\texpected: %v", kl.sendReKeys.Len(), 0)
	}
}

// Tests that Initialise() initialises the keyLifecycle struct correctly.
func TestKeyLifecycle_Initialise(t *testing.T) {
	err := kl.Initialise(ttl, sendKeys, sendReKeys)

	if err != nil {
		t.Errorf("Initialise() resulted in an unexpected error\n\treceived: %v\n\texpected: %v", err, nil)
	}

	if kl.privateKey != nil {
		t.Errorf("Initialise() did not correctly clear the privateKey\n\treceived: %v\n\texpected: %v", kl.privateKey, nil)
	}

	if kl.ttl != ttl {
		t.Errorf("Initialise() did not correctly set ttl\n\treceived: %v\n\texpected: %v", kl.ttl, ttl)
	}

	if kl.sendKeys.Len() != uint(arrSize) {
		t.Errorf("Initialise() did not create a stack for sendKeys of the correct size\n\treceived: %v\n\texpected: %v", kl.sendKeys.Len(), uint(arrSize))
	}

	if kl.sendReKeys.Len() != uint(arrSize) {
		t.Errorf("Initialise() did not create a stack for sendReKeys of the correct size\n\treceived: %v\n\texpected: %v", kl.sendReKeys.Len(), uint(arrSize))
	}

	for i := 0; i < arrSize; i++ {
		if kl.sendKeys.Peak(uint(i)) != &sendKeys[i] {
			t.Errorf("Initialise() did not correctly load the value at index %v onto the sendKeys stack\n\treceived: %v\n\texpected: %v", i, kl.sendKeys.Peak(uint(i)), &sendKeys[i])
		}

		if kl.sendReKeys.Peak(uint(i)) != &sendReKeys[i] {
			t.Errorf("Initialise() did not correctly load the value at index %v onto the sendReKeys stack\n\treceived: %v\n\texpected: %v", i, kl.sendReKeys.Peak(uint(i)), &sendKeys[i])
		}
	}
}

// Tests that Initialise() throws an error when state is not KEYING.
func TestKeyLifecycle_Initialise_ErrorIncorrectState(t *testing.T) {
	kl.state = UNINITIALISED

	err := kl.Initialise(ttl, sendKeys, sendReKeys)

	if err != IncorrectState {
		t.Errorf("Initialise() did not produce an expected error when state = UNINITIALISED\n\treceived: %v\n\texpected: %v", err, IncorrectState)
	}
}

// Tests the thread locking functionality of Initialise().
func TestKeyLifecycle_Initialise_Lock(t *testing.T) {
	kl.Lock()

	result := make(chan bool)

	go func() {
		_ = kl.Initialise(ttl, sendKeys, sendReKeys)
		result <- true
	}()

	select {
	case <-result:
		t.Errorf("Initialise() did not correctly lock the thread")
	case <-time.After(5 * time.Second):
		kl.Unlock()
		return
	}
}

// Tests if all the keys are popped in the correct order by PopSendKey() and
// that when count reaches ttl, the rekey flag is thrown.
func TestKeyLifecycle_PopSendKey(t *testing.T) {
	kl.state = KEYING
	_ = kl.Initialise(ttl, sendKeys, sendReKeys)

	for i := arrSize - 1; i >= 2; i-- {
		key, rekey, err := kl.PopSendKey()

		if !reflect.DeepEqual(key, &sendKeys[i]) {
			t.Errorf("PopSendKey() did not return the correct key\n\treceived: %#v\n\texpected: %#v", key, &sendKeys[i])
		}

		if rekey != false {
			t.Errorf("PopSendKey() rekeyed when it was not supposed to\n\treceived: %#v\n\texpected: %#v", rekey, false)
		}

		if err != nil {
			t.Errorf("PopSendKey() produced an unexpected error\n\treceived: %#v\n\texpected: %#v", err, nil)
		}
	}

	// Check if the rekey flag is thrown
	key, rekey, err := kl.PopSendKey()

	if !reflect.DeepEqual(key, &sendKeys[1]) {
		t.Errorf("PopSendKey() did not return the correct key\n\treceived: %#v\n\texpected: %#v", key, &sendKeys[1])
	}

	if rekey != true {
		t.Errorf("PopSendKey() did not rekey when it was supposed to\n\treceived: %#v\n\texpected: %#v", rekey, true)
	}

	if err != nil {
		t.Errorf("PopSendKey() produced an unexpected error\n\treceived: %#v\n\texpected: %#v", err, nil)
	}
}

// Tests if the correct error is thrown by PopSendKey() when the state is not
// READY.
func TestKeyLifecycle_PopSendKey_ErrorIncorrectState(t *testing.T) {
	_ = kl.Initialise(ttl, sendKeys, sendReKeys)

	kl.state = KEYING

	// Check if the rekey flag is thrown
	key, rekey, err := kl.PopSendKey()

	if !reflect.DeepEqual(key, (*E2EKey)(nil)) {
		t.Errorf("PopSendKey() did not return a nil key when an error was supposed to occur\n\treceived: %#v\n\texpected: %#v", key, (*E2EKey)(nil))
	}

	if rekey != false {
		t.Errorf("PopSendKey() rekeyed when it was not supposed to\n\treceived: %#v\n\texpected: %#v", rekey, true)
	}

	if err != IncorrectState {
		t.Errorf("PopSendKey() did not produce an error when it was supposed to\n\treceived: %#v\n\texpected: %#v", err, IncorrectState)
	}
}

// Tests if the correct error is thrown by PopSendKey() when there are no keys
// to pop.
func TestKeyLifecycle_PopSendKey_NoKeys(t *testing.T) {
	kl = GenerateKeyLifecycle(privateKey, partner)
	kl.state = READY

	// Check if the rekey flag is thrown
	key, rekey, err := kl.PopSendKey()

	fmt.Println(kl.ttl)

	if !reflect.DeepEqual(key, (*E2EKey)(nil)) {
		t.Errorf("PopSendKey() did not return a nil key when an error was supposed to occur\n\treceived: %#v\n\texpected: %#v", key, (*E2EKey)(nil))
	}

	if rekey != false {
		t.Errorf("PopSendKey() rekeyed when it was not supposed to\n\treceived: %#v\n\texpected: %#v", rekey, true)
	}

	if err != NoKeys {
		t.Errorf("PopSendKey() did not produce an error when it was supposed to\n\treceived: %#v\n\texpected: %#v", err, NoKeys)
	}
}

// Tests the thread locking functionality of PopSendKey().
func TestKeyLifecycle_PopSendKey_Lock(t *testing.T) {
	kl.state = KEYING
	_ = kl.Initialise(ttl, sendKeys, sendReKeys)

	kl.Lock()

	result := make(chan bool)

	go func() {
		_, _, _ = kl.PopSendKey()

		result <- true
	}()

	select {
	case <-result:
		t.Errorf("PopSendKey() did not correctly lock the thread")
	case <-time.After(5 * time.Second):
		kl.Unlock()
		return
	}
}

// Tests if all the keys are popped in the correct order by PopSendReKey() and
// that when count reaches ttl, the rekey flag is thrown.
func TestKeyLifecycle_PopSendReKey(t *testing.T) {
	kl.state = KEYING
	_ = kl.Initialise(ttl, sendKeys, sendReKeys)

	for i := arrSize - 1; i >= 2; i-- {
		key, err := kl.PopSendReKey()

		if !reflect.DeepEqual(key, &sendReKeys[i]) {
			t.Errorf("PopSendReKey() did not return the correct key\n\treceived: %#v\n\texpected: %#v", key, &sendReKeys[i])
		}

		if err != nil {
			t.Errorf("PopSendReKey() produced an unexpected error\n\treceived: %#v\n\texpected: %#v", err, nil)
		}
	}
}

// Tests if the correct error is thrown by PopSendReKey() when the state is not
// READY.
func TestKeyLifecycle_PopSendReKey_ErrorIncorrectState(t *testing.T) {
	_ = kl.Initialise(ttl, sendKeys, sendReKeys)

	kl.state = KEYING

	// Check if the rekey flag is thrown
	key, err := kl.PopSendReKey()

	if !reflect.DeepEqual(key, (*E2EKey)(nil)) {
		t.Errorf("PopSendReKey() did not return a nil key when an error was supposed to occur\n\treceived: %#v\n\texpected: %#v", key, (*E2EKey)(nil))
	}

	if err != IncorrectState {
		t.Errorf("PopSendReKey() did not produce an error when it was supposed to\n\treceived: %#v\n\texpected: %#v", err, IncorrectState)
	}
}

// Tests if the correct error is thrown by PopSendReKey() when there are no keys
// to pop.
func TestKeyLifecycle_PopSendReKey_NoKeys(t *testing.T) {
	kl = GenerateKeyLifecycle(privateKey, partner)
	kl.state = READY

	// Check if the rekey flag is thrown
	key, err := kl.PopSendReKey()

	fmt.Println(kl.ttl)

	if !reflect.DeepEqual(key, (*E2EKey)(nil)) {
		t.Errorf("PopSendReKey() did not return a nil key when an error was supposed to occur\n\treceived: %#v\n\texpected: %#v", key, (*E2EKey)(nil))
	}

	if err != NoKeys {
		t.Errorf("PopSendReKey() did not produce an error when it was supposed to\n\treceived: %#v\n\texpected: %#v", err, NoKeys)
	}
}

// Tests the thread locking functionality of PopSendReKey().
func TestKeyLifecycle_PopSendReKey_Lock(t *testing.T) {
	kl.state = KEYING
	_ = kl.Initialise(ttl, sendKeys, sendReKeys)

	kl.Lock()

	result := make(chan bool)

	go func() {
		_, _ = kl.PopSendReKey()

		result <- true
	}()

	select {
	case <-result:
		t.Errorf("PopSendReKey() did not correctly lock the thread")
	case <-time.After(5 * time.Second):
		kl.Unlock()
		return
	}
}

// Test that CopyPrivateKey() returns an actual copy of the private key.
func TestKeyLifecycle_CopyPrivateKey(t *testing.T) {
	kl = GenerateKeyLifecycle(privateKey, partner)
	pKey := kl.CopyPrivateKey()

	// Check value
	if !reflect.DeepEqual(pKey.Bytes(), privateKey.Bytes()) {
		t.Errorf("CopyPrivateKey() did not return the correct private key\n\treceived: %#v\n\texpected: %#v", pKey.Bytes(), privateKey.Bytes())
	}

	if pKey == privateKey {
		t.Errorf("CopyPrivateKey() did not return a copy of privateKey\n\treceived: %#v\n\texpected: %#v", pKey, privateKey)
	}
}

// Check that GetCount() returns the correct count.
func TestKeyLifecycle_GetCount(t *testing.T) {
	if kl.GetCount() != uint32(0) {
		t.Errorf("GetCount() did not return the correct count\n\treceived: %#v\n\texpected: %#v", kl.GetCount(), uint32(0))
	}
}

// Check that GetState() returns the correct state.
func TestKeyLifecycle_GetState(t *testing.T) {
	if kl.GetState() != KEYING {
		t.Errorf("GetState() did not return the correct state\n\treceived: %#v\n\texpected: %#v", kl.GetState(), KEYING)
	}
}

// Check that IncrCount() increases count by 1.
func TestKeyLifecycle_IncrCount(t *testing.T) {
	kl.IncrCount()
	if kl.GetCount() != uint32(1) {
		t.Errorf("IncrCount() did not increment counter by 1\n\treceived: %#v\n\texpected: %#v", kl.GetCount(), uint32(1))
	}
}

// Tests the thread locking functionality of IncrCount().
func TestKeyLifecycle_IncrCount_Lock(t *testing.T) {
	kl.Lock()

	result := make(chan bool)

	go func() {
		kl.IncrCount()

		result <- true
	}()

	select {
	case <-result:
		t.Errorf("IncrCount() did not correctly lock the thread")
	case <-time.After(5 * time.Second):
		kl.Unlock()
		return
	}
}
