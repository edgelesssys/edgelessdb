#include <openenclave/ert_args.h>
#include <openenclave/host.h>
#include <openenclave/trace.h>
#include <semaphore.h>
#include <unistd.h>

#include <algorithm>
#include <array>
#include <cassert>
#include <cerrno>
#include <climits>
#include <csignal>
#include <cstdlib>
#include <exception>
#include <iostream>
#include <regex>
#include <stdexcept>
#include <string>
#include <string_view>
#include <system_error>
#include <thread>

#include "emain_u.h"

using namespace std;

extern "C" char** environ;

static ert_args_t _args;

ert_args_t ert_get_args_ocall() {
  assert(_args.argc > 0);
  return _args;
}

static void _init_args(int argc, char* argv[], char* envp[]) {
  assert(argc > 0);
  assert(argv);
  assert(envp);

  _args.argc = argc;
  _args.argv = argv;
  _args.envp = envp;

  // count envp elements
  while (envp[_args.envc])
    ++_args.envc;

#ifndef NDEBUG
  // initially, envp should be equal to environ
  {
    int i = 0;
    while (environ[i])
      ++i;
    assert(i == _args.envc);
  }
#endif

  _args.auxv = reinterpret_cast<const long*>(_args.envp + _args.envc + 1);

  // count auxv elements
  while (_args.auxv[2 * _args.auxc] || _args.auxv[2 * _args.auxc + 1])
    ++_args.auxc;

  // add cwd to env
  array<char, PATH_MAX> cwd{};
  if (getcwd(cwd.data(), cwd.size()) != cwd.data())
    throw system_error(errno, system_category(), "getcwd");
  if (setenv("EDG_CWD", cwd.data(), 0) != 0)
    throw system_error(errno, system_category(), "setenv");
  _args.envp = environ;
  ++_args.envc;
}

template <typename T>
class FinalAction final {
 public:
  explicit FinalAction(T f) noexcept : f_(std::move(f)), invoke_(true) {
  }

  FinalAction(FinalAction&& other) noexcept
      : f_(std::move(other.f_)), invoke_(other.invoke_) {
    other.invoke_ = false;
  }

  FinalAction(const FinalAction&) = delete;
  FinalAction& operator=(const FinalAction&) = delete;

  ~FinalAction() {
    if (invoke_)
      f_();
  }

 private:
  T f_;
  bool invoke_;
};

static int run(const char* path, bool simulate) {
  assert(path);

  // The semaphore will be unlocked if the program should exit, either because
  // the enclave main thread returned or SIGINT occurred. (Semaphore is the
  // only synchronization primitive that can be used inside a signal handler.)
  static sem_t sem_exit;
  if (sem_init(&sem_exit, 0, 0) != 0)
    throw system_error(errno, system_category(), "sem_init");

  if (simulate)
    cout << "[erthost] running in simulation mode\n";

  oe_enclave_t* enclave = nullptr;
  cout << "[erthost] loading enclave ...\n";

  if (oe_create_emain_enclave(
          path,
          OE_ENCLAVE_TYPE_AUTO,
          OE_ENCLAVE_FLAG_DEBUG_AUTO |
              (simulate ? OE_ENCLAVE_FLAG_SIMULATE : 0),
          nullptr,
          0,
          &enclave) != OE_OK ||
      !enclave)
    throw runtime_error(
        "oe_create_enclave failed. (Set OE_SIMULATION=1 "
        "for simulation mode.)");

  static int return_value = EXIT_FAILURE;

  {
    const FinalAction terminateEnclave([enclave] {
      signal(SIGINT, SIG_DFL);
      oe_terminate_enclave(enclave);
    });

    // SIGPIPE is received, among others, if a socket connection is lost. We
    // don't have signal handling inside the enclave yet and most
    // applications ignore the signal anyway and directly handle the errors
    // returned by the socket functions. Thus, we just ignore it.
    signal(SIGPIPE, SIG_IGN);

    cout << "[erthost] entering enclave ...\n";

    // create enclave main thread
    thread([enclave] {
      if (emain(enclave, &return_value) != OE_OK ||
          sem_post(&sem_exit) != 0)
        abort();
    }).detach();

    signal(SIGINT, [](int) {
      if (sem_post(&sem_exit) != 0)
        abort();
    });

    // wait until either the enclave main thread returned or SIGINT occurred
    while (sem_wait(&sem_exit) != 0)
      if (errno != EINTR)
        throw system_error(errno, system_category(), "sem_wait");
  }

  return return_value;
}

static void _trim_space(string& str) {
  int (&isspace)(int) = ::isspace;  // isspace is overloaded, select the wanted
  str.erase(find_if_not(str.rbegin(), str.rend(), isspace).base(), str.end());
  str.erase(str.begin(), find_if_not(str.begin(), str.end(), isspace));
}

static void _trim_prefix(string& str, string_view prefix) {
  if (str.compare(0, prefix.size(), prefix) == 0)
    str.erase(0, prefix.size());
}

extern "C" oe_log_level_t oe_get_current_logging_level();

static void _log(
    void* /*context*/,
    bool is_enclave,
    const tm* /*t*/,
    long /*usecs*/,
    oe_log_level_t level,
    uint64_t /*host_thread_id*/,
    const char* message) {
  assert(message && *message);
  if (level > oe_get_current_logging_level())
    return;
  const auto level_string = oe_log_level_strings[level];

  // split message of the form "log message ... [/path/to/source:func:line]"
  static const regex re_message(R"(([^]+) \[(.+):(\w+:\d+)]\n)");
  cmatch ma_message;
  if (!regex_match(message, ma_message, re_message)) {
    // not this form, so just print it
    cout << level_string << ": " << message << '\n';
    return;
  }
  string msg = ma_message[1];
  string path = ma_message[2];
  const auto& func_and_line = ma_message[3];

  // strip enclave name
  if (is_enclave)
    msg.erase(0, msg.find(':') + 1);

  // Check if the message contains the same OE error value as the last one.
  // This is a heuristic, but should be good enough.
  static const regex re_error("OE_[A-Z_]+");
  thread_local string last_error;
  if (smatch ma_error; regex_search(msg, ma_error, re_error)) {
    string error = ma_error.str();
    if (error == last_error) {
      // If it's a propagated error without additional info, don't print
      // it.
      if (msg == ':' + last_error)
        return;
    } else
      last_error = move(error);
  } else
    last_error.clear();

  // shorten the path
  const string_view oe_path = "/3rdparty/openenclave/";
  if (const size_t pos = path.find(oe_path); pos != string::npos)
    path.erase(0, pos + oe_path.size());
  else
    _trim_prefix(
        path,
        {__FILE__,
         sizeof __FILE__ - sizeof "src/tools/erthost/erthost.cpp"});

  _trim_space(msg);
  cout << level_string << ": " << msg << " [" << path << ':' << func_and_line
       << "]\n";
}

int main(int argc, char* argv[], char* envp[]) {
  if (argc < 2) {
    cout << "Usage: " << argv[0]
         << " enclave_image_path [enclave args...]\n"
            "Set OE_SIMULATION=1 for simulation mode.\n";
    return EXIT_FAILURE;
  }

  const char* const env_simulation = getenv("OE_SIMULATION");
  const bool simulation = env_simulation && *env_simulation == '1';

  // Configure detailed logging. Prefer OE_LOG_DETAILED value. If not set,
  // enable detailed logging for verbose level.
  bool log_detailed = false;
  const char* const env_log_detailed = getenv("OE_LOG_DETAILED");
  if (env_log_detailed && *env_log_detailed) {
    if (*env_log_detailed == '1')
      log_detailed = true;
  } else if (oe_get_current_logging_level() >= OE_LOG_LEVEL_VERBOSE)
    log_detailed = true;
  if (!log_detailed)
    oe_log_set_callback(nullptr, _log);

  try {
    _init_args(argc - 1, argv + 1, envp);
    return run(argv[1], simulation);
  } catch (const exception& e) {
    cout << e.what() << '\n';
  }

  return EXIT_FAILURE;
}
