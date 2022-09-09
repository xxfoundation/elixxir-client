////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ud

import (
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/user"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/version"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"time"
)

type mockStorage struct{}

func (m mockStorage) GetKV() *versioned.KV {
	return versioned.NewKV(ekv.MakeMemstore())
}

func (m mockStorage) GetClientVersion() version.Version {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) Get(key string) (*versioned.Object, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) Set(key string, object *versioned.Object) error {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) Delete(key string) error {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) GetCmixGroup() *cyclic.Group {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) GetE2EGroup() *cyclic.Group {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) ForwardRegistrationStatus(regStatus storage.RegistrationStatus) error {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) GetRegistrationStatus() storage.RegistrationStatus {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) SetRegCode(regCode string) {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) GetRegCode() (string, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) SetNDF(def *ndf.NetworkDefinition) {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) GetNDF() *ndf.NetworkDefinition {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) GetTransmissionID() *id.ID {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) GetTransmissionSalt() []byte {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) GetReceptionID() *id.ID {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) GetReceptionSalt() []byte {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) GetReceptionRSA() *rsa.PrivateKey {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) GetTransmissionRSA() *rsa.PrivateKey {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) IsPrecanned() bool {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) SetUsername(username string) error {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) GetUsername() (string, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) PortableUserInfo() user.Info {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) GetTransmissionRegistrationValidationSignature() []byte {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) GetReceptionRegistrationValidationSignature() []byte {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) GetRegistrationTimestamp() time.Time {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) SetTransmissionRegistrationValidationSignature(b []byte) {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) SetReceptionRegistrationValidationSignature(b []byte) {
	//TODO implement me
	panic("implement me")
}

func (m mockStorage) SetRegistrationTimestamp(tsNano int64) {
	//TODO implement me
	panic("implement me")
}
