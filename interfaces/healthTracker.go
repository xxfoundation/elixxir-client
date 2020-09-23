package interfaces

type HealthTracker interface {
	AddChannel(chan bool)
	AddFunc(f func(bool))
	IsHealthy() bool
}
