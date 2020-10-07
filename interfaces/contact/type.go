package contact

import "fmt"

type FactType uint8

const (
	Username FactType = 0
	Email    FactType = 1
	Phone    FactType = 2
)

func (t FactType) String() string {
	switch t {
	case Username:
		return "Username"
	case Email:
		return "Email"
	case Phone:
		return "Phone"
	default:
		return fmt.Sprintf("Unknown Fact FactType: %d", t)
	}
}

func (t FactType) IsValid() bool {
	return t == Username || t == Email || t == Phone
}
