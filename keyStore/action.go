package keyStore

type Action uint8

const (
	None Action = iota
	Rekey
	Purge
	Deleted
)

func (a Action) String() string {
	var ret string
	switch a {
	case None:
		ret = "None"
	case Rekey:
		ret = "Rekey"
	case Purge:
		ret = "Purge"
	case Deleted:
		ret = "Deleted"
	}
	return ret
}
