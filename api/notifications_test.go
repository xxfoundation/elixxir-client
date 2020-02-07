package api

import "testing"

// Happy path
func TestClient_RegisterForNotifications(t *testing.T) {
	// Initialize client with dummy storage
	storage := DummyStorage{LocationA: "Blah", StoreA: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&storage, "hello", "", def)
	if err != nil {
		t.Errorf("Failed to initialize dummy client: %s", err.Error())
		return
	}

	// InitNetwork to gateways and reg server
	err = client.InitNetwork()

	if err != nil {
		t.Errorf("Client failed of connect: %+v", err)
	}

	token := make([]byte, 32)

	err = client.RegisterForNotifications(token)
	if err != nil {
		t.Errorf("Expected happy path, received error: %+v", err)
	}
}

// Happy path
func TestClient_UnregisterForNotifications(t *testing.T) {
	// Initialize client with dummy storage
	storage := DummyStorage{LocationA: "Blah", StoreA: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&storage, "hello", "", def)
	if err != nil {
		t.Errorf("Failed to initialize dummy client: %s", err.Error())
	}

	// InitNetwork to gateways and reg server
	err = client.InitNetwork()

	if err != nil {
		t.Errorf("Client failed of connect: %+v", err)
	}

	err = client.UnregisterForNotifications()
	if err != nil {
		t.Errorf("Expected happy path, received error: %+v", err)
	}
}
