package auth

type SentRequestHandler interface {
	Add(sr *SentRequest)
	Delete(sr *SentRequest)
}
