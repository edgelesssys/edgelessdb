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

#include <fcntl.h>
#include <sys/syscall.h>

#include <cstdarg>
#include <functional>
#include <iostream>
#include <map>

#include "oe_internal.h"
#include "syscall_handler.h"

#define ASSERT(x) /*NOLINT*/                                       \
  if (!(x)) {                                                      \
    std::cout << __FILE__ << ':' << __LINE__ << ' ' << #x << '\n'; \
    abort();                                                       \
  }

using namespace std;
using namespace edb;

namespace edb {
// will be overridden for testing
extern function<decltype(oe_fdtable_assign)> fdtable_assign;
}  // namespace edb

namespace {
struct FakeStore : Store {
  std::optional<std::string> Get(std::string_view column_family, std::string_view key) const override {
    const auto it1 = data.find(column_family);
    if (it1 == data.cend())
      return {};
    const auto it2 = it1->second.find(key);
    if (it2 == it1->second.cend())
      return {};
    return it2->second;
  }

  void Put(std::string_view column_family, std::string_view key, std::string_view value) override {
    data[string(column_family)][string(key)] = value;
  }

  std::vector<std::string> GetKeys(std::string_view column_family, std::string_view prefix) const override {
    vector<string> result;
    for (const auto& [k, v] : data.at(string(column_family)))
      if (k.compare(0, prefix.size(), prefix) == 0)
        result.push_back(k);
    return result;
  }

  map<string, map<string, string, less<>>, less<>> data;
};
}  // namespace

static void TestAccess() {
  const auto store = make_shared<FakeStore>();
  store->Put(kCfNameDb, "./mydb/db.opt", {});
  store->Put(kCfNameFrm, "./mydb/mytab.frm", {});
  SyscallHandler handler(store);

  const auto my_access = [&handler](const char* path) {
    return handler.Syscall(SYS_access, reinterpret_cast<long>(path), 0);
  };

  // access existing files succeeds
  ASSERT(0 == my_access("./mydb/db.opt"));
  ASSERT(0 == my_access("./mydb/mytab.frm"));

  // access nonexistent files fails
  errno = 0;
  ASSERT(-1 == my_access("./otherdb/db.opt"));
  ASSERT(ENOENT == errno);
  errno = 0;
  ASSERT(-1 == my_access("./mydb/othertab.frm"));
  ASSERT(ENOENT == errno);

  // access folder of existing db succeeds
  ASSERT(0 == my_access("./mydb"));
  ASSERT(0 == my_access("./mydb/"));

  // access other folder is not handled
  ASSERT(!my_access("./otherdb"));
}

static void TestFile() {
  const auto path = "./foo/db.opt";
  const string_view in = "bar";

  const auto store = make_shared<FakeStore>();
  SyscallHandler handler(store);

  oe_fd_t* file = nullptr;
  fdtable_assign = [&file](oe_fd_t* desc) {
    file = desc;
    return 2;
  };

  // write the file
  ASSERT(2 == handler.Syscall(SYS_open, reinterpret_cast<long>(path), O_CREAT));
  ASSERT(3 == file->ops.fd.write(file, in.data(), in.size()));
  ASSERT(0 == file->ops.fd.close(file));

  // read the file
  ASSERT(2 == handler.Syscall(SYS_open, reinterpret_cast<long>(path), 0));
  string out(in.size(), '\0');
  ASSERT(3 == file->ops.fd.read(file, out.data(), out.size()));
  ASSERT(0 == file->ops.fd.close(file));
  ASSERT(in == out);
}

static void TestOpenError() {
  const auto store = make_shared<FakeStore>();
  SyscallHandler handler(store);

  const auto my_open = [&handler](const char* path, int flags = 0) {
    return handler.Syscall(SYS_open, reinterpret_cast<long>(path), flags);
  };

  // open nonexistent frm fails
  errno = 0;
  ASSERT(-1 == my_open("./foo/bar.frm"));
  ASSERT(ENOENT == errno);

  // open nonexistent opt fails
  errno = 0;
  ASSERT(-1 == my_open("./foo/db.opt"));
  ASSERT(ENOENT == errno);

  // open other file is not handled
  ASSERT(!my_open("./foo/bar.baz"));
}

static void TestDir() {
  const auto store = make_shared<FakeStore>();
  store->Put(kCfNameDb, "./mydb/db.opt", {});
  store->Put(kCfNameFrm, "./mydb/foo.frm", {});
  store->Put(kCfNameFrm, "./mydb/bar.frm", {});
  const SyscallHandler handler(store);

  ASSERT(vector<string>{"mydb"} == handler.Dir("."));
  ASSERT(vector<string>{"mydb"} == handler.Dir("/data/"));
  ASSERT((vector<string>{"bar.frm", "foo.frm"}) == handler.Dir("./mydb"));
  ASSERT((vector<string>{"bar.frm", "foo.frm"}) == handler.Dir("./mydb/"));
  ASSERT((vector<string>{"bar.frm", "foo.frm"}) == handler.Dir("/data/mydb"));
  ASSERT((vector<string>{"bar.frm", "foo.frm"}) == handler.Dir("/data/mydb/"));
  ASSERT(handler.Dir("./otherdb").empty());
}

int main() {
  TestAccess();
  TestFile();
  TestOpenError();
  TestDir();
  cout << "pass\n";
}

// We must define this func to satisfy the linker. It won't be called.
int oe_fdtable_assign(oe_fd_t* /*desc*/) {
  ASSERT(false);
}

// We must define this func to satisfy the linker.
oe_result_t oe_log(oe_log_level_t /*level*/, const char* fmt, ...) {
  va_list valist;
  va_start(valist, fmt);
  vprintf(fmt, valist);
  va_end(valist);
  return OE_OK;
}
