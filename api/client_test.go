////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import "testing"

func TestLogin(t *testing.T) {
	expected := []bool{false, true}
	actual := make([]bool, 2)
	tests := len(expected)
	pass := 0

	for i := 0; i < tests; i++ {
		actual[i] = Login(i, "127.0.0.1")
	}
	for i := 0; i < tests; i++ {
		if actual[i] != expected[i] {
			t.Errorf("Test of Login() failed: expected[%v]=%v, actual[%v]=%v\n",
				i, expected[i], i, actual[i])
		} else {
			pass++
		}
	}

	println("Login():", pass, "out of", tests, "tests passed")
}

func TestGetNick(t *testing.T) {
	expected := []string{"David", "Jim", "Ben", "Rick", "Spencer"}
	actual := make([]string, 5)
	tests := len(expected)
	pass := 0

	for i := 0; i < tests; i++ {
		actual[i] = GetNick(i+1)
	}
	for i := 0; i < tests; i++ {
		if actual[i] != expected[i] {
			t.Errorf("Test of GetNick() failed: expected[%v]=%v, actual[%v]=%v\n",
				i, expected[i], i, actual[i])
		} else {
			pass++
		}
	}

	println("GetNick():", pass, "out of", tests, "tests passed")
}

func TestReceiveAndSend(t *testing.T) {
	// TODO
}
