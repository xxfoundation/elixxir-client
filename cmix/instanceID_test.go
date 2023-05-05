////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmix

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/ekv"
)

// TestInstanceID performs basic smoke testing of the type
func TestInstanceID(t *testing.T) {
	rng := rand.New(rand.NewSource(8675309))

	kv := ekv.MakeMemstore()
	generated, err := InitInstanceID(kv, rng)
	require.NoError(t, err)
	require.Equal(t, instanceIDLength, len(generated[:]))

	gStr := generated.String()

	require.NoError(t, err)

	loaded, err := GetInstanceID(kv)
	require.NoError(t, err)
	require.Equal(t, generated, loaded)

	loadedStr := fmt.Sprintf("%s", loaded)
	require.Equal(t, gStr, loadedStr)

	obj := versioned.Object{
		Data:      []byte("invalid"),
		Version:   0,
		Timestamp: time.Now(),
	}
	kv.Set(instanceIDKey, &obj)
	invalid, err := GetInstanceID(kv)
	require.Error(t, err)
	require.Equal(t, InstanceID{}, invalid)

	kv.Delete(instanceIDKey)
	invalid2, err := GetInstanceID(kv)
	require.Error(t, err)
	require.Equal(t, InstanceID{}, invalid2)

	obj.Data = []byte("")
	kv.Set(instanceIDKey, &obj)
	invalid3, err := GetInstanceID(kv)
	require.Error(t, err)
	require.Equal(t, InstanceID{}, invalid3)
}
