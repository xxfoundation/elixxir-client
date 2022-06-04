package store

// SentRequestHandler allows the lower level to assign and remove services
type SentRequestHandler interface {
	Add(sr *SentRequest)
	Delete(sr *SentRequest)
}
