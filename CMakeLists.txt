cmake_minimum_required(VERSION 3.11)

if (NOT CMAKE_BUILD_TYPE)
  set(CMAKE_BUILD_TYPE Debug)
endif ()

include_directories(${CMAKE_SOURCE_DIR}/server/include ${CMAKE_SOURCE_DIR}/server/build/include)
link_directories(${CMAKE_SOURCE_DIR}/server/build/sql
  ${CMAKE_SOURCE_DIR}/server/build/tpool/
  ${CMAKE_SOURCE_DIR}/server/build/vio/
  ${CMAKE_SOURCE_DIR}/server/build/libmysqld
  ${CMAKE_SOURCE_DIR}/server/build/extra/pcre2/src/pcre2-build/
  ${CMAKE_SOURCE_DIR}/server/build/storage/maria/
  ${CMAKE_SOURCE_DIR}/server/build/storage/csv/
  ${CMAKE_SOURCE_DIR}/server/build/storage/heap/
  ${CMAKE_SOURCE_DIR}/server/build/storage/myisam/
  ${CMAKE_SOURCE_DIR}/server/build/storage/myisammrg/
  ${CMAKE_SOURCE_DIR}/server/build/storage/rocksdb/
  ${CMAKE_SOURCE_DIR}/server/build/mysys
  ${CMAKE_SOURCE_DIR}/server/build/mysys_ssl
  ${CMAKE_SOURCE_DIR}/server/build/dbug
  ${CMAKE_SOURCE_DIR}/server/build/strings
  ${CMAKE_SOURCE_DIR}/server/build/plugin/type_geom
  ${CMAKE_SOURCE_DIR}/server/build/plugin/type_inet
  ${CMAKE_SOURCE_DIR}/server/build/plugin/user_variables
  ${CMAKE_SOURCE_DIR}/server/build/plugin/userstat
  ${CMAKE_SOURCE_DIR}/server/build/wsrep-lib/src/
  ${CMAKE_SOURCE_DIR}/server/build/wsrep-lib/wsrep-API/
  )

add_executable(emariadbd emariadbd.cpp)

# link with embedded mariadb (sqlite style)
# ${CMAKE_SOURCE_DIR}/server/build/scripts/mysql_config --include --libmysqld-libs
# target_link_libraries(emariadbd
#     ${CMAKE_SOURCE_DIR}/server/build/libmysqld/libmariadbd.a
#     pthread
#     crypt
#     lz4
#     aio
#     z
#     m
#     ssl
#     crypto
#     dl
#     tpool
# )

target_link_libraries(emariadbd -static
  ${CMAKE_SOURCE_DIR}/server/build/sql/libsql.a
  ${CMAKE_SOURCE_DIR}/server/build/sql/libsql_builtins.a
  vio
  pcre2-8
  tpool
  aio
  crypt
  aria
  mysys_ssl
  ssl
  crypto
  sql_sequence
  wsrep
  csv
  heap
  rocksdblib
  myisam
  mysys
  dbug
  strings
  z
  m
  myisammrg
  type_geom
  type_inet
  userstat
  wsrep-lib
  pthread
  dl
  wsrep_api_v26
)
