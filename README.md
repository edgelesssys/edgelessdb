# MariaDB running on EdgelessRT

```bash
git clone --recurse-submodules https://github.com/edgelesssys/emariadb.git
```

## Build libmysqld.a

```bash
./build-mariadb.sh
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
../server/build/scripts/mysql_install_db --srcdir=../server/ --datadir=.
```

```bash
cd build
./emariadbd
```