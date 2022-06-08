set -e
echo 'package db'
echo
echo 'const mariadbBootstrap = `'
sed -e 's/`/` + "`" + `/g' -e 's/CREATE TEMPORARY TABLE tmp_/CREATE TABLE tmp_/' "$1/scripts/mysql_system_tables.sql"
echo '`'
