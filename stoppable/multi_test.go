////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package stoppable

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Tests that NewMulti returns a Multi that is running with the given name.
func TestNewMulti(t *testing.T) {
	name := "testMulti"
	multi := NewMulti(name)

	if multi.name != name {
		t.Errorf("NewMulti returned Multi with incorrect name."+
			"\nexpected: %s\nreceived: %s", name, multi.name)
	}
}

// Tests that Multi.Add adds all the stoppables to the list.
func TestMulti_Add(t *testing.T) {
	multi := NewMulti("testMulti")
	expected := []Stoppable{
		NewSingle("testSingle0"),
		NewMulti("testMulti0"),
		NewSingle("testSingle1"),
		NewMulti("testMulti1"),
	}

	for _, stoppable := range expected {
		multi.Add(stoppable)
	}

	if !reflect.DeepEqual(multi.stoppables, expected) {
		t.Errorf("Add did not add the correct Stoppables."+
			"\nexpected: %+v\nreceived: %+v", multi.stoppables, expected)
	}
}

// Unit test of Multi.Name.
func TestMulti_Name(t *testing.T) {
	name := "testMulti"
	multi := NewMulti(name)

	// Add stoppables and created list of their names
	var nameList []string
	for i := 0; i < 10; i++ {
		newName := ""
		if i%2 == 0 {
			newName = "single" + strconv.Itoa(i)
			multi.Add(NewSingle(newName))
		} else {
			newMulti := NewMulti("multi" + strconv.Itoa(i))
			if i != 5 {
				newMulti.Add(NewMulti("multiA"))
				newMulti.Add(NewMulti("multiB"))
			}
			multi.Add(newMulti)
			newName = newMulti.Name()
		}
		nameList = append(nameList, newName)
	}

	expected := name + "{" + strings.Join(nameList, ", ") + "}"

	if multi.Name() != expected {
		t.Errorf("Name failed to return the expected string."+
			"\nexpected: %s\nreceived: %s", expected, multi.Name())
	}
}

// Tests that Multi.Name returns the expected string when it has no stoppables.
func TestMulti_Name_NoStoppables(t *testing.T) {
	name := "testMulti"
	multi := NewMulti(name)

	expected := name + "{}"

	if multi.Name() != expected {
		t.Errorf("Name failed to return the expected string."+
			"\nexpected: %s\nreceived: %s", expected, multi.Name())
	}
}

// Tests that Multi.GetStatus returns the expected Status.
func TestMulti_GetStatus(t *testing.T) {
	multi := NewMulti("testMulti")
	single1 := NewSingle("testSingle1")
	single2 := NewSingle("testSingle2")
	atomic.StoreUint32((*uint32)(&single2.status), uint32(Stopped))
	multi.Add(single1)
	multi.Add(single2)

	status := multi.GetStatus()
	if status != Running {
		t.Errorf("GetStatus returned the wrong status."+
			"\nexpected: %s\nreceived: %s", Running, status)
	}

	atomic.StoreUint32((*uint32)(&single1.status), uint32(Stopping))
	status = multi.GetStatus()
	if status != Stopping {
		t.Errorf("GetStatus returned the wrong status."+
			"\nexpected: %s\nreceived: %s", Stopping, status)
	}

	atomic.StoreUint32((*uint32)(&single1.status), uint32(Stopped))
	status = multi.GetStatus()
	if status != Stopped {
		t.Errorf("GetStatus returned the wrong status."+
			"\nexpected: %s\nreceived: %s", Stopped, status)
	}
}

// Tests that Multi.GetStatus returns the expected Status when it has no
// children.
func TestMulti_GetStatus_NoChildren(t *testing.T) {
	multi := NewMulti("testMulti")

	status := multi.GetStatus()
	if status != Stopped {
		t.Errorf("GetStatus returned the wrong status."+
			"\nexpected: %s\nreceived: %s", Stopped, status)
	}
}

// Tests that Multi.IsRunning returns the expected value when the Multi is
// marked as running, stopping, and stopped.
func TestMulti_IsRunning(t *testing.T) {
	multi := NewMulti("testMulti")
	single1 := NewSingle("testSingle1")
	single2 := NewSingle("testSingle2")
	atomic.StoreUint32((*uint32)(&single2.status), uint32(Stopping))
	multi.Add(single1)
	multi.Add(single2)

	if result := multi.IsRunning(); !result {
		t.Errorf("IsRunning returned the wrong value when running."+
			"\nexpected: %t\nreceived: %t", true, result)
	}

	atomic.StoreUint32((*uint32)(&single1.status), uint32(Stopping))
	atomic.StoreUint32((*uint32)(&single2.status), uint32(Stopped))
	if result := multi.IsRunning(); result {
		t.Errorf("IsRunning returned the wrong value when stopping."+
			"\nexpected: %t\nreceived: %t", false, result)
	}

	atomic.StoreUint32((*uint32)(&single2.status), uint32(Stopped))
	if result := multi.IsRunning(); result {
		t.Errorf("IsRunning returned the wrong value when stopped."+
			"\nexpected: %t\nreceived: %t", false, result)
	}
}

// Tests that Multi.IsStopping returns the expected value when the Multi is
// marked as running, stopping, and stopped.
func TestMulti_IsStopping(t *testing.T) {
	multi := NewMulti("testMulti")
	single1 := NewSingle("testSingle1")
	single2 := NewSingle("testSingle2")
	atomic.StoreUint32((*uint32)(&single2.status), uint32(Stopped))
	multi.Add(single1)
	multi.Add(single2)

	if result := multi.IsStopping(); result {
		t.Errorf("IsStopping returned the wrong value when running."+
			"\nexpected: %t\nreceived: %t", true, result)
	}

	atomic.StoreUint32((*uint32)(&single1.status), uint32(Stopping))
	if result := multi.IsStopping(); !result {
		t.Errorf("IsStopping returned the wrong value when stopping."+
			"\nexpected: %t\nreceived: %t", false, result)
	}

	atomic.StoreUint32((*uint32)(&single1.status), uint32(Stopped))
	if result := multi.IsStopping(); result {
		t.Errorf("IsStopping returned the wrong value when stopped."+
			"\nexpected: %t\nreceived: %t", false, result)
	}
}

// Tests that Multi.IsStopped returns the expected value when the Multi is
// marked as running, stopping, and stopped.
func TestMulti_IsStopped(t *testing.T) {
	multi := NewMulti("testMulti")
	single1 := NewSingle("testSingle1")
	single2 := NewSingle("testSingle2")
	atomic.StoreUint32((*uint32)(&single2.status), uint32(Stopped))
	multi.Add(single1)
	multi.Add(single2)

	if result := multi.IsStopped(); result {
		t.Errorf("IsStopped returned the wrong value when running."+
			"\nexpected: %t\nreceived: %t", true, result)
	}

	atomic.StoreUint32((*uint32)(&single1.status), uint32(Stopping))
	if result := multi.IsStopped(); result {
		t.Errorf("IsStopped returned the wrong value when stopping."+
			"\nexpected: %t\nreceived: %t", false, result)
	}

	atomic.StoreUint32((*uint32)(&single1.status), uint32(Stopped))
	if result := multi.IsStopped(); !result {
		t.Errorf("IsStopped returned the wrong value when stopped."+
			"\nexpected: %t\nreceived: %t", false, result)
	}
}

// Tests that Multi.IsStopped returns true when all of the child stoppables are
// stopped.
func TestMulti_IsStopped_StoppedStatus(t *testing.T) {
	multi := NewMulti("testMulti")
	singles := []*Single{
		NewSingle("testSingle0"),
		NewSingle("testSingle1"),
		NewSingle("testSingle2"),
		NewSingle("testSingle3"),
		NewSingle("testSingle4"),
	}
	for _, single := range singles[:3] {
		atomic.StoreUint32((*uint32)(&single.status), uint32(Stopped))
		multi.Add(single)
	}
	subMulti := NewMulti("subMulti")
	for _, single := range singles[3:] {
		atomic.StoreUint32((*uint32)(&single.status), uint32(Stopped))
		subMulti.Add(single)
	}
	multi.Add(subMulti)

	if !multi.IsStopped() {
		t.Error("IsStopped did not find all stoppables as stopped.")
	}
}

// Error path: tests that Multi.IsStopped returns false when not all of the
// child stoppables are stopped.
func TestMulti_IsStopped_NotStoppedError(t *testing.T) {
	multi := NewMulti("testMulti")
	singles := []*Single{
		NewSingle("testSingle0"),
		NewSingle("testSingle1"),
		NewSingle("testSingle2"),
		NewSingle("testSingle3"),
		NewSingle("testSingle4"),
	}
	for _, single := range singles {
		multi.Add(single)
	}

	for _, single := range singles[:4] {
		atomic.StoreUint32((*uint32)(&single.status), uint32(Stopped))
	}

	if multi.IsStopped() {
		t.Error("IsStopped found all the stoppables as stopped when some are " +
			"still running")
	}
}

// Tests that Multi.Close sends on all Single quit channels.
func TestMulti_Close(t *testing.T) {
	multi := NewMulti("testMulti")
	singles := []*Single{
		NewSingle("testSingle0"),
		NewSingle("testSingle1"),
		NewSingle("testSingle2"),
		NewSingle("testSingle3"),
		NewSingle("testSingle4"),
	}
	for _, single := range singles[:3] {
		multi.Add(single)
	}
	subMulti := NewMulti("subMulti")
	for _, single := range singles[3:] {
		subMulti.Add(single)
	}
	multi.Add(subMulti)

	for _, single := range singles {
		go func(single *Single) {
			select {
			case <-time.NewTimer(3 * time.Second).C:
				t.Errorf("Single %s failed to quit.", single.Name())
			case <-single.Quit():
			}
		}(single)
	}

	err := multi.Close()
	if err != nil {
		t.Errorf("Close() returned an error: %v", err)
	}

	err = multi.Close()
	if err != nil {
		t.Errorf("Close() returned an error: %v", err)
	}
}

// Error path: tests that Multi.Close returns the expected error when the Single
// stoppables are not running.
func TestMulti_Close_StoppableCloseError(t *testing.T) {
	multi := NewMulti("testMulti")
	var singles []*Single
	for i := 0; i < 5; i++ {
		single := NewSingle("testSingle" + strconv.Itoa(i))
		singles = append(singles, single)
		multi.Add(single)
		atomic.StoreUint32((*uint32)(&single.status), uint32(Stopped))
	}

	var wg sync.WaitGroup
	for _, single := range singles {
		wg.Add(1)
		go func(single *Single) {
			select {
			case <-time.NewTimer(15 * time.Millisecond).C:
			case <-single.Quit():
				t.Errorf("Single %s to quit when it should have failed.",
					single.Name())
			}
			wg.Done()
		}(single)
	}

	expectedErr := fmt.Sprintf(closeMultiErr, multi.name, 0, 0)
	expectedErr = strings.SplitN(expectedErr, " 0/0", 2)[0]

	err := multi.Close()
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Close() did not return the expected error."+
			"\nexpected: %s\nreceived: %v", expectedErr, err)
	}

	wg.Wait()

	err = multi.Close()
	if err != nil {
		t.Errorf("Close() returned an error: %v", err)
	}
}
