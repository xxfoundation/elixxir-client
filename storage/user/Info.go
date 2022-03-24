package user

import "gitlab.com/elixxir/client/interfaces/user"

func (u *User) PortableUserInfo() user.Info {
	ci := u.CryptographicIdentity
	return user.Info{
		TransmissionID:        ci.GetTransmissionID().DeepCopy(),
		TransmissionSalt:      copySlice(ci.GetTransmissionSalt()),
		TransmissionRSA:       ci.GetTransmissionRSA(),
		ReceptionID:           ci.GetReceptionID().DeepCopy(),
		RegistrationTimestamp: u.GetRegistrationTimestamp().UnixNano(),
		ReceptionSalt:         copySlice(ci.GetReceptionSalt()),
		ReceptionRSA:          ci.GetReceptionRSA(),
		Precanned:             ci.IsPrecanned(),
		//fixme: set these in the e2e layer, the command line layer
		//needs more logical seperation so this can be removed
		E2eDhPrivateKey: nil,
		E2eDhPublicKey:  nil,
	}

}

func copySlice(s []byte) []byte {
	n := make([]byte, len(s))
	copy(n, s)
	return n
}
