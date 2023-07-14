package remoteSync

import (
	"errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/collective"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/remoteSync/client"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

const errNotLoggedIn = "must log in before using remoteSync features"

type Param struct {
	Username, Path     string
	PasswordHash, Salt []byte

	CommsPub, CommsPriv []byte

	RsId   *id.ID
	RsHost *connect.Host
}

type manager struct {
	params  Param
	rsComms *client.Comms
	token   string
}

func GetRemoteSyncManager(params Param) (collective.RemoteStore, error) {
	if params.Username == "" || params.Path == "" || params.PasswordHash == nil || params.Salt == nil ||
		params.RsId == nil || params.RsHost == nil {
		return nil, errors.New("must fill out all params for remote sync")
	}
	cc, err := client.NewClientComms(params.RsId, params.CommsPub, params.CommsPriv, params.Salt)
	if err != nil {
		return nil, err
	}

	m := &manager{rsComms: cc, params: params}
	err = m.login(params.Username, params.PasswordHash, params.Salt)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (m *manager) login(username string, passwordHash, salt []byte) error {
	resp, err := m.rsComms.Login(m.params.RsHost, &pb.RsAuthenticationRequest{Username: username, PasswordHash: passwordHash, Salt: salt})
	if err != nil {
		return err
	}
	m.token = resp.GetToken()
	expiresAt := time.Unix(0, resp.GetExpiresAt())
	go func() {
		time.Sleep(time.Until(expiresAt))
		m.token = ""
		err = m.login(username, passwordHash, salt)
		if err != nil {
			jww.ERROR.Printf("Failed to log in after token expiry: %+v", err)
		}
	}()
	return nil
}

func (m *manager) Read(path string) ([]byte, error) {
	if m.token == "" {
		return nil, errors.New(errNotLoggedIn)
	}
	resp, err := m.rsComms.Read(m.params.RsHost, &pb.RsReadRequest{Path: path, Token: m.token})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (m *manager) Write(path string, data []byte) error {
	if m.token == "" {
		return errors.New(errNotLoggedIn)
	}
	resp, err := m.rsComms.Write(m.params.RsHost, &pb.RsWriteRequest{
		Path:  path,
		Data:  data,
		Token: m.token,
	})
	if err != nil {
		return err
	} else if resp.Error != "" {
		return errors.New(resp.Error)
	}
	return nil
}

func (m *manager) GetLastModified(path string) (time.Time, error) {
	if m.token == "" {
		return time.Time{}, errors.New(errNotLoggedIn)
	}
	resp, err := m.rsComms.GetLastModified(m.params.RsHost, &pb.RsReadRequest{Path: path, Token: m.token})
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(0, resp.GetTimestamp()), nil
}

func (m *manager) GetLastWrite() (time.Time, error) {
	if m.token == "" {
		return time.Time{}, errors.New(errNotLoggedIn)
	}
	resp, err := m.rsComms.GetLastWrite(m.params.RsHost, &pb.RsLastWriteRequest{Token: m.token})
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(0, resp.GetTimestamp()), nil
}

func (m *manager) ReadDir(path string) ([]string, error) {
	if m.token == "" {
		return nil, errors.New(errNotLoggedIn)
	}

	resp, err := m.rsComms.ReadDir(m.params.RsHost, &pb.RsReadRequest{Path: path, Token: m.token})
	if err != nil {
		return nil, err
	}

	return resp.GetData(), nil
}
