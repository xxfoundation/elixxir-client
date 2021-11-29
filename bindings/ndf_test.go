///////////////////////////////////////////////////////////////////////////////
// Copyright © 2021 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"fmt"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/xx_network/primitives/ndf"
	"gitlab.com/xx_network/primitives/utils"
	"strings"
	"testing"
)

var testCert = `-----BEGIN CERTIFICATE-----
MIIFtjCCA56gAwIBAgIJAN45cZS2HXjNMA0GCSqGSIb3DQEBCwUAMIGMMQswCQYD
VQQGEwJVUzELMAkGA1UECAwCQ0ExEjAQBgNVBAcMCUNsYXJlbW9udDEQMA4GA1UE
CgwHRWxpeHhpcjEUMBIGA1UECwwLRGV2ZWxvcG1lbnQxEzARBgNVBAMMCmVsaXh4
aXIuaW8xHzAdBgkqhkiG9w0BCQEWEGFkbWluQGVsaXh4aXIuaW8wHhcNMjAxMjIz
MjIzMjM5WhcNMjIxMjIzMjIzMjM5WjCBjDELMAkGA1UEBhMCVVMxCzAJBgNVBAgM
AkNBMRIwEAYDVQQHDAlDbGFyZW1vbnQxEDAOBgNVBAoMB0VsaXh4aXIxFDASBgNV
BAsMC0RldmVsb3BtZW50MRMwEQYDVQQDDAplbGl4eGlyLmlvMR8wHQYJKoZIhvcN
AQkBFhBhZG1pbkBlbGl4eGlyLmlvMIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIIC
CgKCAgEAwEfurttMKDZw0VbDGIfqzKqx9Ic+B3auqqzRF6Ggfh/aJThxENnZ1Vt4
9VZ4OhaBoo4JXZwNfhFoGdjjPBdIHTCjiUueLd2eL0L/SO2QgGxPHFuemkRD1q3H
/R+Prxn6vY8vyfBpYMDr3sMzoN0GzZPgXZQJXZi66muz8J28EEVGgf+FQy/jFNPB
lQLAX/dewzwc8kcUOI81Ieo8GyrtRc3WNjqEfJZopCaSavut250zH8KHxVRXLRrd
+Bk6CjizqcLpo21A3TIAwGDSyN0aKCDrERkZvSxwk3tTKp9yxfGB7m4UpEAV6TTN
YrC/FG0E+/3dD+yNkrRNJKx5ePyz+K+bmrjb88wCWARGWxcNTYLCBIhCPc2ZiUt1
QeIezsNc9zvPYn2VH+MDV0agJ58PObNfQStGKhJ5iNRFFSavrhAFCGHgIQwBTslB
2aiw0UHKjsF0yLJJw7xM5yr739za29Mb7Lqlyc/CdwKYUdQN+vdCWFUGmZcp+coF
aaOyqtgQnTeZhQeKJIPbeC6UClkqpiKe2zewiRdblxBzuc0vgJVMFlDbr9OMML9X
7x0rni4RQRcaUAdw76dCKu0sMX6vNgMbE0nkiYG5tkG0cf59/Fhd9AE5LlwzhYLj
M4NrRr9zFfftzyKSTI7XU+C2d9g2p1qUWQLpenJugGetkGNnb80CAwEAAaMZMBcw
FQYDVR0RBA4wDIIKZWxpeHhpci5pbzANBgkqhkiG9w0BAQsFAAOCAgEAqGqT7xlh
1X7KHBev+xMBLF5WTlJbJXIHBtc6Vgi9vF5mVa1OLFczPMYNZA5rXZoYw5QBumCF
k8KvFkZOi95ZPLOypX4uJDU3qKR/qWsbTMxr9mtQlifdIBW4Ln84GzPC4e78M4Wm
oyOc1cxqmZdF+VhdEN0C1LpewlpyPuY/4UOVjvVmDmSGjcuRyxS4h01UAI6pAmYJ
U90PZAHyRAOdMtsLJg3CXj4NSBKypD0/Kmr9dpZWO9LSLmObsYfulslGsgDJHLon
CLTEfEmeA+RHp0RnFRDSPdckCBS34adHts2SpTImMbIQMnR8F8lYeMax53KpY+Nf
mRoIe4X9Knb2IyuMM/TKJ9sQMVeGgnkXcQUQ3hHSlhvfusevKePa1CuizZo77iU9
BAoiCalX6gcrSQvex+hA11rpI3HTDmC9gfIGZJhKuCmLpuaSumTtppV+rV4SqjP8
K4ytv3GFAWFpO1yYqh/q34cGGhSwcxx3SLIKmlSC4QVJwdHAqJ7/PrQ56NRaGgvu
l9Ubx1ScEBcuvfRcq84bRDtfN+zXzVRPbpy4YtjuaX50r5tKrjyS1uZfa2Ra83rj
lAeeSnLOcQxOdXT6+B8fN1vWciu2wbn/PPUwcYyUcysr7C9p3sg2zbbyks3aFTSK
pipz4Cfpkoc1Gc8xx91iBsWYBpqu4p7SXDU=
-----END CERTIFICATE-----
`

// Unit test: Download and verify NDF from hosted location.
// Ensure validity by unmarshalling NDF and checking the scheduling's cert.
func TestDownloadSignedNdf(t *testing.T) {
	// Download and verify the ndf
	content, err := DownloadAndVerifySignedNdf(testCert)
	if err != nil {
		t.Errorf("Failed to download signed NDF: %v", err)
	}
	fmt.Printf("content: %s\n", string(content))

	// Check that it is a marshallable NDF
	downloadedNdf, err := ndf.Unmarshal(content)
	if err != nil {
		t.Fatalf("Failed to unmarshal downloaded NDF: %v", err)
	}

	// Check validity of NDF
	if strings.Compare(downloadedNdf.Registration.TlsCertificate, testCert) != 0 {
		t.Fatalf("Unexpected NDF downloaded, has the spec changed?")
	}

}

// Error case: Pass in the incorrect cert forcing a verification failure.
func TestDownloadSignedNdf_Fail(t *testing.T) {
	// Load an unintended cert
	badCert, err := utils.ReadFile(testkeys.GetGatewayCertPath())
	if err != nil {
		t.Fatalf("Failed to read test certificate: %v", err)
	}
	// Download and verify with unintended cert
	_, err = DownloadAndVerifySignedNdf(string(badCert))
	if err == nil {
		t.Fatalf("Expected failure, should not be able to verify with " +
			"bad certificate")
	}
}

// Unit Test: Call DownloadAndVerifySignedNdfWithUrl with a specified URL.
// Ensure validity by unmarshalling NDF and checking the scheduling's cert.
func TestDownloadSignedNdfWithUrl(t *testing.T) {
	// todo: write test once a proper URL can be passed in
	content, err := DownloadAndVerifySignedNdfWithUrl(ndfUrl, testCert)
	if err != nil {
		t.Errorf("Failed to download signed NDF: %v", err)
	}
	fmt.Printf("content: %s\n", string(content))

	// Check that it is a marshallable NDF
	downloadedNdf, err := ndf.Unmarshal(content)
	if err != nil {
		t.Fatalf("Failed to unmarshal downloaded NDF: %v", err)
	}

	// Check validity of NDF
	if strings.Compare(downloadedNdf.Registration.TlsCertificate, testCert) != 0 {
		t.Fatalf("Unexpected NDF downloaded, has the spec changed?")
	}
}
