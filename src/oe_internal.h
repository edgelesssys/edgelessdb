// Copyright (c) Open Enclave SDK contributors.
// Licensed under the MIT License.

#pragma once

#include <openenclave/bits/result.h>
#include <openenclave/log.h>

extern "C" oe_result_t oe_log(oe_log_level_t level, const char* fmt, ...);

typedef int64_t oe_host_fd_t;
typedef int64_t oe_off_t;

enum oe_fd_type_t {
  OE_FD_TYPE_NONE,
  OE_FD_TYPE_ANY,
  OE_FD_TYPE_FILE,
};

struct oe_fd_t;

/* Common operations on file-descriptor objects. */
typedef struct _oe_fd_ops {
  ssize_t (*read)(oe_fd_t* desc, void* buf, size_t count);

  ssize_t (*write)(oe_fd_t* desc, const void* buf, size_t count);

  ssize_t (*readv)(oe_fd_t* desc, const struct oe_iovec* iov, int iovcnt);

  ssize_t (*writev)(oe_fd_t* desc, const struct oe_iovec* iov, int iovcnt);

  int (*flock)(oe_fd_t* desc, int operation);

  int (*dup)(oe_fd_t* desc, oe_fd_t** new_fd);

  int (*ioctl)(oe_fd_t* desc, unsigned long request, uint64_t arg);

  int (*fcntl)(oe_fd_t* desc, int cmd, uint64_t arg);

  int (*close)(oe_fd_t* desc);

  oe_host_fd_t (*get_host_fd)(oe_fd_t* desc);
} oe_fd_ops_t;

/* File operations. */
typedef struct _oe_file_ops {
  /* Inherited operations. */
  oe_fd_ops_t fd;

  oe_off_t (*lseek)(oe_fd_t* file, oe_off_t offset, int whence);

  ssize_t (*pread)(oe_fd_t* desc, void* buf, size_t count, oe_off_t offset);

  ssize_t (
      *pwrite)(oe_fd_t* desc, const void* buf, size_t count, oe_off_t offset);

  int (*getdents64)(oe_fd_t* file, struct oe_dirent* dirp, uint32_t count);

  int (*fstat)(oe_fd_t* file, struct oe_stat_t* buf);

  int (*ftruncate)(oe_fd_t* file, oe_off_t length);

  int (*fsync)(oe_fd_t* file);
  int (*fdatasync)(oe_fd_t* file);
} oe_file_ops_t;

struct oe_fd_t {
  oe_fd_type_t type;
  union {
    oe_fd_ops_t fd;
    oe_file_ops_t file;
  } ops;
};

extern "C" int oe_fdtable_assign(oe_fd_t* desc);
