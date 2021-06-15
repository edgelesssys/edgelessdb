package core

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/edgelesssys/ego/ecrypto"
)

// PersistenceDir holds the directory name where we store the seal key on the host filesystem when running standalone
const PersistenceDir = "edb-persistence"

// ERocksDBMasterKeyVar is the name of the environment variable holding the master key for eRocksDB.
// Needs to be kept in sync with 3rdparty/edgeless-rocksdb/file/encrypted_file.cc
const ERocksDBMasterKeyVar = "EROCKSDB_MASTERKEY"

// sealedKeyFname is the filename where the key used for the database is stored encrypted on the disk with the SGX product key
const sealedKeyFname = "sealed_key"

func (c *Core) loadMasterKeyFromEnv() ([]byte, bool, error) {
	keyHex, ok := os.LookupEnv(ERocksDBMasterKeyVar)
	if !ok {
		return nil, false, nil
	}

	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, true, err
	}

	return key, true, nil
}

func (c *Core) loadMasterKeyFromFile() ([]byte, error) {
	// If key was already set, return it from env
	key, alreadySet, err := c.loadMasterKeyFromEnv()
	if err != nil {
		return nil, err
	}
	if alreadySet {
		return key, nil
	}

	// If running as a Marble, we force Marblerun to provide the key & handle recovery
	if c.isMarble {
		return nil, errors.New("marblerun did not set required key for edb")
	}

	// If no key was set yet, try to read from disk
	sealedKey, err := ioutil.ReadFile(filepath.Join(c.cfg.DataPath, PersistenceDir, sealedKeyFname))
	if err != nil {
		return nil, err
	}

	// If key was set, unseal from disk
	if c.rt.IsEnclave() {
		key, err = ecrypto.Unseal(sealedKey)
		if err != nil {
			return nil, err
		}
	} else {
		key = sealedKey
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
		return nil, errors.New("generated key is not 16 bytes long")
	}

	// Save & set new key
	if err := c.storeMasterKeytoFile(key); err != nil {
		return nil, err
	}

	return key, nil
}

func (c *Core) storeMasterKeyToEnv(key []byte) error {
	// Set newly generated key for eRocksDB
	if err := os.Setenv(ERocksDBMasterKeyVar, hex.EncodeToString(key)); err != nil {
		return err
	}

	return nil
}

func (c *Core) storeMasterKeytoFile(key []byte) error {
	// Set newly generated key for eRocksDB
	if err := c.storeMasterKeyToEnv(key); err != nil {
		return err
	}

	// Save master key
	var storedKey []byte
	var err error
	if c.rt.IsEnclave() {
		storedKey, err = ecrypto.SealWithProductKey(key)
		if err != nil {
			return err
		}
	} else {
		storedKey = key
	}

	// Create dir
	if err := os.MkdirAll(filepath.Join(c.cfg.DataPath, PersistenceDir), 0700); err != nil {
		return err
	}

	// If there already is an existing key file stored on disk, save it
	if sealedKeyData, err := ioutil.ReadFile(filepath.Join(c.cfg.DataPath, PersistenceDir, sealedKeyFname)); err == nil {
		t := time.Now()
		newFileName := filepath.Join(c.cfg.DataPath, PersistenceDir, sealedKeyFname) + "_" + t.Format("20060102150405") + ".bak"
		ioutil.WriteFile(newFileName, sealedKeyData, 0600)
	}

	// Write the sealed encryption key to disk
	if err := ioutil.WriteFile(filepath.Join(c.cfg.DataPath, PersistenceDir, sealedKeyFname), storedKey, 0600); err != nil {
		return err
	}

	return nil
}

func (c *Core) initMasterKey() ([]byte, error) {
	// First, try to load key from env
	key, ok, err := c.loadMasterKeyFromEnv()
	if err != nil {
		return nil, err
	}

	// Does not exist? Try from disk.
	if !ok {
		// Try to load from file.
		_, err := c.loadMasterKeyFromFile()
		// Does not exist? Generate a new one.
		if os.IsNotExist(err) {
			key, err = c.newMasterKey()
		}
		// Failed to read/decrypt? Abort, maybe enter recovery.
		if err != nil {
			return nil, err
		}
	}

	return key, nil
}
