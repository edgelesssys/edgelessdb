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

#include "rocksdb.h"

#include <rocksdb/utilities/transaction_db.h>

#include <stdexcept>

// we need to use things from mariadb/storage/rocksdb/ha_rocksdb.cc
namespace myrocks {
extern rocksdb::TransactionDB* rdb;
rocksdb::ColumnFamilyHandle* edgeless_get_column_family(const std::string&);
}  // namespace myrocks

using namespace std;
using namespace edb;

static rocksdb::ColumnFamilyHandle* GetCf(string_view name) {
  const auto cf = myrocks::edgeless_get_column_family(string(name));
  if (!cf)
    throw runtime_error("rocksdb: column family not found");
  return cf;
}

std::optional<std::string> RocksDB::Get(std::string_view column_family, std::string_view key) const {
  if (!myrocks::rdb)
    return {};
  string value;
  const auto status = myrocks::rdb->Get({}, GetCf(column_family), key, &value);
  if (status.ok())
    return value;
  if (status.IsNotFound())
    return {};
  throw runtime_error("rocksdb: " + status.ToString());
}

void RocksDB::Put(std::string_view column_family, std::string_view key, std::string_view value) {
  if (!myrocks::rdb)
    throw logic_error("rocksdb: put called before store has been initialized");
  const auto status = myrocks::rdb->Put({}, GetCf(column_family), key, value);
  if (!status.ok())
    throw runtime_error("rocksdb: " + status.ToString());
  // MyRocks disables automatic flush in RocksDB, so we must flush manually.
  myrocks::rdb->FlushWAL(true);
}

void RocksDB::Delete(std::string_view column_family, std::string_view key) {
  if (!myrocks::rdb)
    throw logic_error("rocksdb: delete called before store has been initialized");
  const auto status = myrocks::rdb->Delete({}, GetCf(column_family), key);
  if (!status.ok())
    throw runtime_error("rocksdb: " + status.ToString());
  // see comment in Put
  myrocks::rdb->FlushWAL(true);
}

std::vector<std::string> RocksDB::GetKeys(std::string_view column_family, std::string_view prefix) const {
  if (!myrocks::rdb)
    return {};
  const unique_ptr<rocksdb::Iterator> it(myrocks::rdb->NewIterator({}, GetCf(column_family)));
  vector<string> result;
  for (it->Seek(prefix); it->Valid() && it->key().starts_with(prefix); it->Next())
    result.push_back(it->key().ToString());
  return result;
}
