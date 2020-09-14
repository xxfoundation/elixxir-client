package stoppable

import (
	"testing"
	"time"
)

// Tests happy path of NewSingle().
func TestNewSingle(t *testing.T) {
	name := "test name"
	single := NewSingle(name)

	if single.name != name || single.running != 1 {
		t.Errorf("NewSingle() returned Single with incorrect values."+
			"\n\texpected:  name: %s  running: %d\n\treceived:  name: %s  running: %d",
			name, 1, single.name, single.running)
	}
}

// Tests happy path of Single.IsRunning().
func TestSingle_IsRunning(t *testing.T) {
	single := NewSingle("name")

	if !single.IsRunning() {
		t.Errorf("IsRunning() returned false when it should be running.")
	}

	single.running = 0
	if single.IsRunning() {
		t.Errorf("IsRunning() returned true when it should not be running.")
	}
}

// Tests happy path of Single.Quit().
func TestSingle_Quit(t *testing.T) {
	single := NewSingle("name")

	go func() {
		time.Sleep(150 * time.Nanosecond)
		single.Quit() <- struct{}{}
	}()

	timer := time.NewTimer(2 * time.Millisecond)
	select {
	case <-timer.C:
		t.Errorf("Quit signal not received.")
	case <-single.quit:
	}
}

// Tests happy path of Single.Name().
func TestSingle_Name(t *testing.T) {
	name := "test name"
	single := NewSingle(name)

	if name != single.Name() {
		t.Errorf("Name() returned the incorrect string."+
			"\n\texpected: %s\n\treceived: %s", name, single.Name())
	}
}

// Test happy path of Single.Close().
func TestSingle_Close(t *testing.T) {
	single := NewSingle("name")

	go func() {
		time.Sleep(150 * time.Nanosecond)
		select {
		case <-single.quit:
		}
	}()

	err := single.Close(5 * time.Millisecond)
	if err != nil {
		t.Errorf("Close() returned an error: %v", err)
	}
}

// Tests that Single.Close() returns an error when the timeout is reached.
func TestSingle_Close_Error(t *testing.T) {
	single := NewSingle("name")
	expectedErr := single.name + " failed to close"

	go func() {
		time.Sleep(3 * time.Millisecond)
		select {
		case <-single.quit:
		}
	}()

	err := single.Close(2 * time.Millisecond)
	if err == nil {
		t.Errorf("Close() did not return the expected error."+
			"\n\texpected: %v\n\treceived: %v", expectedErr, err)
	}
}
