package contact

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"strings"
)

type FactList []Fact

func (fl FactList) Stringify() string {
	stringList := make([]string, len(fl))
	for index, f := range fl {
		stringList[index] = f.Stringify()
	}

	return strings.Join(stringList, factDelimiter) + factBreak
}

// unstrignifys facts followed by a facts break and with arbatrary data
// atttached at the end
func UnstringifyFactList(s string) ([]Fact, string, error) {
	parts := strings.SplitN(s, factBreak, 1)
	if len(parts) != 2 {
		return nil, "", errors.New("Invalid fact string passed")
	}
	factStrings := strings.Split(parts[0], factDelimiter)

	var factList []Fact
	for _, fString := range factStrings {
		fact, err := UnstringifyFact(fString)
		if err != nil {
			jww.WARN.Printf("Fact failed to unstringify, dropped: %s",
				err)
		} else {
			factList = append(factList, fact)
		}

	}
	return factList, parts[1], nil
}