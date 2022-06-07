package bindings

import "gitlab.com/elixxir/client/api"

// DownloadAndVerifySignedNdfWithUrl retrieves the NDF from a specified URL.
// The NDF is processed into a protobuf containing a signature which
// is verified using the cert string passed in. The NDF is returned as marshaled
// byte data which may be used to start a client.
func DownloadAndVerifySignedNdfWithUrl(url, cert string) ([]byte, error) {
	return api.DownloadAndVerifySignedNdfWithUrl(url, cert)
}