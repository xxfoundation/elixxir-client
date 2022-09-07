////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package stoppable

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	jww.SetStdoutThreshold(jww.LevelTrace)

	os.Exit(m.Run())
}

// Tests that WaitForStopped does not return an error when all children are
// stopped.
func TestWaitForStopped(t *testing.T) {
	m := newTestMulti()

	err := m.Close()
	if err != nil {
		t.Errorf("Failed to close multi stoppable: %+v", err)
	}

	err = WaitForStopped(m, 2*time.Second)
	if err != nil {
		t.Errorf("WaitForStopped returned an error: %+v", err)
	}
}

// Error path: tests that WaitForStopped returns an error if the timeout is
// reached before all stoppables are checked.
func TestWaitForStopped_TimeoutError(t *testing.T) {
	m := newTestMulti()

	err := m.Close()
	if err != nil {
		t.Errorf("Failed to close multi stoppable: %+v", err)
	}

	expectedErr := fmt.Sprintf(timeoutErr, time.Duration(0), m.Name())

	err = WaitForStopped(m, 0)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("WaitForStopped did not return the expected error."+
			"\nexpected: %s\nrecieved: %+v", expectedErr, err)
	}
}

// Tests that TestCheckErr returns true for stoppable errors and false for all
// other errors
func TestCheckErr(t *testing.T) {
	testValues := []struct {
		err      error
		expected bool
	}{
		{errors.Errorf(ErrMsg, "testThre", "testFunc"), true},
		{errors.Errorf(ErrMsg, "", ""), true},
		{errors.Errorf(errKey), true},
		{errors.Errorf("Random error"), false},
		{errors.Errorf(""), false},
		{nil, false},
	}

	for i, val := range testValues {
		result := CheckErr(val.err)
		if result != val.expected {
			t.Errorf("CheckErr failed to return the expected value (%d)."+
				"\nexpected: %t\nreceived: %t", i, val.expected, result)
		}
	}
}

// newTestMulti creates a new Multi Stoppable that has many Single and Multi
// stoppable children.
func newTestMulti() *Multi {
	singles := make([]*Single, 15)
	for i := range singles {
		singles[i] = NewSingle("testSingle_" + strconv.Itoa(i))
		go func(single *Single) {
			<-single.Quit()
			time.Sleep(600 * time.Millisecond)
			single.ToStopped()
		}(singles[i])
	}

	m := NewMulti("testMulti")
	for _, s := range singles[:5] {
		m.Add(s)
	}
	m0 := NewMulti("testMulti_0")
	for _, s := range singles[5:8] {
		m0.Add(s)
	}
	m.Add(m0)
	m1 := NewMulti("testMulti_1")
	for _, s := range singles[8:10] {
		m1.Add(s)
	}
	m2 := NewMulti("testMulti_2")
	for _, s := range singles[10:13] {
		m2.Add(s)
	}
	m1.Add(m2)
	m.Add(m1)
	for _, s := range singles[13:] {
		m.Add(s)
	}
	m.Add(NewMulti("testMulti_3"))

	return m
}
