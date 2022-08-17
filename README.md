EdgelessDB
[![Unit Tests][unit-tests-badge]][unit-tests]
[![GitHub license][license-badge]](LICENSE)
[![Discord Chat][discord-badge]][discord]
==

<img src="src/logo.svg" alt="logo" width="40%">

[EdgelessDB](https://edgeless.systems/products/edgelessdb) is an open-source MySQL-compatible database for [confidential computing](https://confidentialcomputing.io). EdgelessDB runs entirely inside runtime-encrypted Intel SGX enclaves. In contrast to other databases, EdgelessDB ensures that all data is always encrypted—in memory as well as on disk. EdgelessDB has no storage constraints and delivers close to native performance.

Central to EdgelessDB is the concept of a *manifest*. The manifest is defined in JSON and is similar to a smart contract. It defines the initial state of the database, including access control, in an attestable way.

Architecturally, EdgelessDB is based on [MariaDB](https://github.com/MariaDB/server). As storage engine, it uses an enhanced version of [RocksDB](https://rocksdb.org/). The file encryption of EdgelessDB's storage engine is designed and built for the enclave and its very strong attacker model. In this context, EdgelessDB's storage engine provides confidentiality, integrity, freshness, auditability, and recoverability for data. Other databases, even when running inside enclaves using general-purpose frameworks, do not have these security properties.

## Use cases

1. Bring security to the next level and replace your existing database with EdgelessDB. The added security may allow you to shift sensitive databases from on-premises to the cloud.
2. Build exciting new *confidential apps* by leveraging EdgelessDB's manifest feature and security properties, for example pooling and analyzing sensitive data between multiple parties.

## Key features

* Always encrypted: in addition to authenticated encryption on disk, the data is also encrypted in memory at runtime.
* Manifest: defines the initial database state, including access control.
* Remote attestation: proves that the EdgelessDB instance runs in a secure enclave and enforces the manifest.

For details see [concepts](https://docs.edgeless.systems/edgelessdb/#/getting-started/concepts).

## Getting started

Run EdgelessDB on an SGX-capable system:
```sh
docker run -t --name my-edb -p3306:3306 -p8080:8080 --device /dev/sgx_enclave --device /dev/sgx_provision ghcr.io/edgelesssys/edgelessdb-sgx-1gb
```

Or try it in simulation mode on any system:
```sh
docker run -t --name my-edb -p3306:3306 -p8080:8080 -e OE_SIMULATION=1 ghcr.io/edgelesssys/edgelessdb-sgx-1gb
```

You may want to start with [using EdgelessDB as a high-security SQL database](https://docs.edgeless.systems/edgelessdb/#/getting-started/quickstart-sgx) in a possibly untrusted environment.

Or [check out the demo](demo) to see how EdgelessDB's confidential-computing features can be used for secure multi-party data processing.

## Documentation

See [the docs](https://docs.edgeless.systems/edgelessdb) for details on EdgelessDB concepts, configuration, and usage.

## Community & help

* Got a question? Please get in touch via [Discord][discord] or file an [issue](https://github.com/edgelesssys/edgelessdb/issues).
* If you see an error message or run into an issue, please make sure to create a [bug report](https://github.com/edgelesssys/edgelessdb/issues).
* Get the latest news and announcements on [Twitter](https://twitter.com/EdgelessSystems), [LinkedIn](https://www.linkedin.com/company/edgeless-systems/) or sign up for our monthly [newsletter](http://eepurl.com/hmjo3H).
* Visit our [blog](https://blog.edgeless.systems/) for technical deep-dives and tutorials.

## Contribute

* Read [CONTRIBUTING.md](CONTRIBUTING.md) for information on issue reporting, code guidelines, and our PR process.
* [BUILD.md](BUILD.md) includes general information on how to work in this repo.
* Pull requests are welcome! You need to agree to our [Contributor License Agreement](https://cla-assistant.io/edgelesssys/edgelessdb).
* This project and everyone participating in it are governed by the [Code of Conduct](/CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.
* To report a security issue, write to security@edgeless.systems.

<!-- refs -->
[unit-tests]: https://github.com/edgelesssys/edgelessdb/actions
[unit-tests-badge]: https://github.com/edgelesssys/edgelessdb/workflows/Unit%20Tests/badge.svg
[license-badge]: https://img.shields.io/github/license/edgelesssys/edgelessdb
[discord]: https://discord.gg/rH8QTH56JN
[discord-badge]: https://img.shields.io/badge/chat-on%20Discord-blue
