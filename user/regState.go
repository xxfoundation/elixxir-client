package user

const (
	NotStarted            uint32 = iota // Set on session creation
	KeyGenComplete               = 1000 // Set upon generation of session information
	PermissioningComplete        = 2000 // Set upon completion of RegisterWithPermissioning
	UDBComplete                  = 3000 // Set upon completion of RegisterWithUdb
)
