# EDB

## Build
### Requirements
Feel free to add missing ones! This is based on our `ghcr.io/edgelesssys/edgelessrt-dev:nightly` image.
```sh
sudo apt install libncurses5-dev libcurl4-openssl-dev bison liblz4-dev bbe
```

### Build
```sh
mkdir build
cd build
cmake ..
make -j`nproc`
```

## Test
```sh
go test ./...
go test -v -tags integration ./edb -e ../build/edb
```

### Run emariadbd
```sh
cd build
make emariadbd
mariadb/scripts/mysql_install_db --srcdir=../3rdparty/mariadb --auth-root-authentication-method=normal
erthost emariadbd.signed --datadir=./data --default-storage-engine=rocksdb
```
