set -e
echo 'package db'
echo
echo 'const mariadbBootstrap = `'
sed 's/`/` + "`" + `/g' "$1/scripts/mysql_system_tables.sql"
echo '`'
