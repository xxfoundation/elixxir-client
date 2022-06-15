package bindings

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/xxdk"
)

// sets the log level
func init() {
	jww.SetLogThreshold(jww.LevelInfo)
	jww.SetStdoutThreshold(jww.LevelInfo)
}

//client tracker singleton, used to track clients so they can be referenced by
//id back over the bindings
var clientTrackerSingleton = &clientTracker{
	clients: make(map[int]*Client),
	count:   0,
}

// Client BindingsClient wraps the xxdk.Cmix, implementing additional functions
// to support the gomobile Client interface
type Client struct {
	api *xxdk.Cmix
	id  int
}

// NewClient creates client storage, generates keys, connects, and registers
// with the network. Note that this does not register a username/identity, but
// merely creates a new cryptographic identity for adding such information
// at a later date.
//
// Users of this function should delete the storage directory on error.
func NewClient(network, storageDir string, password []byte, regCode string) error {
	if err := xxdk.NewClient(network, storageDir, password, regCode); err != nil {
		return errors.New(fmt.Sprintf("Failed to create new client: %+v",
			err))
	}
	return nil
}

// Login will load an existing client from the storageDir
// using the password. This will fail if the client doesn't exist or
// the password is incorrect.
// The password is passed as a byte array so that it can be cleared from
// memory and stored as securely as possible using the memguard library.
// Login does not block on network connection, and instead loads and
// starts subprocesses to perform network operations.
// TODO: add in custom parameters instead of the default
func Login(storageDir string, password []byte) (*Client, error) {

	client, err := xxdk.Login(storageDir, password, xxdk.GetDefaultParams())
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to login: %+v", err))
	}

	return clientTrackerSingleton.make(client), nil
}

func (c *Client) GetID() int {
	return c.id
}
