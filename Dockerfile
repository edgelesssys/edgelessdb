FROM ubuntu:focal-20220531 AS build

RUN apt update && DEBIAN_FRONTEND=noninteractive apt install -y \
  bbe \
  bison \
  build-essential \
  clang-10 \
  cmake \
  doxygen \
  git \
  liblz4-dev \
  libssl-dev \
  ninja-build \
  zlib1g-dev

ARG erttag=v0.3.3
ARG edbtag=v0.3.0
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
FROM ubuntu:focal-20220531
ARG PSW_VERSION=2.17.100.3-focal1
ARG DCAP_VERSION=1.14.100.3-focal1
RUN apt update && apt install -y gnupg libcurl4 wget \
  && wget -qO- https://download.01.org/intel-sgx/sgx_repo/ubuntu/intel-sgx-deb.key | apt-key add \
  && echo 'deb [arch=amd64] https://download.01.org/intel-sgx/sgx_repo/ubuntu focal main' >> /etc/apt/sources.list \
  && wget -qO- https://packages.microsoft.com/keys/microsoft.asc | apt-key add \
  && echo 'deb [arch=amd64] https://packages.microsoft.com/ubuntu/20.04/prod focal main' >> /etc/apt/sources.list \
  && apt update && apt install -y --no-install-recommends \
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
  libsgx-urts=$PSW_VERSION \
  && apt install -d az-dcap-client libsgx-dcap-default-qpl=$DCAP_VERSION
COPY --from=build /edbbuild/edb /edbbuild/edb-enclave.signed /edgelessdb/src/entry.sh /
COPY --from=build /opt/edgelessrt/bin/erthost /opt/edgelessrt/bin/
ENV PATH=${PATH}:/opt/edgelessrt/bin AZDCAP_DEBUG_LOG_LEVEL=error
ENTRYPOINT ["/entry.sh"]
EXPOSE 3306 8080
