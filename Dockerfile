# syntax=docker/dockerfile:experimental

FROM alpine/git:latest AS pull
ADD http://worldtimeapi.org/api/ip /time.tmp
RUN --mount=type=secret,id=repoaccess,dst=/root/.netrc,required=true git clone --recurse-submodule https://github.com/edgelesssys/mariadb-ert.git /mariadb

FROM ghcr.io/edgelesssys/edgelessrt-dev AS build
COPY --from=pull /mariadb /mariadb
WORKDIR /mariadb
RUN ./build-mariadb.sh
COPY ./my.cnf /etc/
RUN mkdir -p /opt/mysql && cd /opt/mysql && server/mysql_install_db --srcdir=./server --ldata=/opt/mysql/data --defaults-file=/etc/my.cnf --user=$LOGNAME --auth-root-authentication-method=normal
# build emariadb
WORKDIR /mariadb/build
RUN cmake ..
RUN --mount=type=secret,id=signingkey,dst=/coordinator/build/private.pem,required=true make


FROM ghcr.io/edgelesssys/edgelessrt-deploy AS release
LABEL description="emariadbd"
COPY --from=build /mariadb/build/emariadb /
COPY --from=build /opt/mysql .
COPY ./my.cnf /etc/
# ENTRYPOINT ["erthost", "coordinator-enclave.signed"]
ENTRYPOINT ["/emariadbd"]