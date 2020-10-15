package contact

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
)

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

func (t FactType) Stringify() string {
	switch t {
	case Username:
		return "U"
	case Email:
		return "E"
	case Phone:
		return "P"
	}
	jww.FATAL.Panicf("Unknown Fact FactType: %d", t)
	return "error"
}

func UnstringifyFactType(s string) (FactType, error) {
	switch s {
	case "U":
		return Username, nil
	case "E":
		return Email, nil
	case "P":
		return Phone, nil
	}
	return 3, errors.Errorf("Unknown Fact FactType: %s", s)
}

func (t FactType) IsValid() bool {
	return t == Username || t == Email || t == Phone
}
