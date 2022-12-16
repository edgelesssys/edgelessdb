#!/bin/sh
set -e

mkdir -p /dev/sgx
ln -s /dev/sgx_enclave /dev/sgx/enclave

if [ -n "${PCCS_ADDR}" ]; then
	PCCS_URL=https://${PCCS_ADDR}/sgx/certification/v3/
fi

if [ -z "${PCCS_URL}" ] && [ "$(cat /sys/devices/virtual/dmi/id/chassis_asset_tag)" != 7783-7084-3265-9085-8269-3286-77 ]; then
	PCCS_URL=https://172.17.0.1:8081/sgx/certification/v3/
fi

if [ -n "${PCCS_URL}" ]; then
	apt-get install -qq libsgx-dcap-default-qpl
	echo "PCCS_URL: ${PCCS_URL}"
	echo "PCCS_URL=${PCCS_URL}\nUSE_SECURE_CERT=FALSE" > /etc/sgx_default_qcnl.conf
else
	apt-get install -qq az-dcap-client
fi

./edb "$@"
