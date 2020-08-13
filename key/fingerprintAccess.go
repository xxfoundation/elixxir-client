package key

type FingerprintAccess interface {
	AddFingerprints([]*Key) error
	//recieves a list of fingerprints
	RemoveFingerprints([]*Key) error
}
