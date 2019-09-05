////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import "testing"

func TestParseClientVersion_Success(t *testing.T) {
	version, err := parseClientVersion("1.2.3456")
	expectedVersion := clientVersion{
		major: 1,
		minor: 2,
		patch: "3456",
	}
	if err != nil {
		t.Error(err)
	}
	if version.minor != expectedVersion.minor {
		t.Errorf("Expected %+v for minor version, got %+v",
			expectedVersion.minor, version.minor)
	}
	if version.major != expectedVersion.major {
		t.Errorf("Expected %+v for major version, got %+v",
			expectedVersion.major, version.major)
	}
	if version.patch != expectedVersion.patch {
		t.Errorf("Expected %+v for patch version, got %+v",
			expectedVersion.patch, version.patch)
	}
}

func TestParseClientVersion_Failure(t *testing.T) {
	_, err := parseClientVersion("")
	if err == nil {
		t.Error("Expected error for empty version string")
	}
	_, err = parseClientVersion("0")
	if err == nil {
		t.Error("Expected error for version string with one number")
	}
	_, err = parseClientVersion("0.0")
	if err == nil {
		t.Error("Expected error for version string with two numbers")
	}
	_, err = parseClientVersion("a.4.0")
	if err == nil {
		t.Error("Expected error for version string with non-numeric major version")
	}
	_, err = parseClientVersion("4.a.0")
	if err == nil {
		t.Error("Expected error for version string with non-numeric minor version")
	}
}

// If the registration version starts with zeroes, anything with major version 0
// should be compatible, with any (positive) minor version and any patch
func TestClientVersion_IsCompatible_Zero(t *testing.T) {
	theirVersion := &clientVersion{
		major: 0,
		minor: 0,
		patch: "stuff",
	}
	ourVersion_compatible := &clientVersion{
		major: 0,
		minor: 1,
		patch: "even more stuff",
	}
	if !ourVersion_compatible.isCompatible(theirVersion) {
		t.Errorf("Our version %v should have been compatible with their version %v",
			ourVersion_compatible, theirVersion)
	}
	ourVersion_incompatible := &clientVersion{
		major: 1,
		minor: 0,
		patch: "other stuff",
	}
	if ourVersion_incompatible.isCompatible(theirVersion) {
		t.Errorf("Our version %v shouldn't have been compatible with their version %v",
			ourVersion_incompatible, theirVersion)
	}
}

// If the registration version is a real version (non-zero), the boundaries
// of compatibility should be enforced. That is, the major version should be the
// same, and the client's minor version should be greater than or equal to the
// registration server's minor version to be deemed compatible.
func TestClientVersion_IsCompatible_Nonzero(t *testing.T) {
	theirVersion := &clientVersion{
		major: 1,
		minor: 4,
		patch: "51",
	}
	ourVersion_compatible := []*clientVersion{{
		major: 1,
		minor: 4,
		patch: "50",
	}, {
		major: 1,
		minor: 4,
		patch: "52",
	}, {
		major: 1,
		minor: 4,
		patch: "51",
	}, {
		major: 1,
		minor: 10,
		patch: "50",
	}, {
		major: 1,
		minor: 10,
		patch: "52",
	}, {
		major: 1,
		minor: 10,
		patch: "51",
	}}
	for i := 0; i < len(ourVersion_compatible); i++ {
		if !ourVersion_compatible[i].isCompatible(theirVersion) {
			t.Errorf("Versions (ours) %v (and theirs) %v were incorrectly incompatible at index %v.",
				ourVersion_compatible[i], theirVersion, i)
		}
	}

	ourVersion_incompatible := []*clientVersion{{
		major: 0,
		minor: 0,
		patch: "0",
	}, {
		major: 2,
		minor: 0,
		patch: "1",
	}, {
		major: 1,
		minor: 3,
		patch: "9",
	}, {
		major: 2,
		minor: 5,
		patch: "9",
	}}
	for i := 0; i < len(ourVersion_incompatible); i++ {
		if ourVersion_incompatible[i].isCompatible(theirVersion) {
			t.Errorf("Versions (ours) %v (and theirs) %v were incorrectly compatible at index %v.",
				ourVersion_incompatible[i], theirVersion, i)
		}
	}
}
