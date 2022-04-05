package store

// SentRequestHandler allows the lower fevel to assign and remove services
type SentRequestHandler interface {
	Add(sr *SentRequest)
	Delete(sr *SentRequest)
}
