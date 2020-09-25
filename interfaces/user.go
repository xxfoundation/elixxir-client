package interfaces

type User interface {
	GetID() []byte
	GetSalt() []byte
	GetRSAPrivateKeyPem() []byte
	GetRSAPublicKeyPem() []byte
	IsPrecanned() bool
	GetCmixDhPrivateKey() []byte
	GetCmixDhPublicKey() []byte
	GetE2EDhPrivateKey() []byte
	GetE2EDhPublicKey() []byte
	GetContact() Contact
}
