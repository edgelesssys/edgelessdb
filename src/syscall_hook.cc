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

typedef unsigned int uint;
typedef unsigned long myf;

#include <my_dir.h>

#include <cassert>
#include <cstring>
#include <exception>
#include <memory>

#include "oe_internal.h"
#include "rocksdb.h"
#include "syscall_handler.h"

using namespace std;
using namespace edb;

static SyscallHandler handler(make_shared<RocksDB>());

extern "C" oe_result_t edgeless_syscall_hook(long number, long x1, long x2, long /*x3*/, long /*x4*/, long /*x5*/, long /*x6*/, long* ret) {
  assert(ret);

  try {
    const auto res = handler.Syscall(number, x1, x2);
    if (!res)
      return OE_UNEXPECTED;
    *ret = *res;
  } catch (const exception& ex) {
    oe_log(OE_LOG_LEVEL_ERROR, "syscall_hook %ld: %s\n", number, ex.what());
    *ret = -1;
    errno = EIO;
  }

  return OE_OK;
}

// To avoid implementing redirections for syscalls on directories, we replaced a few relevant my_dir calls with edgeless_my_dir.
MY_DIR* edgeless_my_dir(const char* path, myf /*MyFlags*/) {
  // dummy stat for all paths is sufficient to satisfy MariaDB
  static MY_STAT mystat = [] {
    MY_STAT mystat{};
    mystat.st_mode = MY_S_IFDIR;
    return mystat;
  }();

  try {
    const auto subpaths = handler.Dir(path);
    auto entries = make_unique<fileinfo[]>(subpaths.size());  // NOLINT
    auto dir = make_unique<MY_DIR>();
    dir->number_of_files = subpaths.size();

    // fill entries
    for (size_t i = 0; i < subpaths.size(); ++i) {
      const auto& subpath = subpaths[i];
      auto& entry = entries[i];
      entry.mystat = &mystat;
      entry.name = new char[subpath.size() + 1]();
      memcpy(entry.name, subpath.data(), subpath.size());
    }

    dir->dir_entry = entries.release();
    return dir.release();
  } catch (const exception& ex) {
    oe_log(OE_LOG_LEVEL_ERROR, "my_dir: %s\n", ex.what());
    return nullptr;
  }
}

void edgeless_my_dirend(MY_DIR* buffer) {
  assert(buffer);
  assert(buffer->dir_entry);
  for (size_t i = 0; i < buffer->number_of_files; ++i)
    delete[] buffer->dir_entry[i].name;
  delete[] buffer->dir_entry;
  delete buffer;
}
