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

package rt

import (
	"errors"
)

// RuntimeMock is a Runtime mock.
type RuntimeMock struct{}

// IsEnclave tells the application if it is running in an enclave or not.
func (r RuntimeMock) IsEnclave() bool {
	return false
}

// GetRemoteReport gets a report signed by the enclave platform for use in remote attestation.
func (r RuntimeMock) GetRemoteReport(reportData []byte) ([]byte, error) {
	if !(0 < len(reportData) && len(reportData) <= 64) {
		return nil, errors.New("invalid data")
	}
	return []byte{2, 3, 4}, nil
}

// GetProductSealKey gets a key derived from the signer and product id of the enclave.
func (r RuntimeMock) GetProductSealKey() ([]byte, error) {
	return []byte{3, 4, 5}, nil
}

// RestartHostProcess restarts the process hosting this enclave.
func (RuntimeMock) RestartHostProcess() {
}
