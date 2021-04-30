package rt

import (
	"errors"
)

// RuntimeMock is a Runtime mock.
type RuntimeMock struct{}

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
