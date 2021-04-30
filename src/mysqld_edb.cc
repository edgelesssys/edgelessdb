#include <netdb.h>

#include "my_global.h"
//
#include "log.h"

static constexpr auto kEdbInternalAddr = "EDB_INTERNAL_ADDR";  // must be kept sync with edb/db/mariadb.go

extern "C" int edgeless_listen_internal_ready;
int edgeless_listen_internal_ready;

static void AbortPerror(const char* message) {
  sql_perror(message);
  unireg_abort(1);
}

void edgeless_listen_internal() {
  const char* edb_internal_addr = getenv(kEdbInternalAddr);
  if (!edb_internal_addr)
    return;

  // split addr into host and port
  std::string addr = edb_internal_addr;
  const size_t pos_colon = addr.find(':');
  if (!(1 <= pos_colon && pos_colon < addr.size() - 1))
    abort();
  addr[pos_colon] = 0;

  // get sockaddr
  addrinfo hints{};
  hints.ai_family = AF_INET;
  hints.ai_socktype = SOCK_STREAM;
  addrinfo* ai = nullptr;
  if (getaddrinfo(addr.c_str(), addr.c_str() + pos_colon + 1, &hints, &ai) != 0)
    AbortPerror("getaddrinfo");

  // create listen socket
  const auto listen_sock = mysql_socket_socket(key_socket_tcpip, AF_INET, SOCK_STREAM, 0);
  if (mysql_socket_getfd(listen_sock) == INVALID_SOCKET)
    AbortPerror("socket");
  const int opt = 1;
  const int res = mysql_socket_setsockopt(listen_sock, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof opt);
  assert(res == 0);
  if (mysql_socket_bind(listen_sock, ai->ai_addr, ai->ai_addrlen) != 0)
    AbortPerror("bind");
  freeaddrinfo(ai);
  if (mysql_socket_listen(listen_sock, 3) != 0)
    AbortPerror("listen");

  __atomic_store_n(&edgeless_listen_internal_ready, 1, __ATOMIC_SEQ_CST);

  // accept connections
  do {
    const auto accepted_sock = mysql_socket_accept(key_socket_client_connection, listen_sock, nullptr, 0);
    if (mysql_socket_getfd(accepted_sock) == INVALID_SOCKET)
      AbortPerror("accept");
    handle_accepted_socket(accepted_sock, listen_sock);

    // stop listening if env var has been cleared
    edb_internal_addr = getenv(kEdbInternalAddr);
  } while (edb_internal_addr && *edb_internal_addr);

  if (mysql_socket_close(listen_sock) != 0)
    AbortPerror("close");
}
