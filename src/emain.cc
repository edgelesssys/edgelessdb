/* Copyright (c) Edgeless Systems GmbH

   This program is free software; you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation; version 2 of the License.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program; if not, write to the Free Software
   Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1335  USA */

#include <openenclave/ert.h>
#include <sys/mount.h>
#include <sys/stat.h>

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

  // Preparing memfs
  const Memfs memfs(kMemfsName);
  if (mount("/", "/memfs", kMemfsName, 0, nullptr) != 0) {
    cout << "mount memfs failed\n";
    return EXIT_FAILURE;
  }
  if (mkdir("/memfs/tmp", 0777) == -1) {
    cout << "creating directory '/memfs/tmp' failed: " << strerror(errno) << endl;
    return EXIT_FAILURE;
  }
  if (mkdir("/memfs/data", 0777) == -1) {
    cout << "creating directory '/memfs/data' failed: " << strerror(errno) << endl;
    return EXIT_FAILURE;
  }
  if (umount("/memfs") != 0) {
    cout << "umount memfs failed\n";
    return EXIT_FAILURE;
  }

  // Mounting memfs for /tmp and /data
  if (mount("/tmp", "/tmp", kMemfsName, 0, nullptr) != 0) {
    cout << "mount memfs failed\n";
    return EXIT_FAILURE;
  }
  if (mount("/data", "/data", kMemfsName, 0, nullptr) != 0) {
    cout << "mount memfs failed\n";
    return EXIT_FAILURE;
  }

  // Mounting hostfs for access to config file
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

extern "C" int OPENSSL_rdtsc() {
  return 0;  // not available
}
