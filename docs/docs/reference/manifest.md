# Manifest
The manifest defines the initial database state. Clients can verify that an EdgelessDB instance was initialized with a specific manifest. Here's a sample manifest:
```json
{
    "sql": [
        "CREATE USER reader REQUIRE ISSUER '/CN=Owner CA' SUBJECT '/CN=Reader'",
        "CREATE USER writer REQUIRE ISSUER '/CN=Owner CA' SUBJECT '/CN=Writer'",
        "CREATE DATABASE test",
        "CREATE TABLE test.data (i INT)",
        "GRANT SELECT ON test.data TO reader",
        "GRANT INSERT ON test.data TO writer"
    ],
    "ca": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----\n",
    "debug": false,
    "recovery": "-----BEGIN PUBLIC KEY-----\n...\n------END PUBLIC KEY-----\n"
}
```

`sql` is a list of SQL statements that define the initial state of the database. They're executed once during initialization.

`ca` is a CA certificate in PEM format with escaped line breaks. It's used to verify user certificates. The user certificates therefore must be signed with the CA's private key. You can also sign user certificates by different CAs and concatenate the CA certificates.

`debug` (optional) enables the use of the debug logging [configuration](configuration.md) options. Note that this could leak data, so it's disabled by default.

`recovery` (optional) holds an RSA public key in PEM format with escaped line breaks. If set, EdgelessDB will return the master key RSA-encrypted with this key when setting the manifest. Use it to perform [recovery](../advanced/recovery.md) after the host machine was changed.
