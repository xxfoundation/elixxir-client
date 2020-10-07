package bindings

import "gitlab.com/elixxir/client/switchboard"

// Listener provides a callback to hear a message
// An object implementing this interface can be called back when the client
// gets a message of the type that the regi    sterer specified at registration
// time.
type Listener interface {
	// Hear is called to receive a message in the UI
	Hear(message Message)
	// Returns a name, used for debugging
	Name() string
}

// id object returned when a listener is created and is used to delete it from
// the system. Beyond calling unregister it has no uses.
type ListenerID struct {
	id switchboard.ListenerID
}

func (lid ListenerID) GetUserID() []byte {
	return lid.id.GetUserID().Bytes()
}

func (lid ListenerID) GetMessageType() int {
	return int(lid.id.GetMessageType())
}

func (lid ListenerID) GetName() string {
	return lid.id.GetName()
}
