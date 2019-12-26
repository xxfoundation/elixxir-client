package user

const (
	NotStarted            uint32 = iota //Set on session creation
	KeyGenComplete               = 1    //Set upon generation of session information
	PermissioningComplete        = 2    //Set upon completion of RegisterWithPermissioning
	UDBComplete                  = 3    //Set upon completion of RegisterWithUdb
)
