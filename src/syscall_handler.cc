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

#include "syscall_handler.h"

#include <fcntl.h>
#include <sys/syscall.h>

#include <cassert>
#include <cerrno>
#include <cstring>
#include <regex>
#include <stdexcept>
#include <string>

#include "syscall_file.h"

using namespace std;
using namespace edb;

static const regex re_folder(R"(\./[^./]+)");
static const regex re_path_to_known_file(R"(\./[^./]+/(db\.opt|[^./]+\.frm))");

static bool StrEndsWith(string_view str, string_view suffix) {
  return str.size() >= suffix.size() && str.compare(str.size() - suffix.size(), suffix.size(), suffix) == 0;
}

static bool IsKnownExtension(string_view path) {
  return StrEndsWith(path, ".frm") || StrEndsWith(path, ".opt");
}

static string_view GetCf(string_view path) {
  if (StrEndsWith(path, ".frm"))
    return kCfNameFrm;
  if (StrEndsWith(path, ".opt"))
    return kCfNameDb;
  throw runtime_error("unexpected path");
}

SyscallHandler::SyscallHandler(StorePtr store)
    : store_(move(store)) {
}

std::optional<int> SyscallHandler::Syscall(long number, long x1, long x2) {
  switch (number) {
    case SYS_open:
      return Open(reinterpret_cast<char*>(x1), static_cast<int>(x2));
    case SYS_access:
      return Access(reinterpret_cast<char*>(x1));
    default:
      return {};
  }
}

size_t SyscallHandler::Read(std::string_view path, void* buf, size_t count, size_t offset) const {
  const string_view cf = GetCf(path);
  optional<string> value;

  {
    const lock_guard lock(mutex_);
    value = store_->Get(cf, path);
  }

  if (!value)
    throw logic_error("not found");

  if (value->size() <= offset)
    return 0;

  count = min(count, value->size() - offset);
  memcpy(buf, value->data() + offset, count);
  return count;
}

void SyscallHandler::Write(std::string_view path, std::string_view buf, size_t offset) {
  const string_view cf = GetCf(path);

  const lock_guard lock(mutex_);

  string value = store_->Get(cf, path).value_or(string());

  const size_t required_size = offset + buf.size();
  if (required_size < offset)
    throw overflow_error("write offset overflow");
  if (value.size() < required_size)
    value.resize(required_size);

  memcpy(value.data() + offset, buf.data(), buf.size());

  store_->Put(cf, path, value);
}

std::optional<int> SyscallHandler::Open(const char* pathname, int flags) {
  assert(pathname && *pathname);
  const string_view path = pathname;

  if (!IsKnownExtension(path))
    return {};
  if (!regex_match(path.cbegin(), path.cend(), re_path_to_known_file))
    throw invalid_argument("unexpected pathname");

  if (!(flags & O_CREAT) && !Exists(path)) {
    errno = ENOENT;
    return -1;
  }

  return RedirectOpenFile(path, this);
}

std::optional<int> SyscallHandler::Access(const char* pathname) const {
  assert(pathname && *pathname);

  string path = pathname;
  const bool known_ext = IsKnownExtension(path);

  if (known_ext) {
    if (!regex_match(path.cbegin(), path.cend(), re_path_to_known_file))
      throw invalid_argument("unexpected pathname");
  } else if (regex_match(path.cbegin(), path.cend(), re_folder)) {
    // It might be a db folder. Check if db.opt exists.
    path += "/db.opt";
  } else
    return {};

  if (Exists(path))
    return 0;
  if (!known_ext)
    return {};

  errno = ENOENT;
  return -1;
}

bool SyscallHandler::Exists(std::string_view path) const {
  const string_view cf = GetCf(path);
  const lock_guard lock(mutex_);
  return store_->Get(cf, path).has_value();
}
