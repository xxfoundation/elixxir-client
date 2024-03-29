package remoteSync

import (
	"time"

	"github.com/pkg/errors"

	"gitlab.com/elixxir/client/v4/collective"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/remoteSync/client"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
)

// This error matches the one returned by the remote sync server repo
var errNotLoggedIn = errors.New("Invalid token, login required")

// manager is an internal struct which implements the RemoteStore interface
// and stores info surrounding a Remote Sync server.
type manager struct {
	rsComms Comms
	rsHost  *connect.Host
	rng     csprng.Source

	username, password string

	token []byte
}

// NewRemoteSyncStore returns a collective.RemoteStore interface which can
// be used to interact with a remote sync server. This accepts a username and
// password for the remote sync server, an ID, a host for the server connection,
// and an RNG source.
func NewRemoteSyncStore(username, password string, rsCert []byte, rsId *id.ID,
	rsHost *connect.Host, rng csprng.Source) (collective.RemoteStore, error) {
	if username == "" || password == "" || rsId == nil || rsHost == nil {
		return nil, errors.New("Critical input for remote sync missing")
	}
	cc, err := client.NewClientComms(rsId, rsCert, nil, cmix.NewSalt(rng, 32))
	if err != nil {
		return nil, err
	}

	m := &manager{
		rng:      rng,
		rsComms:  cc,
		rsHost:   rsHost,
		username: username,
		password: password,
	}
	err = m.login()
	if err != nil {
		return nil, err
	}

	return m, nil
}

// login is an internal function which fetches a token on start or if the
// current one is invalid.
func (m *manager) login() error {
	salt := cmix.NewSalt(m.rng, 32)
	h := hash.CMixHash.New()
	h.Write([]byte(m.password))
	h.Write(salt)
	passwordHash := h.Sum(nil)

	resp, err := m.rsComms.Login(m.rsHost, &pb.RsAuthenticationRequest{
		Username:     m.username,
		PasswordHash: passwordHash,
		Salt:         salt,
	})
	if err != nil {
		return err
	}
	m.token = resp.GetToken()
	return nil
}

// Read reads a resource from a path on the remote sync server.
func (m *manager) Read(path string) ([]byte, error) {
	resp, err := m.rsComms.Read(m.rsHost, &pb.RsReadRequest{
		Path:  path,
		Token: m.token,
	})
	if err != nil {
		if errors.Is(err, errNotLoggedIn) {
			if err = m.login(); err != nil {
				return nil, errors.Errorf(
					"Failed to read due to failed login: %+v", err)
			}
			return m.Read(path)
		}
		return nil, err
	}
	return resp.Data, nil
}

// Write writes the data to a path on a remote sync server.
func (m *manager) Write(path string, data []byte) error {
	_, err := m.rsComms.Write(m.rsHost, &pb.RsWriteRequest{
		Path:  path,
		Data:  data,
		Token: m.token,
	})
	if err != nil {
		if errors.Is(err, errNotLoggedIn) {
			if err = m.login(); err != nil {
				return errors.Errorf(
					"Failed to write due to failed login: %+v", err)
			}
			return m.Write(path, data)
		}
		return err
	}
	return nil
}

// GetLastModified returns the time that the path on the remote sync server was
// last modified.
func (m *manager) GetLastModified(path string) (time.Time, error) {
	resp, err := m.rsComms.GetLastModified(m.rsHost, &pb.RsReadRequest{
		Path:  path,
		Token: m.token,
	})
	if err != nil {
		if errors.Is(err, errNotLoggedIn) {
			if err = m.login(); err != nil {
				return time.Time{}, errors.Errorf(
					"Failed to get last modified due to failed login: %+v", err)
			}
			return m.GetLastModified(path)
		}
		return time.Time{}, err
	}

	return time.Unix(0, resp.GetTimestamp()), nil
}

// GetLastWrite time for a remote sync server.
func (m *manager) GetLastWrite() (time.Time, error) {
	resp, err := m.rsComms.GetLastWrite(
		m.rsHost, &pb.RsLastWriteRequest{Token: m.token})
	if err != nil {
		if errors.Is(err, errNotLoggedIn) {
			if err = m.login(); err != nil {
				return time.Time{}, errors.Errorf(
					"Failed to get last write due to failed login: %+v", err)
			}
			return m.GetLastWrite()
		}
		return time.Time{}, err
	}

	return time.Unix(0, resp.GetTimestamp()), nil
}

// ReadDir returns all data for a path on a remote sync server.
func (m *manager) ReadDir(path string) ([]string, error) {
	resp, err := m.rsComms.ReadDir(m.rsHost, &pb.RsReadRequest{
		Path:  path,
		Token: m.token,
	})
	if err != nil {
		if errors.Is(err, errNotLoggedIn) {
			if err = m.login(); err != nil {
				return nil, errors.Errorf(
					"Failed to read dir due to failed login: %+v", err)
			}
			return m.ReadDir(path)
		}
		return nil, err
	}

	return resp.GetData(), nil
}
