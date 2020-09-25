package fact

import "fmt"

type Type uint8

const (
	Username Type = 0
	Email    Type = 1
	Phone    Type = 2
)

func (t Type) String() string {
	switch t {
	case Username:
		return "Username"
	case Email:
		return "Email"
	case Phone:
		return "Phone"
	default:
		return fmt.Sprintf("Unknown Fact Type: %d", t)
	}
}

func (t Type) IsValid() bool {
	return t == Username || t == Email || t == Phone
}
