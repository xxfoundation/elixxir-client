package auth

type requestFormat struct {
	ecrFormat
	id      []byte
	facts   []byte
	message []byte
}
