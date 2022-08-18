# Build and development guide

## Build the Docker image
Generate a signing key and build the image:
```sh
openssl genrsa -out private.pem -3 3072
DOCKER_BUILDKIT=1 docker build -t edb --secret id=signingkey,src=private.pem - < Dockerfile
```

Add `--build-arg heapsize=x` where x is the desired enclave heap size in MB. By default, heap size is 1024 MB.

Add `--build-arg production=ON` to build a production enclave. By default, a debug enclave is built.

Get the [era](https://github.com/edgelesssys/era) configuration:

```sh
docker run --rm --entrypoint cat edb edgelessdb-sgx.json > edgelessdb-sgx.json
```

## Run the Docker image
You can run EdgelessDB in simulation mode on any system:
```sh
docker run -t --name my-edb -p3306:3306 -p8080:8080 -e OE_SIMULATION=1 edb
```

If your dev environment supports SGX-FLC:
```sh
docker run -t --name my-edb -p3306:3306 -p8080:8080 --device /dev/sgx_enclave --device /dev/sgx_provision edb
```

If your dev environment supports SGX without FLC:
```sh
docker run -t --name my-edb -p3306:3306 -p8080:8080 --device /dev/isgx -v /var/run/aesmd:/var/run/aesmd edb
```
Note that you'll get attestation errors on such systems.

## Nightly Docker images
Use these images to try the latest changes from the main branch:
* `ghcr.io/edgelesssys/edgelessdb-debug-1gb:nightly`
* `ghcr.io/edgelesssys/edgelessdb-debug-4gb:nightly`

## Build from source
*Prerequisites*:
* [Edgeless RT](https://github.com/edgelesssys/edgelessrt) is installed and sourced
* Go 1.17 or newer

On Ubuntu 20.04 build with:
```sh
sudo apt install bbe bison build-essential cmake liblz4-dev libssl-dev zlib1g-dev
mkdir build
cd build
cmake ..
make -j`nproc`
```

You may add the following flags to the `cmake` command:
* `-DCMAKE_BUILD_TYPE=Release` to enable optimizations.
* `-DHEAPSIZE=x` where x is the desired enclave heap size in MB. By default, heap size is 1024 MB.
* `-DPRODUCTION=ON` to build a production enclave.

### Run
After building, you can run EdgelessDB from the build directory:
```sh
./edb
```

## "not implemented" errors
If you built a debug enclave, you may get `not implemented` errors at runtime. This is because Edgeless RT doesn't implement all syscalls and POSIX functions. EdgelessDB doesn't strictly rely on the missing ones, so you can ignore the errors.

## Test
EdgelessDB tests verify basic SQL functionality and all of the additional CC features. In addition, we use MariaDB tests to ensure that we retain compatibility.

### EdgelessDB tests
```sh
cd build
ctest --output-on-failure
```

### MariaDB tests
*Prerequisite*: A fresh EdgelessDB instance with default config is running.
```sh
curl -k -d@src/test_manifest.json https://127.0.0.1:8080/manifest
cd build/mariadb
MARIADB_CC_TEST=1 MYSQL_TEST_TLS=1 ctest --output-on-failure
```

## Configuration
In addition to the [end user configuration](https://docs.edgeless.systems/edgelessdb/#/reference/configuration), the following environment variables may be useful for development:
* `EDG_EDB_DATA_PATH`: The path on the host file system where EdgelessDB will store its data. Defaults to `$PWD/data`.

## Run emariadbd
During development it may be useful to run emariadbd. This is mariadbd inside the enclave, but without the additional EdgelessDB functionality.
```sh
cd build
make emariadbd
mariadb/scripts/mysql_install_db --srcdir=../3rdparty/edgeless-mariadb --auth-root-authentication-method=normal --no-defaults
erthost emariadbd.signed --no-defaults --datadir=./data --default-storage-engine=rocksdb
```
