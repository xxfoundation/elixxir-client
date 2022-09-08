package payments

import (
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/client/xxdk"
	cmix2 "gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/cyclic"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

type MockPaymentE2e struct {
	t   *testing.T
	rng csprng.Source
}

func (me2e *MockPaymentE2e) GetRng() *fastRNG.StreamGenerator {
	return fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
}
func (me2e *MockPaymentE2e) GetE2E() e2e.Handler {
	return nil
}
func (me2e *MockPaymentE2e) GetCmix() cmix.Client {
	return nil
}
func (me2e *MockPaymentE2e) GetReceptionIdentity() xxdk.ReceptionIdentity {
	rid := id.NewIdFromString("zezima", id.User, me2e.t)
	pk, err := rsa.GenerateKey(me2e.rng, 2048)
	if err != nil {
		me2e.t.Fatalf("Failed to generate reception identity: %+v", err)
	}
	salt := cmix2.NewSalt(me2e.rng, 16)
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	dhpk := dh.GeneratePrivateKey(2048, grp, me2e.rng)
	return xxdk.ReceptionIdentity{
		ID:            rid,
		RSAPrivatePem: rsa.CreatePrivateKeyPem(pk),
		Salt:          salt,
		DHKeyPrivate:  dhpk.Bytes(),
		E2eGrp:        grp.GetPBytes(),
	}
}

func TestManager_Request(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	m, err := NewOrLoad(kv, &MockPaymentE2e{
		t:   t,
		rng: csprng.NewSystemRNG(),
	}, nil, nil)
	if err != nil {
		t.Fatalf("Failed to init payments: %+v", err)
	}
	pid, err := m.Request("10", "", "", "", id.NewIdFromString("test", id.User, t))
	if err != nil {
		t.Fatalf("Failed to request payment: %+v", err)
	}
	req, ok := m.GetRequest(pid)
	if !ok {
		t.Fatal("Failed to store request in manager")
	}
	if !(req.GetStatus() == Sent) {
		t.Fatalf("Stored payment request status differs from expected\n\tStored: %s\n\tExpected: %s\n", req.GetStatus(), Sent)
	}
}

func TestManager_Approve(t *testing.T) {

}

func TestManager_GetRequest(t *testing.T) {

}

func TestPayment_GetInfo(t *testing.T) {

}

func TestPayment_GetStatus(t *testing.T) {

}
