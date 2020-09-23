package params

import "testing"

func TestGetDefaultE2E(t *testing.T) {
	if GetDefaultE2E().Type != Standard {
		t.Errorf("GetDefaultE2E did not return Standard")
	}
}

func TestSendType_String(t *testing.T) {
	e := E2E{Type: Standard}
	if e.Type.String() != "Standard" {
		t.Errorf("Running String on Standard E2E type got %s", e.Type.String())
	}

	e = E2E{Type: KeyExchange}
	if e.Type.String() != "KeyExchange" {
		t.Errorf("Running String on KeyExchange E2E type got %s", e.Type.String())
	}

	e = E2E{Type: SendType(40)}
	if e.Type.String() != "Unknown SendType 40" {
		t.Errorf("Running String on unknown E2E type got %s", e.Type.String())
	}
}
