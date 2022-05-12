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

#pragma once

#include <mutex>
#include <optional>
#include <string_view>

#include "store.h"

namespace edb {

constexpr std::string_view kCfNameFrm = "edg_frm_cf";
constexpr std::string_view kCfNameDb = "edg_db_cf";

/*
SyscallHandler intercepts filesystem calls and redirects .frm and db.opt files to the store.

MariaDB would usually write different types of files to its data directory. We mount this directory
in memfs for security, excluding the encrypted RocksDB files. However, .frm and db.opt files need
to be persistent. To achieve this, we intercept access to them and store them in RocksDB.
*/
class SyscallHandler final {
 public:
  explicit SyscallHandler(StorePtr store);

  // Returns an int if the syscall was handled; otherwise, returns none.
  std::optional<int> Syscall(long number, long x1, long x2);

  // Returns the directory contents backed by the store.
  std::vector<std::string> Dir(std::string_view pathname) const;

  // These are called for open files backed by the store.
  size_t Read(std::string_view path, void* buf, size_t count, size_t offset) const;
  void Write(std::string_view path, std::string_view buf, size_t offset);
  size_t Size(std::string_view path) const;

 private:
  std::optional<int> Open(const char* pathname, int flags);
  std::optional<int> Access(const char* pathname) const;
  std::optional<int> Rename(const char* oldpath, const char* newpath);
  std::optional<int> Unlink(const char* pathname);
  bool Exists(std::string_view path) const;

  StorePtr store_;
  mutable std::mutex mutex_;
};

}  // namespace edb
