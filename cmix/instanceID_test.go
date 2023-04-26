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
	generated, err := NewRandomInstanceID(rng)

	require.NoError(t, err)
	require.Equal(t, instanceIDLength, len(generated[:]))

	gStr := generated.String()

	kv := versioned.NewKV(ekv.MakeMemstore())
	err = StoreInstanceID(generated, kv)
	require.NoError(t, err)

	loaded, err := LoadInstanceID(kv)
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
	invalid, err := LoadInstanceID(kv)
	require.Error(t, err)
	require.Equal(t, InstanceID{}, invalid)

	kv.Delete(instanceIDKey, 0)
	invalid2, err := LoadInstanceID(kv)
	require.Error(t, err)
	require.Equal(t, InstanceID{}, invalid2)

	obj.Data = []byte("")
	kv.Set(instanceIDKey, &obj)
	invalid3, err := LoadInstanceID(kv)
	require.Error(t, err)
	require.Equal(t, InstanceID{}, invalid3)
}
