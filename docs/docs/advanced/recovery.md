# Recovery

:::note

Recovery is only available when EdgelessDB is running standalone, and only when a recovery key has been defined in the manifest. When used with MarbleRun, consider using its [recovery feature](https://docs.edgeless.systems/marblerun/features/recovery).

:::

Persistent storage for confidential applications requires a bit of attention.
By design, SGX sealing keys are unique to a single CPU, which means using the default SGX sealing methods has some caveats.
For example, sealing data while running on one host could mean the data can't be unsealed when running on another host later on.

EdgelessDB generates a *master key* for encryption. This key is then sealed to disk. When scheduled on the same CPU, EdgelessDB unseals the master key and thus restarts autonomously. However, when EdgelessDB is moved to another physical host, it enters recovery mode and waits for the master key to be passed over the HTTP REST API.

To obtain the master key, [the manifest allows for specifying a designated *recovery key*](../reference/manifest.md). The recovery key is a public RSA key. During the initial upload of the manifest, EdgelessDB returns the master key RSA-encrypted with the public key specified in the manifest.

:::caution

The holder of the corresponding private key can recover the database and, therefore, access the contents of the database. It's important that this key is kept secure. After the initial upload of the manifest, EdgelessDB won't release the master key.

:::

## Adding a recovery key to the manifest
Generate an RSA key pair:
```bash
openssl genrsa -out private.pem 3072
openssl rsa -in private.pem -pubout -out public.pem
```

Escape the line breaks of the public key:
```bash
awk 1 ORS='\\n' public.pem
```

Copy the escaped public key into the manifest:
```json
{
    ...
    "recovery": "-----BEGIN PUBLIC KEY-----\n...\n------END PUBLIC KEY-----\n"
}
```

## Performing the recovery
If EdgelessDB is unable to unseal the master key upon launch, it will enter recovery mode. You will have to upload the key via the `/recover` endpoint of the HTTP REST API.

To do so, you need to:
* Get the temporary root certificate (valid only during recovery mode)
* Decode the Base64 encoded output that was returned to you during the upload of the manifest
* Decrypt the decoded output with the corresponding RSA private key of the key defined in the manifest
* Upload the binary decoded and decrypted key to the `/recover` endpoint

Assuming you saved the output from the manifest upload step in a file called `master_key`, perform recovery like this:

```bash
era -c edgelessdb-sgx.json -h localhost:8080 -output-root edb_temp.pem
base64 -d master_key \
  | openssl pkeyutl -inkey private.pem -decrypt \
    -pkeyopt rsa_padding_mode:oaep -pkeyopt rsa_oaep_md:sha256 \
  | curl --cacert edb_temp.pem --data-binary @- https://localhost:8080/recover
```

On success this should give the following output:
```shell-session
{"status":"success","data":"Recovery successful."}
```

## Resetting EdgelessDB
If you choose not to recover the current state of the database, you can reset EdgelessDB to a clean state by deleting its data directory.
