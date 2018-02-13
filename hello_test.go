package client

import "testing"

func TestGreetings(t *testing.T) {
	actual := Greetings("GlaDOS")
	expected := "Hello, GlaDOS!"

	pass := 0
	test := 1

	if actual != expected {
		t.Errorf("Greetings: Actual (%s) differed from expected (%s).",
			actual, expected)
	} else {
		pass++
	}

	t.Logf("Greetings: %v out of %v tests passed", pass, test)
}
