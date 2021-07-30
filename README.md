<img src="logo.svg" alt="logo" width="40%">

# Introduction

[![Unit Tests][unit-tests-badge]][unit-tests]
[![GitHub license][license-badge]](LICENSE)
[![Discord Chat][discord-badge]][discord]

[EdgelessDB](https://edgeless.systems/products/edgelessdb) (EDB) is a MySQL-compatible database for [confidential computing](https://confidentialcomputing.io) (CC). It's based on [MariaDB](https://github.com/MariaDB/server) with [MyRocks](https://mariadb.com/kb/en/myrocks) storage engine.

## Key features
* Always encrypted: in addition to authenticated encryption on disk, the data is also encrypted in memory at runtime.
* Manifest: defines the initial database state. This enables [new kinds of applications](https://edgeless.systems/products/edgelessdb).
* Remote attestation: proves that the EDB instance runs in a secure enclave and enforces the manifest.

For details see [concepts](https://docs.edgeless.systems/edgelessdb/#/getting-started/concepts).

## Getting started
Run EDB on an SGX-capable system:
```sh
docker run --name my-edb -p3306:3306 -p8080:8080 --privileged -v /dev/sgx:/dev/sgx -t ghcr.io/edgelesssys/edgelessdb-sgx-1gb
```

Or try it in simulation mode on any system:
```sh
docker run --name my-edb -p3306:3306 -p8080:8080 -e OE_SIMULATION=1 -t ghcr.io/edgelesssys/edgelessdb-sgx-1gb
```

You may want to start with [using EDB as a high-security SQL database](https://docs.edgeless.systems/edgelessdb/#/getting-started/quickstart-sgx) in a possibly untrusted environment.

Or [check out the demo](demo) to see how EDB's CC features can be used for secure multi-party data processing.

## Documentation
See [the docs](https://docs.edgeless.systems/edgelessdb) for details on EDB concepts, configuration, and usage.

## Contribute
Read [CONTRIBUTING.md](CONTRIBUTING.md) for information on issue reporting, code guidelines, and our PR process.

[BUILD.md](BUILD.md) includes general information on how to work in this repo.

<!-- refs -->
[unit-tests]: https://github.com/edgelesssys/edgelessdb/actions
[unit-tests-badge]: https://github.com/edgelesssys/edgelessdb/workflows/Unit%20Tests/badge.svg
[license-badge]: https://img.shields.io/github/license/edgelesssys/edgelessdb
[discord]: https://discord.gg/rH8QTH56JN
[discord-badge]: https://img.shields.io/badge/chat-on%20Discord-blue
