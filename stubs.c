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
