FROM ubuntu:focal-20210827 AS build

RUN apt update && DEBIAN_FRONTEND=noninteractive apt install -y \
  bbe \
  bison=2:3.5.1+dfsg-1 \
  build-essential=12.8ubuntu1.1 \
  clang-10=1:10.0.0-4ubuntu1 \
  cmake=3.16.3-1ubuntu1 \
  doxygen \
  git \
  liblz4-dev=1.9.2-2ubuntu0.20.04.1 \
  libssl-dev=1.1.1f-1ubuntu2.8 \
  ninja-build=1.10.0-1build1 \
  zlib1g-dev=1:1.2.11.dfsg-2ubuntu1.2

ARG erttag=v0.2.7 edbtag=v0.1.1
RUN git clone -b $erttag --depth=1 https://github.com/edgelesssys/edgelessrt \
  && git clone -b $edbtag --depth=1 https://github.com/edgelesssys/edgelessdb \
  && mkdir ertbuild edbbuild

# install ert
RUN cd edgelessrt && export SOURCE_DATE_EPOCH=$(git log -1 --pretty=%ct) && cd /ertbuild \
  && cmake -GNinja -DCMAKE_BUILD_TYPE=Release -DBUILD_TESTS=OFF /edgelessrt \
  && ninja install

# build edb
RUN cd edgelessdb && export SOURCE_DATE_EPOCH=$(git log -1 --pretty=%ct) && cd /edbbuild \
  && . /opt/edgelessrt/share/openenclave/openenclaverc \
  && cmake -DCMAKE_BUILD_TYPE=Release -DBUILD_TESTS=OFF /edgelessdb \
  && make -j`nproc` edb-enclave

# sign edb
ARG heapsize=1024 production=OFF
RUN --mount=type=secret,id=signingkey,dst=/edbbuild/private.pem,required=true \
  cd edbbuild \
  && . /opt/edgelessrt/share/openenclave/openenclaverc \
  && cmake -DHEAPSIZE=$heapsize -DPRODUCTION=$production /edgelessdb \
  && make sign-edb

# deploy
FROM ubuntu:focal-20210827
ARG PSW_VERSION=2.13.103.1-focal1 DCAP_VERSION=1.10.103.1-focal1
RUN apt update && apt install -y gnupg wget \
  && wget -qO- https://download.01.org/intel-sgx/sgx_repo/ubuntu/intel-sgx-deb.key | apt-key add \
  && echo 'deb [arch=amd64] https://download.01.org/intel-sgx/sgx_repo/ubuntu focal main' >> /etc/apt/sources.list \
  && wget -qO- https://packages.microsoft.com/keys/microsoft.asc | apt-key add \
  && echo 'deb [arch=amd64] https://packages.microsoft.com/ubuntu/20.04/prod focal main' >> /etc/apt/sources.list \
  && apt update && apt install -y --no-install-recommends \
  az-dcap-client \
  libsgx-ae-pce=$PSW_VERSION \
  libsgx-ae-qe3=$DCAP_VERSION \
  libsgx-ae-qve=$DCAP_VERSION \
  libsgx-dcap-ql=$DCAP_VERSION \
  libsgx-dcap-ql-dev=$DCAP_VERSION \
  libsgx-enclave-common=$PSW_VERSION \
  libsgx-headers=$PSW_VERSION \
  libsgx-launch=$PSW_VERSION \
  libsgx-pce-logic=$DCAP_VERSION \
  libsgx-qe3-logic=$DCAP_VERSION \
  libsgx-urts=$PSW_VERSION
COPY --from=build /edbbuild/edb /edbbuild/edb-enclave.signed /
COPY --from=build /opt/edgelessrt/bin/erthost /opt/edgelessrt/bin/
ENV PATH=${PATH}:/opt/edgelessrt/bin AZDCAP_DEBUG_LOG_LEVEL=error
ENTRYPOINT ["./edb"]
EXPOSE 3306 8080
