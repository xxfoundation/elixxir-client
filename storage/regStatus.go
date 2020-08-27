package storage

type RegistrationStatus uint32

const (
	NotStarted            RegistrationStatus = 0     // Set on session creation
	KeyGenComplete        RegistrationStatus = 10000 // Set upon generation of session information
	PermissioningComplete RegistrationStatus = 20000 // Set upon completion of RegisterWithPermissioning
	UDBComplete           RegistrationStatus = 30000 // Set upon completion of RegisterWithUdb
)
