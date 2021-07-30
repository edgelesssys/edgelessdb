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
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/edgelesssys/edgelessdb/edb/rt"
	"github.com/edgelesssys/ego/ecrypto"
)

// PersistenceDir holds the directory name where we store the seal key on the host filesystem when running standalone
const PersistenceDir = "edb-persistence"

// ERocksDBMasterKeyVar is the name of the environment variable holding the master key for eRocksDB.
// Needs to be kept in sync with 3rdparty/edgeless-rocksdb/file/encrypted_file.cc
const ERocksDBMasterKeyVar = "EROCKSDB_MASTERKEY"

// sealedKeyFname is the filename where the key used for the database is stored encrypted on the disk with the SGX product key
const sealedKeyFname = "sealed_key"

// ErrKeyIncorrectSize is an error type returned when the key used by ERocksDB is not 16 bytes (= 128 bit) long
var ErrKeyIncorrectSize = errors.New("key is not 16 bytes long")

// ErrKeyNotProvidedMarblerun is an error type thrown when edb was run as a Marble, but Marblerun did not provide a key in the environment
var ErrKeyNotProvidedMarblerun = errors.New("marblerun did not set required key for edb")

// ErrKeyNotAllowedToChangeMarblerun is an error type thrown when edb attempts to change the sealing key provided by Marblerun
var ErrKeyNotAllowedToChangeMarblerun = errors.New("cannot change sealing key when running under marblerun")

func (c *Core) loadMasterKeyFromEnv() ([]byte, error) {
	keyHex, ok := os.LookupEnv(ERocksDBMasterKeyVar)
	if !ok {
		return nil, nil
	}

	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, err
	} else if len(key) != 16 {
		return nil, errors.New("key in environment has incorrect size")
	}

	return key, nil
}

func (c *Core) loadMasterKey() ([]byte, error) {
	// If key was already set, return it from env
	key, err := c.loadMasterKeyFromEnv()
	if err != nil {
		return nil, err
	}
	if key != nil {
		return key, nil
	}

	// If running as a Marble, we force Marblerun to provide the key & handle recovery
	if c.isMarble {
		return nil, ErrKeyNotProvidedMarblerun
	}

	// If no key was set yet, try to read from disk
	key, err = c.fs.ReadFile(filepath.Join(c.cfg.DataPath, PersistenceDir, sealedKeyFname))
	if err != nil {
		return nil, err
	}

	// If key was set, unseal from disk
	if c.rt.IsEnclave() {
		key, err = ecrypto.Unseal(key)
		if err != nil {
			return nil, err
		}
	}

	// This should not happen as it should have been only stored on disk when it was 16 bytes long, but let's be safe here...
	if len(key) != 16 {
		return nil, ErrKeyIncorrectSize
	}

	if err := c.storeMasterKeyToEnv(key); err != nil {
		return nil, err
	}

	// Return unsealed key
	return key, nil
}

func (c *Core) newMasterKey() ([]byte, error) {
	// Generate new random key
	key := make([]byte, 16)
	n, err := rand.Read(key)
	if err != nil {
		return nil, err
	} else if n != 16 {
		return nil, ErrKeyIncorrectSize
	}

	// Save & set new key
	if err := c.storeMasterKey(key); err != nil {
		return nil, err
	}

	return key, nil
}

func (c *Core) storeMasterKeyToEnv(key []byte) error {
	// Set newly generated key for eRocksDB
	if len(key) != 16 {
		return ErrKeyIncorrectSize
	}

	return os.Setenv(ERocksDBMasterKeyVar, hex.EncodeToString(key))
}

func (c *Core) storeMasterKey(key []byte) error {
	// Set newly generated key for eRocksDB
	if err := c.storeMasterKeyToEnv(key); err != nil {
		return err
	}

	// Save master key
	if c.rt.IsEnclave() {
		var err error
		key, err = ecrypto.SealWithProductKey(key)
		if err != nil {
			return err
		}
	}

	// Create dir
	if err := os.MkdirAll(filepath.Join(c.cfg.DataPath, PersistenceDir), 0700); err != nil {
		return err
	}

	// If there already is an existing key file stored on disk, save it
	fname := filepath.Join(c.cfg.DataPath, PersistenceDir, sealedKeyFname)
	if sealedKeyData, err := c.fs.ReadFile(fname); err == nil {
		t := time.Now()
		newFileName := fname + "_" + t.Format("20060102150405") + ".bak"
		c.fs.WriteFile(newFileName, sealedKeyData, 0600)
	}
	// Write the sealed encryption key to disk
	if err := c.fs.WriteFile(fname, key, 0600); err != nil {
		return err
	}

	return nil
}

func (c *Core) setMasterKey(key []byte) error {
	// Not allowed to change keys when running under Marblerun
	if c.isMarble {
		return ErrKeyNotAllowedToChangeMarblerun
	}

	c.masterKey = key
	return c.storeMasterKey(key)
}

func (c *Core) mustInitMasterKey() {
	// Try to load from env or file.
	key, err := c.loadMasterKey()
	// Does not exist? Generate a new one.
	if os.IsNotExist(err) {
		key, err = c.newMasterKey()
		if err != nil {
			panic(err)
		}
	} else if err == ErrKeyNotProvidedMarblerun {
		panic(err)
	}
	// Failed to read/decrypt? Enter recovery.
	if err != nil {
		rt.Log.Println("Failed to initialize master key:", err)
		rt.Log.Println("Entering recovery mode...")
		c.advanceState(stateRecovery)
		return
	}
	c.advanceState(stateInitialized)
	c.masterKey = key
}
