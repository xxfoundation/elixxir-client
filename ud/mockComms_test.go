package ud

import (
	"crypto/ed25519"
	"fmt"
	"time"

	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"

	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/channel"
)

type mockComms struct {
	udHost                  *connect.Host
	userRsaPub              *rsa.PublicKey
	userEd25519PubKey       []byte
	udLeaseEd25519Signature []byte
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

func (m *mockComms) SetUserRSAPubKey(userRsaPub *rsa.PublicKey) {
	m.userRsaPub = userRsaPub
}

func (m *mockComms) SetUserEd25519PubKey(key ed25519.PublicKey) {
	m.userEd25519PubKey = []byte(key)
}

func (m *mockComms) SetLeaseSignature(signature []byte) {
	m.udLeaseEd25519Signature = signature
}

func (m mockComms) SendChannelLeaseRequest(host *connect.Host, message *pb.ChannelLeaseRequest) (*pb.ChannelLeaseResponse, error) {

	fmt.Printf("message sig: %x\n", message.UserPubKeyRSASignature)
	fmt.Printf("rsa pub key: %v\n", m.userRsaPub)
	err := channel.VerifyChannelIdentityRequest(message.UserPubKeyRSASignature,
		message.UserEd25519PubKey,
		time.Now(),
		time.Unix(0, message.Timestamp),
		m.userRsaPub)

	if err != nil {
		panic(err)
	}

	d, _ := time.ParseDuration("4h30m")

	response := &pb.ChannelLeaseResponse{
		Lease:                   time.Now().Add(d).UnixNano(),
		UserEd25519PubKey:       m.userEd25519PubKey,
		UDLeaseEd25519Signature: m.udLeaseEd25519Signature,
	}

	return response, nil
}
