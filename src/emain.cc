#include <openenclave/ert.h>
#include <sys/mount.h>

#include <cassert>
#include <cstdlib>
#include <cstring>
#include <iostream>

using namespace std;
using namespace ert;

static constexpr auto kMemfsName = "edg_memfs";

extern "C" void invokemain();

int emain() {
  if (oe_load_module_host_epoll() != OE_OK ||
      oe_load_module_host_file_system() != OE_OK ||
      oe_load_module_host_resolver() != OE_OK ||
      oe_load_module_host_socket_interface() != OE_OK) {
    cout << "oe_load_module_host failed\n";
    return EXIT_FAILURE;
  }

  const Memfs memfs(kMemfsName);
  if (mount("/", "/tmp", kMemfsName, 0, nullptr) != 0) {
    cout << "mount memfs failed\n";
    return EXIT_FAILURE;
  }

  if (mount("/", "/edg/hostfs", OE_HOST_FILE_SYSTEM, 0, nullptr) != 0) {
    cout << "mount hostfs failed\n";
    return EXIT_FAILURE;
  }

  invokemain();
  return EXIT_SUCCESS;
}

ert_args_t ert_get_args() {
  // Get args from the host.
  ert_args_t args{};
  if (ert_get_args_ocall(&args) != OE_OK || args.argc < 0 || args.envc < 0)
    abort();

  // Copy argv.
  char** argv = nullptr;
  ert_copy_strings_from_host_to_enclave(
      args.argv, &argv, static_cast<size_t>(args.argc));
  assert(argv);

  // Copy env.
  char** env = nullptr;
  ert_copy_strings_from_host_to_enclave(
      args.envp, &env, static_cast<size_t>(args.envc));
  assert(env);

  // Keep all env vars that begin with EDG_
  int edg_count = 0;
  for (int i = 0; env[i]; ++i) {
    if (memcmp(env[i], "EDG_", 4) == 0) {
      env[edg_count] = env[i];
      ++edg_count;
    }
  }
  env[edg_count] = nullptr;

  ert_args_t result{};
  result.argc = args.argc;
  result.argv = argv;
  result.envc = edg_count;
  result.envp = env;
  return result;
}
