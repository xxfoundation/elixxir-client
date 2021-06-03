///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package stoppable

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
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

	if multi.running != running {
		t.Errorf("NewMulti returned Multi with incorrect running."+
			"\nexpected: %d\nreceived: %d", running, multi.running)
	}
}

// Tests that Multi.IsRunning returns the expected value when the Multi is
// marked as both running and not running.
func TestMulti_IsRunning(t *testing.T) {
	multi := NewMulti("testMulti")

	if !multi.IsRunning() {
		t.Errorf("IsRunning returned the wrong value when running."+
			"\nexpected: %t\nreceived: %t", true, multi.IsRunning())
	}

	multi.running = stopped
	if multi.IsRunning() {
		t.Errorf("IsRunning returned the wrong value when not running."+
			"\nexpected: %t\nreceived: %t", false, multi.IsRunning())
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

	expected := name + ": {" + strings.Join(nameList, ", ") + "}"

	if multi.Name() != expected {
		t.Errorf("Name failed to return the expected string."+
			"\nexpected: %s\nreceived: %s", expected, multi.Name())
	}
}

// Tests that Multi.Name returns the expected string when it has no stoppables.
func TestMulti_Name_NoStoppables(t *testing.T) {
	name := "testMulti"
	multi := NewMulti(name)

	expected := name + ": {" + "}"

	if multi.Name() != expected {
		t.Errorf("Name failed to return the expected string."+
			"\nexpected: %s\nreceived: %s", expected, multi.Name())
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
			case <-time.NewTimer(5 * time.Millisecond).C:
				t.Errorf("Single %s failed to quit.", single.Name())
			case <-single.Quit():
			}
		}(single)
	}

	err := multi.Close(5 * time.Millisecond)
	if err != nil {
		t.Errorf("Close() returned an error: %v", err)
	}

	err = multi.Close(0)
	if err != nil {
		t.Errorf("Close() returned an error: %v", err)
	}
}

// Tests that Multi.Close sends on all Single quit channels.
func TestMulti_Close_Error(t *testing.T) {
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

	for _, single := range singles[:2] {
		go func(single *Single) {
			select {
			case <-time.NewTimer(5 * time.Millisecond).C:
				t.Errorf("Single %s failed to quit.", single.Name())
			case <-single.Quit():
			}
		}(single)
	}
	expectedErr := fmt.Sprintf(closeMultiErr, multi.name, 0, 0)
	expectedErr = strings.SplitN(expectedErr, " 0/0", 2)[0]

	err := multi.Close(5 * time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Close() did not return the expected error."+
			"\nexpected: %s\nreceived: %v", expectedErr, err)
	}

	err = multi.Close(0)
	if err != nil {
		t.Errorf("Close() returned an error: %v", err)
	}
}
