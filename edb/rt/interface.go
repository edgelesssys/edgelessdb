package rt

// Runtime is an enclave runtime.
type Runtime interface {
	// GetRemoteReport gets a report signed by the enclave platform for use in remote attestation.
	GetRemoteReport(reportData []byte) ([]byte, error)

	// GetProductSealKey gets a key derived from the signer and product id of the enclave.
	GetProductSealKey() ([]byte, error)
}
