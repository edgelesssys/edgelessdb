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

#include <memory>
#include <optional>
#include <string>
#include <string_view>
#include <vector>

namespace edb {

class Store {
 public:
  virtual ~Store() = default;
  virtual std::optional<std::string> Get(std::string_view column_family, std::string_view key) const = 0;
  virtual void Put(std::string_view column_family, std::string_view key, std::string_view value) = 0;
  virtual void Delete(std::string_view column_family, std::string_view key) = 0;
  virtual std::vector<std::string> GetKeys(std::string_view column_family, std::string_view prefix) const = 0;
};

typedef std::shared_ptr<Store> StorePtr;

}  // namespace edb
