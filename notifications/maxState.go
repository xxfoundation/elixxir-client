package notifications

import jww "github.com/spf13/jwalterweatherman"

func (m *manager) SetMaxState(maxState NotificationState) error {
	jww.WARN.Printf("SetMaxState not implemented")
	return nil
}

func (m *manager) GetMaxState(maxState NotificationState) {
	jww.WARN.Printf("GetMaxState not implemented")
}
