/* Copyright (c) Edgeless Systems GmbH

   This program is free software; you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation; version 2 of the License.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program; if not, write to the Free Software
   Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1335  USA */

package core

import (
	"encoding/hex"
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/edgelesssys/edb/edb/db"
	"github.com/edgelesssys/edb/edb/rt"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadMasterKeyFromEnv(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// Start with a clear environment
	core, _ := newCoreWithMocks()
	os.Clearenv()
	defer os.Clearenv()

	// Check if we don't get a key back with a clear environment
	key, err := core.loadMasterKeyFromEnv()
	require.NoError(err)
	assert.Nil(key)

	// Set a key, check if we get it back
	os.Setenv(ERocksDBMasterKeyVar, "4142434445464748494A4B4C4D4E4F50")
	key, err = core.loadMasterKeyFromEnv()
	require.NoError(err)
	assert.Equal([]byte{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P'}, key)
}

func TestLoadMasterKey(t *testing.T) {
	assert := assert.New(t)

	// Start with a clear environment
	core, _ := newCoreWithMocks()
	os.Clearenv()
	defer os.Clearenv()

	// Load master key from file
	readKey, err := core.loadMasterKey()
	assert.NoError(err)
	assert.Equal(core.masterKey, readKey)
}

func TestNewMasterKey(t *testing.T) {
	assert := assert.New(t)

	// Start with a clear environment
	core, tempPath := newCoreWithMocks()
	os.Clearenv()
	defer os.Clearenv()

	// Generate a key
	key, err := core.newMasterKey()
	assert.NoError(err)

	// Check if a key file was created
	readKey, err := core.fs.ReadFile(filepath.Join(tempPath, PersistenceDir, sealedKeyFname))
	assert.NoError(err)
	assert.Equal(key, readKey)
}

func TestStoreMasterKeyToEnv(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// Start with a clear environment
	core, _ := newCoreWithMocks()
	os.Clearenv()
	defer os.Clearenv()

	// Store mock key in environment
	mockKey := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	assert.NoError(core.storeMasterKeyToEnv(mockKey))

	// Check if we can successfully retrieve the mock key from environment
	keyFromEnv, err := hex.DecodeString(os.Getenv(ERocksDBMasterKeyVar))
	require.NoError(err)
	assert.EqualValues(mockKey, keyFromEnv)
}

func TestStoreMasterKey(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// Start with a clear environment
	core, tempPath := newCoreWithMocks()
	os.Clearenv()
	defer os.Clearenv()

	// Set a key and check if it was written to disk
	mockKey := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	assert.NoError(core.storeMasterKey(mockKey))
	readKey, err := core.fs.ReadFile(filepath.Join(tempPath, PersistenceDir, sealedKeyFname))
	assert.NoError(err)
	assert.Equal(mockKey, readKey)
	fsInfo, err := core.fs.ReadDir(tempPath)
	require.NoError(err)
	log.Println(len(fsInfo))

	// Write a second key and check if the directory holds two files. One backup key, one new key.
	secondMockKey := []byte{4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19}
	assert.NoError(core.storeMasterKey(secondMockKey))

	// Check if we can access both keys, the new one and the old backuped one
	fsInfo, err = core.fs.ReadDir(path.Join(tempPath, PersistenceDir))
	require.NoError(err)
	require.Len(fsInfo, 2)

	newKey, err := core.fs.ReadFile(filepath.Join(tempPath, PersistenceDir, fsInfo[0].Name()))
	require.NoError(err)
	assert.Equal(secondMockKey, newKey)
	oldKey, err := core.fs.ReadFile(filepath.Join(tempPath, PersistenceDir, fsInfo[1].Name()))
	require.NoError(err)
	assert.Equal(mockKey, oldKey)
}

func TestSetMasterKey(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// Start with a clear environment
	core, tempPath := newCoreWithMocks()
	os.Clearenv()
	defer os.Clearenv()

	// Set a new key
	mockKey := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	assert.NoError(core.setMasterKey(mockKey))

	// Verify the key was set in environment
	keyFromEnv, err := hex.DecodeString(os.Getenv(ERocksDBMasterKeyVar))
	require.NoError(err)
	assert.EqualValues(mockKey, keyFromEnv)

	// Verify that the key was written to disk
	keyFromDisk, err := core.fs.ReadFile(filepath.Join(tempPath, PersistenceDir, sealedKeyFname))
	assert.NoError(err)
	assert.Equal(mockKey, keyFromDisk)
	assert.Equal(keyFromEnv, keyFromDisk)

	// Check if it fails when running under Marblerun
	core.isMarble = true
	assert.Error(core.setMasterKey(mockKey))
}

// Note: TestMustInitMasterKey cannot test entering the recovery mode, as the unit tests don't run in an enclave.
func TestMustInitMasterKey(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// Start with a clear environment
	os.Clearenv()
	defer os.Clearenv()

	// We have to manually create a Core here as NewCore automatically initializes the key
	rt := rt.RuntimeMock{}
	db := db.DatabaseMock{}
	fs := afero.Afero{Fs: afero.NewMemMapFs()}
	tempPath, err := fs.TempDir("", "")
	if err != nil {
		panic(err)
	}
	cfg := Config{DataPath: tempPath}
	core := &Core{state: stateUninitialized, cfg: cfg, rt: &rt, db: &db, fs: fs, isMarble: false}

	// Let core generate a random key when nothing was set
	core.mustInitMasterKey()

	// Verify the key was set in environment
	keyFromEnv, err := hex.DecodeString(os.Getenv(ERocksDBMasterKeyVar))
	require.NoError(err)
	assert.NotNil(keyFromEnv)

	// Verify that the key was written to disk
	keyFromDisk, err := core.fs.ReadFile(filepath.Join(tempPath, PersistenceDir, sealedKeyFname))
	assert.NoError(err)
	assert.Equal(keyFromEnv, keyFromDisk)

	// Check that we are in state initialized
	assert.Equal(stateInitialized, core.state)

	// Delete from env, forcefully reset state for the unit test
	os.Clearenv()
	core.state = stateUninitialized

	core.mustInitMasterKey()
	keyFromEnv, err = hex.DecodeString(os.Getenv(ERocksDBMasterKeyVar))
	require.NoError(err)
	assert.Equal(keyFromDisk, keyFromEnv)

	// Delete from env, forcefully reset state for the unit test, run as Marble
	// This should fail as we expect that Marblerun always provides the key in the environment, not from a file
	os.Clearenv()
	core.state = stateUninitialized
	core.isMarble = true

	// MustInitMasterKey will panic when is detected it was run as a Marble but has its key missing
	// The test execution continues in the recovery function after MustInitMasterKey screamed and panicked.
	defer func() {
		if r := recover(); r != nil {
			assert.Equal(ErrKeyNotProvidedMarblerun, r)
			keyFromEnv, err = hex.DecodeString(os.Getenv(ERocksDBMasterKeyVar))
			require.NoError(err)
			assert.Empty(keyFromEnv)
		}
	}()
	core.mustInitMasterKey()

	// Should not be executed. If it did, something went wrong.
	assert.NotNil(nil, "Test did not enter panic recovery, but was expected to.")
}
