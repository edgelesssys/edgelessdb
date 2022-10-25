# EdgelessDB demo walkthrough
This demo shows how to initialize EdgelessDB with a manifest and how users verify and connect to EdgelessDB.

Familiarize yourself with the EdgelessDB [concepts](https://docs.edgeless.systems/edgelessdb/getting-started/concepts) before proceeding.

We consider these roles:
* The *owner* creates the manifest and initializes EdgelessDB
* *Readers* can read data from a set of tables
* *Writers* can write to a set of tables (but not read)

Prerequisites:
* EdgelessDB is [installed](https://docs.edgeless.systems/edgelessdb/getting-started/install) and running
* [era](https://github.com/edgelesssys/era) and [mysql-client](https://packages.ubuntu.com/focal/mysql-client) are installed

## Generate keys and certificates
EdgelessDB identifies clients based on their X.509 certificates. The owner includes the CA certificate(s) of the client certificates in the manifest.

Generate all required keys and certificates for this demo by:
```sh
./genkeys.sh
```

## Owner: Initialize EdgelessDB with the manifest
The `genkeys.sh` script also adds the CA certificate to `manifest.json`, yielding this manifest:
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
    "ca": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----\n"
}
```

When you initialize EdgelessDB with this manifest, EdgelessDB will use `ca` to verify client certificates and execute the `sql` statements. The initial state of the database will thus consist of the table `test.data` and two users `reader` and `writer`.

Get the attested EdgelessDB root certificate and initialize EdgelessDB with the manifest:
```
cd owner
era -c ../edgelessdb-sgx.json -h localhost:8080 -output-root edb.pem
curl --cacert edb.pem --data-binary @../manifest.json https://localhost:8080/manifest
```

## Writer: Verify EdgelessDB, connect with mysql, and add data
Get the attested EdgelessDB root certificate and verify the manifest signature:
```
$ cd ../writer

$ era -c ../edgelessdb-sgx.json -h localhost:8080 -output-root edb.pem
Root certificate written to edb.pem

$ curl --cacert edb.pem https://localhost:8080/signature
5a646b895b1ead16ae16530e54180267de441ccd0198889471a5713a4a679c23

$ sha256sum ../manifest.json
5a646b895b1ead16ae16530e54180267de441ccd0198889471a5713a4a679c23  ../manifest.json
```

Note that the hash sums are equal. This proves that EdgelessDB has been initialized with this manifest.

Connect with `mysql` and add data:
```
$ mysql -h127.0.0.1 -uwriter --ssl-ca edb.pem --ssl-cert cert.pem --ssl-key key.pem

mysql> INSERT INTO test.data values (2),(5);
Query OK, 2 rows affected (0,01 sec)

mysql> SELECT * FROM test.data;
ERROR 1142 (42000): SELECT command denied to user 'writer'@'127.0.0.1' for table 'data'

mysql> quit
```

Note that the writer can insert data, but not read it.

## Reader: Verify EdgelessDB, connect with mysql, and use data
First, get the attested EdgelessDB root certificate and verify the manifest signature like you did for the writer. Then connect with `mysql` and use the data:
```
$ cd ../reader

$ era -c ../edgelessdb-sgx.json -h localhost:8080 -output-root edb.pem

[... manifest verification ...]

$ mysql -h127.0.0.1 -ureader --ssl-ca edb.pem --ssl-cert cert.pem --ssl-key key.pem

mysql> INSERT INTO test.data values (7);
ERROR 1142 (42000): INSERT command denied to user 'reader'@'127.0.0.1' for table 'data'

mysql> SELECT * FROM test.data;
+------+
| i    |
+------+
|    2 |
|    5 |
+------+
2 rows in set (0,01 sec)

mysql> SELECT AVG(i) FROM test.data;
+--------+
| AVG(i) |
+--------+
| 3.5000 |
+--------+
1 row in set (0,01 sec)

mysql> quit
```

Note that the reader can't insert data, but only read it.
