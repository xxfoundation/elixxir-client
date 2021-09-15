package reception

import (
	"fmt"
	"gitlab.com/elixxir/client/storage/rounds"
	"strconv"
	"strings"
)

type IdentityUse struct {
	Identity

	// Denotes if the identity is fake, in which case we do not process messages
	Fake bool

	UR *rounds.UnknownRounds
	ER *rounds.EarliestRound
	CR *rounds.CheckedRounds
}

func (iu IdentityUse) GoString() string {
	str := make([]string, 0, 7)

	str = append(str, "Identity:"+iu.Identity.GoString())
	str = append(str, "StartValid:"+iu.StartValid.String())
	str = append(str, "EndValid:"+iu.EndValid.String())
	str = append(str, "Fake:"+strconv.FormatBool(iu.Fake))
	str = append(str, "UR:"+fmt.Sprintf("%+v", iu.UR))
	str = append(str, "ER:"+fmt.Sprintf("%+v", iu.ER))
	str = append(str, "CR:"+fmt.Sprintf("%+v", iu.CR))

	return "{" + strings.Join(str, ", ") + "}"
}
