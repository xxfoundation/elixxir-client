package key

type fingerprintAccess interface {
	// Receives a list of fingerprints to add. Overrides on collision.
	add([]*Key)
	// Receives a list of fingerprints to delete. Ignores any not available Keys
	remove([]*Key)
}
