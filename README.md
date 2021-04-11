# MariaDB running on EdgelessRT

```bash
git clone https://github.com/edgelesssys/emariadb.git
```

## Build emariadbd

```bash
mkdir build
cd build
cmake ..
make -j 8
```

## Run MariaDB server

### Setup MariaDB

```bash
cd build
mariadb/scripts/mysql_install_db --srcdir=../server/ --auth-root-authentication-method=normal
```

```bash
cd build
erthost enclave.signed --datadir=./data --default-storage-engine=rocksdb
```
