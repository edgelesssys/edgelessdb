# syntax=docker/dockerfile:experimental

FROM alpine/git:latest AS pull
RUN --mount=type=secret,id=repoaccess,dst=/root/.netrc,required=true git clone --recurse-submodule https://github.com/edgelesssys/mariadb-ert.git /mariadb

FROM ghcr.io/edgelesssys/edgelessrt-dev AS build
RUN apt update && apt install -y libncurses5-dev bison libaio-dev
COPY --from=pull /mariadb /mariadb
WORKDIR /mariadb
RUN ./build-mariadb.sh
COPY ./my.cnf /etc/
RUN mkdir -p /opt/mysql && server/build/scripts/mysql_install_db --srcdir=./server --ldata=/opt/mysql/data --defaults-file=/etc/my.cnf --user=$LOGNAME --auth-root-authentication-method=normal
# build emariadb
WORKDIR /mariadb/build
RUN cmake ..
RUN --mount=type=secret,id=signingkey,dst=/coordinator/build/private.pem,required=true make


FROM ghcr.io/edgelesssys/edgelessrt-deploy AS release
LABEL description="emariadbd"
COPY --from=build /mariadb/build/emariadbd /
COPY --from=build /opt/mysql /opt/mysql
COPY ./my.cnf /etc/
# ENTRYPOINT ["erthost", "coordinator-enclave.signed"]
ENTRYPOINT ["/emariadbd", "--user=root"]
