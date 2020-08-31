package params

type CMIX struct {
	Retries uint
}

func GetDefaultCMIX() CMIX {
	return CMIX{Retries: 3}
}
