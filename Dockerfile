FROM ghcr.io/edgelesssys/edgelessdb/build-base:v0.3.1 AS build

# don't run `apt-get update` because required packages are cached in build-base for reproducibility
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
  bbe \
  bison \
  build-essential \
  ca-certificates \
  clang-10 \
  cmake \
  git \
  liblz4-dev \
  libncurses-dev \
  libssl-dev \
  ninja-build \
  zlib1g-dev

ARG erttag=v0.3.5
ARG edbtag=v0.3.1
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
  && make sign-edb \
  && cat edgelessdb-sgx.json

# deploy
FROM ubuntu:focal-20220801
ARG PSW_VERSION=2.17.100.3-focal1
ARG DCAP_VERSION=1.14.100.3-focal1
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates gnupg libcurl4 wget \
  && wget -qO- https://download.01.org/intel-sgx/sgx_repo/ubuntu/intel-sgx-deb.key | apt-key add \
  && echo 'deb [arch=amd64] https://download.01.org/intel-sgx/sgx_repo/ubuntu focal main' >> /etc/apt/sources.list \
  && wget -qO- https://packages.microsoft.com/keys/microsoft.asc | apt-key add \
  && echo 'deb [arch=amd64] https://packages.microsoft.com/ubuntu/20.04/prod focal main' >> /etc/apt/sources.list \
  && apt-get update && apt-get install -y --no-install-recommends \
  libsgx-ae-id-enclave=$DCAP_VERSION \
  libsgx-ae-pce=$PSW_VERSION \
  libsgx-ae-qe3=$DCAP_VERSION \
  libsgx-dcap-ql=$DCAP_VERSION \
  libsgx-enclave-common=$PSW_VERSION \
  libsgx-launch=$PSW_VERSION \
  libsgx-pce-logic=$DCAP_VERSION \
  libsgx-qe3-logic=$DCAP_VERSION \
  libsgx-urts=$PSW_VERSION \
  && apt-get install -d az-dcap-client libsgx-dcap-default-qpl=$DCAP_VERSION
COPY --from=build /edbbuild/edb /edbbuild/edb-enclave.signed /edbbuild/edgelessdb-sgx.json /edgelessdb/src/entry.sh /
COPY --from=build /opt/edgelessrt/bin/erthost /opt/edgelessrt/bin/
ENV PATH=${PATH}:/opt/edgelessrt/bin
ENTRYPOINT ["/entry.sh"]
EXPOSE 3306 8080
