package ud

import (
	"crypto/ed25519"
	"time"

	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"

	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/channel"
)

type mockComms struct {
	udHost            *connect.Host
	userRsaPub        *rsa.PublicKey
	userEd25519PubKey []byte
	udPrivKey         *ed25519.PrivateKey
	username          string
}

func (m mockComms) SendRegisterUser(host *connect.Host, message *pb.UDBUserRegistration) (*messages.Ack, error) {
	return nil, nil
}

func (m mockComms) SendRegisterFact(host *connect.Host, message *pb.FactRegisterRequest) (*pb.FactRegisterResponse, error) {
	return nil, nil
}

func (m mockComms) SendConfirmFact(host *connect.Host, message *pb.FactConfirmRequest) (*messages.Ack, error) {
	return nil, nil
}

func (m mockComms) SendRemoveFact(host *connect.Host, message *pb.FactRemovalRequest) (*messages.Ack, error) {
	return nil, nil
}

func (m mockComms) SendRemoveUser(host *connect.Host, message *pb.FactRemovalRequest) (*messages.Ack, error) {
	return nil, nil
}

func (m *mockComms) AddHost(hid *id.ID, address string, cert []byte, params connect.HostParams) (host *connect.Host, err error) {
	h, err := connect.NewHost(hid, address, cert, params)
	if err != nil {
		return nil, err
	}

	m.udHost = h
	return h, nil
}

func (m mockComms) GetHost(hostId *id.ID) (*connect.Host, bool) {
	return m.udHost, true
}

func (m *mockComms) SetUDEd25519PrivateKey(key *ed25519.PrivateKey) {
	m.udPrivKey = key
}

func (m *mockComms) SetUserRSAPubKey(userRsaPub *rsa.PublicKey) {
	m.userRsaPub = userRsaPub
}

func (m *mockComms) SetUserEd25519PubKey(key ed25519.PublicKey) {
	m.userEd25519PubKey = []byte(key)
}

func (m *mockComms) SetUsername(u string) {
	m.username = u
}

func (m mockComms) SendChannelLeaseRequest(host *connect.Host, message *pb.ChannelLeaseRequest) (*pb.ChannelLeaseResponse, error) {

	err := channel.VerifyChannelIdentityRequest(message.UserPubKeyRSASignature,
		message.UserEd25519PubKey,
		time.Now(),
		time.Unix(0, message.Timestamp),
		m.userRsaPub)
	if err != nil {
		panic(err)
	}

	d, _ := time.ParseDuration("4h30m")
	lease := time.Now().Add(d).UnixNano()
	signature := channel.SignChannelLease(m.userEd25519PubKey, m.username,
		time.Unix(0, lease), *m.udPrivKey)

	if err != nil {
		panic(err)
	}

	response := &pb.ChannelLeaseResponse{
		Lease:                   lease,
		UserEd25519PubKey:       m.userEd25519PubKey,
		UDLeaseEd25519Signature: signature,
	}

	return response, nil
}
