package globals

import (
	jww "github.com/spf13/jwalterweatherman"
	"os"
	"io/ioutil"
	"log"
)

// We're now logging everything to this notepad so that the CUI can replace it
// with its own notepad and get logging statements from the client
var N = jww.NewNotepad(jww.LevelError, jww.LevelWarn, os.Stdout,
	ioutil.Discard, "CLIENT", log.Ldate|log.Ltime)
