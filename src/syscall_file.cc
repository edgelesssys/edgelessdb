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

#include "syscall_file.h"

#include <sys/stat.h>

#include <cassert>
#include <cstring>
#include <exception>
#include <functional>
#include <mutex>
#include <string>

#include "oe_internal.h"

using namespace std;
using namespace edb;

namespace edb {
// can be overridden for testing
function<decltype(oe_fdtable_assign)> fdtable_assign = oe_fdtable_assign;
}  // namespace edb

namespace {
struct File {
  oe_fd_t base{};
  string path;
  mutex mut;
  size_t offset = 0;
  SyscallHandler* handler = nullptr;
};
}  // namespace

static ssize_t file_read(oe_fd_t* desc, void* buf, size_t count) {
  try {
    auto& file = *reinterpret_cast<File*>(desc);
    const lock_guard lock(file.mut);
    const size_t res = file.handler->Read(file.path, buf, count, file.offset);
    file.offset += res;
    return res;
  } catch (const exception& ex) {
    oe_log(OE_LOG_LEVEL_ERROR, "file_read: %s\n", ex.what());
    errno = EIO;
    return -1;
  }
}

static ssize_t file_write(oe_fd_t* desc, const void* buf, size_t count) {
  try {
    auto& file = *reinterpret_cast<File*>(desc);
    const lock_guard lock(file.mut);
    file.handler->Write(file.path, string_view(static_cast<const char*>(buf), count), file.offset);
    file.offset += count;
    return count;
  } catch (const exception& ex) {
    oe_log(OE_LOG_LEVEL_ERROR, "file_write: %s\n", ex.what());
    errno = EIO;
    return -1;
  }
}

static int file_dup(oe_fd_t* /*desc*/, oe_fd_t** /*new_file_out*/) {
  errno = ENOSYS;
  return -1;
}

static int file_ioctl(oe_fd_t* /*desc*/, unsigned long /*request*/, uint64_t /*arg*/) {
  errno = ENOSYS;
  return -1;
}

static int file_fcntl(oe_fd_t* /*desc*/, int /*cmd*/, uint64_t /*arg*/) {
  errno = ENOSYS;
  return -1;
}

static int file_close(oe_fd_t* desc) {
  delete reinterpret_cast<File*>(desc);
  return 0;
}

static oe_host_fd_t file_get_host_fd(oe_fd_t* /*desc*/) {
  errno = ENOSYS;
  return -1;
}

static oe_off_t file_lseek(oe_fd_t* desc, oe_off_t offset, int whence) {
  try {
    auto& file = *reinterpret_cast<File*>(desc);
    const lock_guard lock(file.mut);

    switch (whence) {
      case SEEK_SET:
        break;
      case SEEK_CUR:
        offset += file.offset;
        break;
      case SEEK_END:
        offset += file.handler->Size(file.path);
        break;
      default:
        errno = EINVAL;
        return -1;
    }

    if (offset < 0) {
      errno = EINVAL;
      return -1;
    }

    file.offset = offset;
  } catch (const exception& ex) {
    oe_log(OE_LOG_LEVEL_ERROR, "file_lseek: %s\n", ex.what());
    errno = EIO;
    return -1;
  }

  return offset;
}

static ssize_t file_pread(
    oe_fd_t* /*desc*/,
    void* /*buf*/,
    size_t /*count*/,
    oe_off_t /*offset*/) {
  errno = ENOSYS;
  return -1;
}

static ssize_t file_pwrite(
    oe_fd_t* /*desc*/,
    const void* /*buf*/,
    size_t /*count*/,
    oe_off_t /*offset*/) {
  errno = ENOSYS;
  return -1;
}

static int file_getdents64(
    oe_fd_t* /*desc*/,
    struct oe_dirent* /*dirp*/,
    unsigned int /*count*/) {
  errno = ENOSYS;
  return -1;
}

static int file_fstat(oe_fd_t* desc, struct oe_stat_t* buf) {
  auto& st = *reinterpret_cast<struct stat*>(buf);

  // Zero the stat buf, but consider that oe_stat is smaller because it doesn't contain the unused fields at the end of stat.
  // Size taken from a static_assert in openenclave/include/openenclave/internal/syscall/sys/stat.h
  // As the struct is generated on build from an EDL file, we cannot include it, but must hardcode the size here.
  constexpr size_t sizeof_oe_stat = 120;
  static_assert(sizeof_oe_stat < sizeof st);
  memset(&st, 0, sizeof_oe_stat);

  try {
    auto& file = *reinterpret_cast<File*>(desc);
    const lock_guard lock(file.mut);
    st.st_size = file.handler->Size(file.path);
  } catch (const exception& ex) {
    oe_log(OE_LOG_LEVEL_ERROR, "file_fstat: %s\n", ex.what());
    errno = EIO;
    return -1;
  }

  return 0;
}

static int file_ftruncate(oe_fd_t* /*desc*/, oe_off_t /*length*/) {
  errno = ENOSYS;
  return -1;
}

static int file_fsync(oe_fd_t* /*desc*/) {
  return 0;
}

int edb::RedirectOpenFile(std::string_view path, SyscallHandler* handler) {
  assert(!path.empty());
  assert(handler);

  auto file = make_unique<File>();
  file->base.type = OE_FD_TYPE_FILE;
  file->path = path;
  file->handler = handler;

  auto& ops = file->base.ops;
  ops.fd.read = file_read;
  ops.fd.write = file_write;
  ops.fd.dup = file_dup;
  ops.fd.ioctl = file_ioctl;
  ops.fd.fcntl = file_fcntl;
  ops.fd.close = file_close;
  ops.fd.get_host_fd = file_get_host_fd;
  ops.file.lseek = file_lseek;
  ops.file.pread = file_pread;
  ops.file.pwrite = file_pwrite;
  ops.file.getdents64 = file_getdents64;
  ops.file.fstat = file_fstat;
  ops.file.ftruncate = file_ftruncate;
  ops.file.fsync = file_fsync;
  ops.file.fdatasync = file_fsync;

  const int fd = fdtable_assign(&file->base);
  if (fd < 0)
    return -1;

  (void)file.release();
  return fd;
}
