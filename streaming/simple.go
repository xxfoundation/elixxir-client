package streaming

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/connect"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/receive"
	"io"
	"sync"
)

// Params describes parameters for a simple streaming object
type Params struct {
	E2E e2e.Params
}

// SimpleStreaming is the public facing interface for a simple cmix stream
type SimpleStreaming interface {
	io.Reader

	io.Writer

	io.Closer
}

// simple is the private struct implementing SimpleStreaming
type simple struct {
	params Params

	// Underlying connection object used for reading/writing to cmix
	c connect.Connection

	// read data buffer populated by callback
	readBuf     []byte
	readBufLock sync.Mutex

	// ID of listener registered to receive read data
	listenerID receive.ListenerID
}

// NewStream creates a simple struct and returns it as a SimpleStreaming interface
// Accepts connect.Connection interface and Params object
// Returns a simple stream wrapped as a SimpleStreaming interface
func NewStream(c connect.Connection, params Params) (SimpleStreaming, error) {
	s := &simple{
		c:           c,
		params:      params,
		readBuf:     nil,
		readBufLock: sync.Mutex{},
	}
	// register a listener to feed into the readBuf
	lid := c.RegisterListener(catalog.SimpleStream, s)
	s.listenerID = lid
	return s, nil
}

/* io.Writer implementation */

// Write bytes in p to the simple stream via SendE2E, returning num bytes written
// This is currently just a wrapper around connection.SendE2E, since max message size & splitting is totally opaque to this layer
func (s *simple) Write(p []byte) (int, error) {
	_, _, _, err := s.c.SendE2E(catalog.SimpleStream, p, s.params.E2E)
	if err != nil {
		return 0, errors.WithMessage(err, "Failed to send e2e streaming message")
	}
	return len(p), nil
}

/* io.Reader implementation */

// Read at most len(p) bytes into array p, returning num bytes read
// This reads off of simple.readBuf, which is a bytes buffer populated as the Hear callback is called
func (s *simple) Read(p []byte) (n int, err error) {
	s.readBufLock.Lock()
	defer s.readBufLock.Unlock()

	// TODO: this will need consistency testing specifically to ensure no cases lose data on the end
	n = copy(p, s.readBuf)
	if len(p) < len(s.readBuf) {
		s.readBuf = s.readBuf[len(p):]
	} else {
		s.readBuf = []byte{}
	}
	return
}

/* io.Closer implementation*/

// Close the simple streaming interface
func (s *simple) Close() error {
	s.c.Unregister(s.listenerID)
	return nil
}

/* receive.Listener implementation */
// Note: these are not accessible to the top level, since we return the SimpleStreaming interface
// Allows for a clean reading implementation

// Hear is a callback registered on the connection object, populates the read buffer when called
func (s *simple) Hear(item receive.Message) {
	s.readBufLock.Lock()
	defer s.readBufLock.Unlock()

	s.readBuf = append(s.readBuf, item.Payload...)
	return
}

// Name returns a name for the listener registered
func (s *simple) Name() string {
	return "simple-streaming"
}
