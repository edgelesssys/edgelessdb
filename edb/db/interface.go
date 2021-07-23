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

package db

import "crypto"

// Database is a secure database that can be initialized by a manifest.
type Database interface {
	// GetCertificate gets the database certificate.
	GetCertificate() ([]byte, crypto.PrivateKey)
	// Initialize sets up a database according to the jsonManifest.
	Initialize(jsonManifest []byte) error
	// Start starts the database.
	Start() error
	// GetManifestSignature returns the signature of the manifest that has been used to initialize the database.
	GetManifestSignature() []byte
}

type manifest struct {
	SQL   []string
	CA    string
	Debug bool
}
