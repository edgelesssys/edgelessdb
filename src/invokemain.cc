extern "C" void invokemain();
int mysqld_main(int argc, char** argv);

int main() {
  invokemain();
}

// edgeless_mysqld_main is like mysqld_main, but with C linkage
extern "C" int edgeless_mysqld_main(int argc, char** argv) {
  return mysqld_main(argc, argv);
}
