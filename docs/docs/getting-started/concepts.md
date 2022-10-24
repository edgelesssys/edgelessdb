# Concepts

## Confidential computing
Confidential computing protects data in use by performing computations in hardware-based secure enclaves. The most prominent enclave to date is probably [Intel SGX](https://www.intel.de/content/www/de/de/architecture-and-technology/software-guard-extensions.html).
Enclaves prevent unauthorized access or modification of applications and data while in use, thereby increasing the security assurances for organizations that manage sensitive and regulated data.
For information about confidential computing, see the Confidential Computing Consortium [white paper](https://confidentialcomputing.io/white-papers/).

## Manifest
Before an instance of EdgelessDB becomes operational, it needs to be initialized with a manifest. The manifest is a simple JSON file that defines how the data stored in EdgelessDB can be accessed by different parties. Clients can (and should) verify that a given instance of EdgelessDB adheres to a certain manifest before they connect via TLS.

## Remote attestation
Remote attestation cryptographically proves that the EdgelessDB instance
* has a certain version or hash,
* runs in a secure enclave on legit hardware,
* and has been initialized with a specific manifest.

The *attestation report* (or *quote*) binds these facts to EdgelessDB's TLS certificate.
