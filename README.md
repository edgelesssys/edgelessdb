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

### EDB tests
```sh
cd build
ctest --output-on-failure
```

### MariaDB tests
*Prerequisite*: A fresh EDB instance with default config is running.
```sh
curl -k -d@src/test_manifest.json https://127.0.0.1:8080/manifest
cd build/mariadb
MYSQL_TEST_TLS=1 ctest --output-on-failure
```

### Run emariadbd
```sh
cd build
make emariadbd
mariadb/scripts/mysql_install_db --srcdir=../3rdparty/mariadb --auth-root-authentication-method=normal
erthost emariadbd.signed --datadir=./data --default-storage-engine=rocksdb
```

## Docker images

### Build

```sh
openssl genrsa -out private.pem -3 3072
DOCKER_BUILDKIT=1 docker build -t ghcr.io/edgelesssys/edb --secret id=signingkey,src=private.pem --secret id=repoaccess,src=$HOME/.netrc - < Dockerfile
```

### Run

SGX-FLC:
```sh
docker run --name edb -p3306:3306 -p8080:8080 --privileged -v /dev/sgx:/dev/sgx -t ghcr.io/edgelesssys/edb
```

SGX-non-FLC:
```sh
docker run --name edb -p3306:3306 -p8080:8080 --device /dev/isgx -v /var/run/aesmd:/var/run/aesmd -t ghcr.io/edgelesssys/edb
```

Simulation mode:
```sh
docker run --name edb -p3306:3306 -p8080:8080 -e OE_SIMULATION=1 -t ghcr.io/edgelesssys/edb
```
