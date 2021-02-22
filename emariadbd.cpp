// #include <mysql.h>
// #include <stdlib.h>
// #include <stdio.h>
// #include <unistd.h>


// static char *server_args[] = {
//   "emariadb",       /* this string is not used */
//   "--datadir=.",
//   "--key_buffer_size=32M"
// };
// static char *server_groups[] = {
//   "mysqld",
//   "mariadb",
//   "embedded",
//   "server",
//   (char *)NULL
// };

/*
  main() for mysqld.
  Calls mysqld_main() entry point exported by sql library.
*/

extern int mysqld_main(int argc, char **argv);

int main(int argc, char **argv)
{
  return mysqld_main(argc, argv);
}

// int main(int argc, char *argv[]) {
//   if (mysql_library_init(sizeof(server_args) / sizeof(char *),
//                         server_args, server_groups)) {
//     fprintf(stderr, "could not initialize MariaDB server library\n");
//     exit(1);
//   }
//   printf("Successfully started the MariaDB server\n");
//   sleep(100);

//   /* Use any MariaDB API functions here */

//   mysql_library_end();

//   return EXIT_SUCCESS;
// }