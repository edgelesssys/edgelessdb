cmake_minimum_required(VERSION 3.11)
project(edb)

if (NOT CMAKE_BUILD_TYPE)
  set(CMAKE_BUILD_TYPE Debug)
endif ()

include(ExternalProject)
include(ProcessorCount)

execute_process(
  COMMAND git submodule update --init --depth=1 --recursive server
  WORKING_DIRECTORY ${CMAKE_SOURCE_DIR})

ProcessorCount(NPROC)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_EXTENSIONS OFF)
add_compile_options(-Wall -Wextra -pedantic -Werror)

ExternalProject_Add(mariadb
  URL ${CMAKE_SOURCE_DIR}/server
  SOURCE_DIR mariadb-src
  PATCH_COMMAND patch -p1 < ${CMAKE_SOURCE_DIR}/mariadb.patch
  BINARY_DIR mariadb
  CMAKE_ARGS
    -DCMAKE_BUILD_TYPE=${CMAKE_BUILD_TYPE}
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
  BUILD_COMMAND make -j ${NPROC} mariadbd my_print_defaults resolveip
  INSTALL_COMMAND "")

set(MARIADB ${CMAKE_BINARY_DIR}/mariadb)

add_executable(emariadbd emariadbd.cpp)
add_dependencies(emariadbd mariadb)

target_link_libraries(emariadbd -static-libgcc -static-libstdc++
  ${MARIADB}/sql/libsql.a
  ${MARIADB}/sql/libsql_builtins.a
  ${MARIADB}/vio/libvio.a
  ${MARIADB}/extra/pcre2/src/pcre2-build/libpcre2-8.a
  ${MARIADB}/tpool/libtpool.a
  libcrypt.a
  ${MARIADB}/storage/maria/libaria.a
  ${MARIADB}/mysys_ssl/libmysys_ssl.a
  libssl.a
  libcrypto.a
  ${MARIADB}/sql/libsql_sequence.a
  ${MARIADB}/sql/libwsrep.a
  ${MARIADB}/storage/csv/libcsv.a
  ${MARIADB}/storage/heap/libheap.a
  ${MARIADB}/storage/myisam/libmyisam.a
  ${MARIADB}/mysys/libmysys.a
  ${MARIADB}/dbug/libdbug.a
  ${MARIADB}/strings/libstrings.a
  libz.a
  m
  ${MARIADB}/storage/myisammrg/libmyisammrg.a
  ${MARIADB}/plugin/type_geom/libtype_geom.a
  ${MARIADB}/plugin/type_inet/libtype_inet.a
  ${MARIADB}/plugin/userstat/libuserstat.a
  ${MARIADB}/wsrep-lib/src/libwsrep-lib.a
  pthread
  dl
  ${MARIADB}/wsrep-lib/wsrep-API/libwsrep_api_v26.a
  ${MARIADB}/storage/rocksdb/librocksdb_se.a
  ${MARIADB}/storage/rocksdb/librocksdb_aux_lib.a
  ${MARIADB}/storage/rocksdb/librocksdblib.a
  liblz4.a
  rt
)
