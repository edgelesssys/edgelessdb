# Running EdgelessDB under MarbleRun

To run EdgelessDB as a service in a confidential cluster, combine it with [MarbleRun](https://marblerun.sh).

When running EdgelessDB as a Marble, secrets will be [managed by MarbleRun](https://docs.edgeless.systems/marblerun/#/features/secrets-management). EdgelessDB will no longer generate its own root certificate nor its sealing key. The root certificate for EdgelessDB needs to be defined in MarbleRun's manifest. Furthermore, EdgelessDB's own recovery method will be unavailable. MarbleRun will handle recovery for your entire cluster.

## Extend the MarbleRun manifest
To add EdgelessDB to your MarbleRun cluster, add to the [MarbleRun manifest](https://docs.edgeless.systems/marblerun/#/workflows/define-manifest)
* the `edgelessdb` package
* an encryption key `edb_masterkey`
* a root certificate `edb_rootcert`
* and a Marble `edb_marble` that applies the secrets.

Here's a template:
```json
{
    "Packages": {
        "edgelessdb": {
            "SecurityVersion": 1,
            "ProductID": 16,
            "SignerID": "67d7b00741440d29922a15a9ead427b6faf1d610238ae9826da345cea4fee0fe"
        }
    },
    "Marbles": {
        "edb_marble": {
            "Package": "edgelessdb",
            "Parameters": {
                "Env": {
                    "EROCKSDB_MASTERKEY": "{{ hex .Secrets.edb_masterkey.Private }}",
                    "EDB_ROOT_CERT": "{{ pem .Secrets.edb_rootcert.Cert }}",
                    "EDB_ROOT_KEY": "{{ pem .Secrets.edb_rootcert.Private }}"
                }
            }
        }
    },
    "Secrets": {
        "edb_masterkey": {
            "Type": "symmetric-key",
            "Size": 128
        },
        "edb_rootcert": {
            "Type": "cert-ecdsa",
            "Size": 256,
            "Cert": {
                "IsCA": true,
                "Subject": {
                    "Organization": [
                        "My EdgelessDB root"
                    ]
                }
            }
        }
    }
}
```

## Launch the MarbleRun Coordinator
[Set up the MarbleRun Coordinator](https://docs.edgeless.systems/marblerun/#/deployment/cloud?id=deploy-marblerun) and [set the MarbleRun manifest](https://docs.edgeless.systems/marblerun/#/workflows/set-manifest).

## Launch as a Marble
To run EdgelessDB as a Marble, add `-marble` as a parameter and define the required Marble definitions as environment variables:

```bash
docker run -t \
  --name my-edb \
  -p3306:3306 \
  -p8080:8080 \
  --device /dev/sgx_enclave --device /dev/sgx_provision \
  -e EDG_MARBLE_TYPE=edb_marble \
  -e EDG_MARBLE_COORDINATOR_ADDR=172.17.0.1:2001 \
  ghcr.io/edgelesssys/edgelessdb-sgx-1gb \
  -marble
```

Set `EDG_MARBLE_COORDINATOR_ADDR` to the address of your Coordinator instance. Keep `172.17.0.1` (the gateway of Docker's default network bridge) if the Coordinator runs on the same host.

## Remote attestation
When running as a Marble, you can either attest an EdgelessDB instance by itself or by attesting the whole cluster once through the MarbleRun Coordinator. Given that EdgelessDB's certificates are issued and provided by MarbleRun, you can establish trust via MarbleRun's public key infrastructure (PKI) to your EdgelessDB instances.
