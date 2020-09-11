package params

type Unsafe struct {
	CMIX
}

func GetDefaultUnsafe() Unsafe {
	return Unsafe{CMIX: GetDefaultCMIX()}
}
