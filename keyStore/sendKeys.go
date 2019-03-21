package keyStore

//Stores all the keys from a single key negotiation.  Will be stored an a map keyed on user id.  To send a message,
//the user to send to will be looked up and then a send key will be popped from the negotiation.
type SendKeyset struct {
	// List of Keys used for sending. When a key is used it is deleted.
	sendKeys *LIFO

	// List of ReKey Keys that can be sent. When a key is used it is deleted.
	sendReKeys *LIFO

	// pointer to controling lifecycle
	lifecycle *KeyLifecycle
}
