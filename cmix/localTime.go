package cmix

import "time"

type localTime struct{}

func (localTime) NowMs() int64 {
	t := time.Now()
	return t.UnixNano() / int64(time.Millisecond)
}
