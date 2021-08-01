EdgelessDB
[![Unit Tests][unit-tests-badge]][unit-tests]
[![GitHub license][license-badge]](LICENSE)
[![Discord Chat][discord-badge]][discord]
==

<img src="logo.svg" alt="logo" width="40%">

[EdgelessDB](https://edgeless.systems/products/edgelessdb) is a MySQL-compatible database for [confidential computing](https://confidentialcomputing.io). EdgelessDB runs entirely inside runtime-encrypted Intel SGX enclaves. In contrast to other databases, EdgelessDB ensures that all data is always stronlgy encrypted - in memory as well as on disk. Despite running in enclaves, EdgelessDB doesn't have storage constraints and delivers close to native performance.

Central to EdgelessDB is the concept of a *manifest*. The manifest is defined in JSON and is somewhat akin to a smart contract. The manifest defines in an attestable way the initial state of the database, including accesss control.

Architecturally, EdgelessDB is based on [MariaDB](https://github.com/MariaDB/server). As storage engine, it uses an enhanced version of [RocksDB](https://rocksdb.org/). The file encryption of EdgelessDB's storage engine is designed and built for the enclave and its very strong attacker model. In this context, EdgelessDB's storage engine provides confidentiality, integrity, freshness, auditability, and recoverability for data. Other databases (even when being run inside enclaves using general-purpose frameworks) do not have these security properties.

## Use cases

1. Bring security to the next level and replace your existing database with EdgelessDB. The added security may allow you to shift sensitive databases from the on-prem to the cloud. 
2. Build exciting new *confidential apps* by leveraging EdgelessDB's manifest feature and security properties, e.g., pool and analyze sensitive data between multiple parties.

## Key features

* Always encrypted: in addition to authenticated encryption on disk, the data is also encrypted in memory at runtime.
* Manifest: defines the initial database state. This enables [new kinds of applications](https://edgeless.systems/products/edgelessdb).
* Remote attestation: proves that the EdgelessDB instance runs in a secure enclave and enforces the manifest.

For details see [concepts](https://docs.edgeless.systems/edgelessdb/#/getting-started/concepts).

## Getting started

Run EdgelessDB on an SGX-capable system:
```sh
docker run --name my-edb -p3306:3306 -p8080:8080 --privileged -v /dev/sgx:/dev/sgx -t ghcr.io/edgelesssys/edgelessdb-sgx-1gb
```

Or try it in simulation mode on any system:
```sh
docker run --name my-edb -p3306:3306 -p8080:8080 -e OE_SIMULATION=1 -t ghcr.io/edgelesssys/edgelessdb-sgx-1gb
```

You may want to start with [using EdgelessDB as a high-security SQL database](https://docs.edgeless.systems/edgelessdb/#/getting-started/quickstart-sgx) in a possibly untrusted environment.

Or [check out the demo](demo) to see how EdgelessDB's confidential-computing features can be used for secure multi-party data processing.

## Documentation

See [the docs](https://docs.edgeless.systems/edgelessdb) for details on EdgelessDB concepts, configuration, and usage.

## Contribute

Read [CONTRIBUTING.md](CONTRIBUTING.md) for information on issue reporting, code guidelines, and our PR process.

[BUILD.md](BUILD.md) includes general information on how to work in this repo.

<!-- refs -->
[unit-tests]: https://github.com/edgelesssys/edgelessdb/actions
[unit-tests-badge]: https://github.com/edgelesssys/edgelessdb/workflows/Unit%20Tests/badge.svg
[license-badge]: https://img.shields.io/github/license/edgelesssys/edgelessdb
[discord]: https://discord.gg/rH8QTH56JN
[discord-badge]: https://img.shields.io/badge/chat-on%20Discord-blue
