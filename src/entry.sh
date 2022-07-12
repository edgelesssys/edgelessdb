#!/bin/sh
set -e

if [ -n "${PCCS_ADDR}" ]; then
	PCCS_URL=https://${PCCS_ADDR}/sgx/certification/v3/
fi

if [ -n "${PCCS_URL}" ]; then
	apt-get install -qq libsgx-dcap-default-qpl
	ln -fs /usr/lib/x86_64-linux-gnu/libdcap_quoteprov.so.1 /usr/lib/x86_64-linux-gnu/libdcap_quoteprov.so
	echo "PCCS_URL: ${PCCS_URL}"
	echo "PCCS_URL=${PCCS_URL}\nUSE_SECURE_CERT=FALSE" > /etc/sgx_default_qcnl.conf
else
	apt-get install -qq az-dcap-client
fi

./edb "$@"
