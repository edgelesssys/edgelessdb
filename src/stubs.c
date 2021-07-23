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

#include <openenclave/ert_stubs.h>

ERT_STUB(backtrace_symbols_fd, 0)
ERT_STUB(fedisableexcept, -1)
ERT_STUB(getcontext, -1)
ERT_STUB_SILENT(gnu_dev_major, 0)
ERT_STUB_SILENT(gnu_dev_minor, 0)
ERT_STUB(makecontext, 0)
ERT_STUB(mallinfo, 0)
ERT_STUB_SILENT(pthread_setname_np, 0)
ERT_STUB(pthread_yield, -1)
ERT_STUB(setcontext, -1)
ERT_STUB(__fdelt_chk, 0)

// musl implements POSIX which returns int, but we
// compile mariadb with glibc which returns char*
// see man strerror
char* strerror_r(int err) {
  char* strerror();
  // sufficient for mariadb to just return strerror() here
  return strerror(err);
}
// musl defines this in strerror_r.c. We must also do it to prevent multiple definition error.
OE_WEAK_ALIAS(strerror_r, __xpg_strerror_r);
