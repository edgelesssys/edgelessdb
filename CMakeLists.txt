cmake_minimum_required(VERSION 3.11)
project(edb VERSION 0.2.1)

if (NOT CMAKE_BUILD_TYPE)
  set(CMAKE_BUILD_TYPE Debug)
endif ()

include(ExternalProject)
include(ProcessorCount)
find_package(OpenEnclave CONFIG REQUIRED)

option(BUILD_TESTS "" ON)

set(MARIADB_CMAKE_ARGS
  -DCMAKE_BUILD_TYPE=${CMAKE_BUILD_TYPE}
  -DEDG_WITH_EDB=ON
  -DEDG_WITH_EROCKS=ON
  -DEDG_EDBDIR=${CMAKE_SOURCE_DIR}
  -DUPDATE_SUBMODULES=OFF
  -DDEFAULT_CHARSET=utf8mb4
  -DDEFAULT_COLLATION=utf8mb4_general_ci
  -DPLUGIN_ROCKSDB=STATIC
  -DPLUGIN_ARCHIVE=NO
  -DPLUGIN_AUDIT_NULL=NO
  -DPLUGIN_AUTH_0X0100=NO
  -DPLUGIN_AUTH_ED25519=NO
  -DPLUGIN_AUTH_SOCKET=NO
  -DPLUGIN_AUTH_TEST_PLUGIN=NO
  -DPLUGIN_BLACKHOLE=NO
  -DPLUGIN_CONNECT=NO
  -DPLUGIN_DAEMON_EXAMPLE=NO
  -DPLUGIN_DEBUG_KEY_MANAGEMENT=NO
  -DPLUGIN_DIALOG_EXAMPLES=NO
  -DPLUGIN_DISKS=NO
  -DPLUGIN_EXAMPLE=NO
  -DPLUGIN_EXAMPLE_KEY_MANAGEMENT=NO
  -DPLUGIN_FEDERATED=NO
  -DPLUGIN_FEDERATEDX=NO
  -DPLUGIN_FEEDBACK=NO
  -DPLUGIN_FILE_KEY_MANAGEMENT=NO
  -DPLUGIN_FTEXAMPLE=NO
  -DPLUGIN_FUNC_TEST=NO
  -DPLUGIN_HANDLERSOCKET=NO
  -DPLUGIN_INNOBASE=NO
  -DPLUGIN_LOCALES=NO
  -DPLUGIN_METADATA_LOCK_INFO=NO
  -DPLUGIN_MROONGA=NO
  -DPLUGIN_PARTITION=NO
  -DPLUGIN_PERFSCHEMA=NO
  -DPLUGIN_QA_AUTH_CLIENT=NO
  -DPLUGIN_QA_AUTH_INTERFACE=NO
  -DPLUGIN_QA_AUTH_SERVER=NO
  -DPLUGIN_QUERY_CACHE_INFO=NO
  -DPLUGIN_QUERY_RESPONSE_TIME=NO
  -DPLUGIN_S3=NO
  -DPLUGIN_SEQUENCE=NO
  -DPLUGIN_SERVER_AUDIT=NO
  -DPLUGIN_SIMPLE_PASSWORD_CHECK=NO
  -DPLUGIN_SPHINX=NO
  -DPLUGIN_SPIDER=NO
  -DPLUGIN_SQL_ERRLOG=NO
  -DPLUGIN_TEST_SQL_DISCOVERY=NO
  -DPLUGIN_TEST_SQL_SERVICE=NO
  -DPLUGIN_TEST_VERSIONING=NO
  -DPLUGIN_THREAD_POOL_INFO=NO
  -DPLUGIN_TYPE_MYSQL_JSON=NO
  -DPLUGIN_TYPE_TEST=NO
  -DPLUGIN_USER_VARIABLES=NO
  -DPLUGIN_WSREP_INFO=NO
  -DWITH_PCRE=bundled
)

if(NOT EDB_DEV)
  execute_process(
    COMMAND git submodule update --init --depth=1 --recursive 3rdparty/edgeless-mariadb 3rdparty/edgeless-rocksdb
    WORKING_DIRECTORY ${CMAKE_SOURCE_DIR})
endif()

ProcessorCount(NPROC)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_EXTENSIONS OFF)
add_compile_options(-Wall -Wextra -pedantic -Werror)
if(TIDY)
  set(CMAKE_CXX_CLANG_TIDY clang-tidy-10)
endif()

#
# build mariadb
#

if(NOT BUILD_TESTS)
  set(MARIADB_TARGETS mariadbd my_print_defaults resolveip)
endif()

ExternalProject_Add(mariadb
  SOURCE_DIR ${CMAKE_CURRENT_SOURCE_DIR}/3rdparty/edgeless-mariadb
  BUILD_ALWAYS ON
  BINARY_DIR mariadb
  CMAKE_ARGS ${MARIADB_CMAKE_ARGS}
  BUILD_COMMAND make -j ${NPROC} ${MARIADB_TARGETS}
  INSTALL_COMMAND "")

set(MARIADB ${CMAKE_BINARY_DIR}/mariadb)

set(MARIADB_LIBS
  ${MARIADB}/sql/libsql.a
  ${MARIADB}/sql/libsql_builtins.a
  ${MARIADB}/vio/libvio.a
  ${MARIADB}/extra/pcre2/src/pcre2-build/libpcre2-8.a
  ${MARIADB}/tpool/libtpool.a
  ${MARIADB}/storage/maria/libaria.a
  ${MARIADB}/mysys_ssl/libmysys_ssl.a
  ssl
  crypto
  ${MARIADB}/sql/libsql_sequence.a
  ${MARIADB}/sql/libwsrep.a
  ${MARIADB}/storage/csv/libcsv.a
  ${MARIADB}/storage/heap/libheap.a
  ${MARIADB}/storage/myisam/libmyisam.a
  ${MARIADB}/mysys/libmysys.a
  ${MARIADB}/dbug/libdbug.a
  ${MARIADB}/strings/libstrings.a
  z
  ${MARIADB}/storage/myisammrg/libmyisammrg.a
  ${MARIADB}/plugin/type_geom/libtype_geom.a
  ${MARIADB}/plugin/type_inet/libtype_inet.a
  ${MARIADB}/plugin/userstat/libuserstat.a
  ${MARIADB}/wsrep-lib/src/libwsrep-lib.a
  ${MARIADB}/wsrep-lib/wsrep-API/libwsrep_api_v26.a
  ${MARIADB}/storage/rocksdb/librocksdb_se.a
  ${MARIADB}/storage/rocksdb/librocksdb_aux_lib.a
  ${MARIADB}/storage/rocksdb/librocksdblib.a
  lz4
)

set(MARIADB_LIBC -static-libgcc -static-libstdc++
  crypt m pthread dl rt)

#
# Configure enclave.conf
#

# default size
if (NOT HEAPSIZE)
  set(HEAPSIZE 1024)
endif()
math(EXPR ENCLAVECONF_NUMHEAPPAGES "${HEAPSIZE} * 256")

set(ENCLAVECONF_DEBUG 1)
if(PRODUCTION)
  set(ENCLAVECONF_DEBUG 0)
endif()

configure_file(src/enclave.conf enclave.conf)

#
# Generate key
#

add_custom_command(
  OUTPUT private.pem
  COMMAND openssl genrsa -out private.pem -3 3072
  COMMAND openssl rsa -in private.pem -pubout -out public.pem)

add_custom_target(genkey DEPENDS ${CMAKE_BINARY_DIR}/enclave.conf private.pem)

#
# build mariadbd (statically linked) for testing purposes
#

add_executable(mariadbd 3rdparty/edgeless-mariadb/sql/main.cc)
add_dependencies(mariadbd mariadb)
set_target_properties(mariadbd PROPERTIES EXCLUDE_FROM_ALL ON)

target_link_libraries(mariadbd
  -Wl,-Bstatic ${MARIADB_LIBS} -Wl,-Bdynamic
  ${MARIADB_LIBC})

#
# build emariadbd for testing purposes
#

add_executable(emariadbd 3rdparty/edgeless-mariadb/sql/main.cc src/stubs.c)
add_dependencies(emariadbd mariadb genkey)
set_target_properties(emariadbd PROPERTIES EXCLUDE_FROM_ALL ON)

target_link_libraries(emariadbd
  openenclave::oeenclave
  openenclave::ertdeventry
  ${MARIADB_LIBS})

add_custom_command(TARGET emariadbd POST_BUILD
  COMMAND openenclave::oesign sign -e $<TARGET_FILE:emariadbd> -c
  ${CMAKE_BINARY_DIR}/enclave.conf -k private.pem)

#
# build edb-noenclave
#

add_custom_target(edb-noenclave-lib
  go build -buildmode=c-archive -o ${CMAKE_BINARY_DIR}/edb-noenclave.a
  WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}/cmd/edb)
add_executable(edb-noenclave src/invokemain.cc src/mysqld_main.cc)
add_dependencies(edb-noenclave edb-noenclave-lib mariadb)

target_link_libraries(edb-noenclave
  ${CMAKE_BINARY_DIR}/edb-noenclave.a
  -Wl,-Bstatic ${MARIADB_LIBS} -Wl,-Bdynamic
  ${MARIADB_LIBC})

#
# build edb
#

# We want to override OPENSSL_rdtsc in libcrypto. To avoid multiple definition errors, we extract the
# obj containing the original function, rename the symbol, and link the obj before linking libcrypto.
add_custom_command(OUTPUT x86_64cpuid.o
  DEPENDS /usr/lib/x86_64-linux-gnu/libcrypto.a
  COMMAND ar x /usr/lib/x86_64-linux-gnu/libcrypto.a x86_64cpuid.o)
add_custom_command(OUTPUT x86_64cpuid_.o
  DEPENDS x86_64cpuid.o
  COMMAND bbe -e 's/OPENSSL_rdtsc/OPENSSL_rdtsC/' x86_64cpuid.o > x86_64cpuid_.o)

add_library(edb-lib
  src/emain.cc
  src/syscall_file.cc
  src/syscall_handler.cc)
target_include_directories(edb-lib SYSTEM PRIVATE 3rdparty/edgeless-mariadb/include 3rdparty/edgeless-rocksdb/include)
target_link_libraries(edb-lib PRIVATE openenclave::oe_includes)

add_custom_target(edb-golib
  ${CMAKE_SOURCE_DIR}/src/build_golib.sh ${CMAKE_BINARY_DIR} ${PROJECT_VERSION}
  WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}/cmd/edb)

add_executable(edb-enclave src/mysqld_main.cc src/stubs.c x86_64cpuid_.o)
add_dependencies(edb-enclave edb-golib mariadb)

target_link_libraries(edb-enclave
  openenclave::oeenclave
  openenclave::ertcalls
  edb-lib
  openenclave::oehostepoll
  openenclave::oehostfs
  openenclave::oehostresolver
  openenclave::oehostsock
  ${CMAKE_BINARY_DIR}/edb.a
  ${MARIADB_LIBS})

add_custom_command(
  OUTPUT edb-enclave.signed
  DEPENDS edb-enclave genkey
  COMMAND openenclave::oesign sign -e $<TARGET_FILE:edb-enclave> -c ${CMAKE_BINARY_DIR}/enclave.conf -k private.pem
  COMMAND openenclave::oesign eradump -e edb-enclave.signed > edgelessdb-sgx.json)

add_custom_target(sign-edb ALL DEPENDS edb-enclave.signed)

configure_file(src/edb edb COPYONLY)

#
# tests
#

if(BUILD_TESTS)
  add_executable(syscall_test
    src/syscall_test.cc
    src/syscall_file.cc
    src/syscall_handler.cc)
  target_compile_options(syscall_test PRIVATE -fsanitize=address,undefined -fno-sanitize-recover=all)
  target_link_options(syscall_test PRIVATE -fsanitize=address,undefined)
  target_link_libraries(syscall_test openenclave::oe_includes)

  enable_testing()
  add_test(NAME unit-tests COMMAND go test -race -count=3 ./... WORKING_DIRECTORY ${CMAKE_SOURCE_DIR})
  add_test(syscall-test syscall_test)
  add_test(NAME integration-noenclave COMMAND go test -v -tags integration ./edb -e ${CMAKE_BINARY_DIR}/edb-noenclave WORKING_DIRECTORY ${CMAKE_SOURCE_DIR})
  add_test(NAME integration COMMAND go test -v -tags integration ./edb -e ${CMAKE_BINARY_DIR}/edb -sgx-config ${CMAKE_BINARY_DIR}/edgelessdb-sgx.json WORKING_DIRECTORY ${CMAKE_SOURCE_DIR})
endif()
