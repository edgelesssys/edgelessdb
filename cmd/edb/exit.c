#include <stdlib.h>

int edgeless_exit_ensure_link;

void exit(int status) {
  void edgeless_exit();
  edgeless_exit(status);
  abort();  // unreachable; avoids "noreturn does return" warning
}
