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

#include "store.h"

namespace edb {

class RocksDB final : public Store {
 public:
  std::optional<std::string> Get(std::string_view column_family, std::string_view key) const override;
  void Put(std::string_view column_family, std::string_view key, std::string_view value) override;
  void Delete(std::string_view column_family, std::string_view key) override;
  std::vector<std::string> GetKeys(std::string_view column_family, std::string_view prefix) const override;
};

}  // namespace edb
