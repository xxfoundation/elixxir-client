package api

import (
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/elixxir/client/ud"
)

/*
// Returns true if the cryptographic identity has been registered with
// the CMIX user discovery agent.
// Note that clients do not need to perform this step if they use
// out of band methods to exchange cryptographic identities
// (e.g., QR codes), but failing to be registered precludes usage
// of the user discovery mechanism (this may be preferred by user).
func (c *Client) IsRegistered() bool {
	jww.INFO.Printf("IsRegistered()")
	return false
}

// RegisterIdentity registers an arbitrary username with the user
// discovery protocol. Returns an error when it cannot connect or
// the username is already registered.
func (c *Client) RegisterIdentity(username string) error {
	jww.INFO.Printf("RegisterIdentity(%s)", username)
	return nil
}

// RegisterEmail makes the users email searchable after confirmation.
// It returns a registration confirmation token to be used with
// ConfirmRegistration or an error on failure.
func (c *Client) RegisterEmail(email string) ([]byte, error) {
	jww.INFO.Printf("RegisterEmail(%s)", email)
	return nil, nil
}

// RegisterPhone makes the users phone searchable after confirmation.
// It returns a registration confirmation token to be used with
// ConfirmRegistration or an error on failure.
func (c *Client) RegisterPhone(phone string) ([]byte, error) {
	jww.INFO.Printf("RegisterPhone(%s)", phone)
	return nil, nil
}

// ConfirmRegistration sends the user discovery agent a confirmation
// token (from register Email/Phone) and code (string sent via Email
// or SMS to confirm ownership) to confirm ownership.
func (c *Client) ConfirmRegistration(token, code []byte) error {
	jww.INFO.Printf("ConfirmRegistration(%s, %s)", token, code)
	return nil
}

// Search accepts a "separator" separated list of search elements with
// an associated list of searchTypes. It returns a ContactList which
// allows you to iterate over the found contact objects.
func (c *Client) Search(data, separator string, searchTypes []byte) []contact.Contact {
	jww.INFO.Printf("Search(%s, %s, %s)", data, separator, searchTypes)
	return nil
}

// SearchWithHandler is a non-blocking search that also registers
// a callback interface for user disovery events.
func (c *Client) SearchWithCallback(data, separator string, searchTypes []byte,
	cb func(results []contact.Contact)) {
	resultCh := make(chan []contact.Contact, 1)
	go func(out chan []contact.Contact, data, separator string, srchTypes []byte) {
		out <- c.Search(data, separator, srchTypes)
		close(out)
	}(resultCh, data, separator, searchTypes)

	go func(in chan []contact.Contact, cb func(results []contact.Contact)) {
		select {
		case contacts := <-in:
			cb(contacts)
			//TODO: Timer
		}
	}(resultCh, cb)
}*/

func (c *Client) StartUD() (*ud.Manager, error) {
	m, err := ud.NewManager(c.comms, c.rng, c.switchboard, c.storage, c.network)
	if err!=nil{
		return nil, err
	}

	c.serviceProcessies = append(c.serviceProcessies, m.StartProcesses())
	c.runner.Add(m.StartProcesses())
	return m, nil
}
