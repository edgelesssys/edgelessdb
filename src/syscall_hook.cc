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

#include <cassert>
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
