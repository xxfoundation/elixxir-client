package store

type SentRequestHandler interface {
	Add(sr *SentRequest)
	Delete(sr *SentRequest)
}
