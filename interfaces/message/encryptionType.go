package message

type EncryptionType uint8

const (
	None EncryptionType = 0
	E2E  EncryptionType = 1
)
