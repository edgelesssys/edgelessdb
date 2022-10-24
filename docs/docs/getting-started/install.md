# Install EdgelessDB

EdgelessDB is provided as a Docker image. There are two flavors:
* `ghcr.io/edgelesssys/edgelessdb-sgx-1gb` with 1 GB of enclave heap memory
* `ghcr.io/edgelesssys/edgelessdb-sgx-4gb` with 4 GB of enclave heap memory

Use `edgelessdb-sgx-1gb` primarily to test EdgelessDB on machines with limited resources. Prefer `edgelessdb-sgx-4gb` for production deployments.

:::info

A future version of EdgelessDB will have a dynamic heap size.

:::

## Prepare the SGX system
Skip this section if you want to run EdgelessDB in simulation mode. You may also skip this section if you are running on an SGX-enabled VM in Azure (DC2 series).

### Hardware
The hardware must support SGX-FLC and it must be enabled in the BIOS. Use the following commands to check:
```bash
sudo apt install cpuid
cpuid | grep SGX
```

This should give you output like the following:
```shell-session
      SGX: Software Guard Extensions supported = true
      SGX_LC: SGX launch config supported      = true
   SGX capability (0x12/0):
      SGX1 supported                         = true
```

* `SGX: Software Guard Extensions supported` is true if the hardware supports it.
* `SGX_LC: SGX launch config supported` is true if the hardware also supports FLC.
* `SGX1 supported` is true if it's enabled in the BIOS.

### Driver
The SGX driver exposes a device:
```bash
ls /dev/*sgx*
```

If the output is empty, install the driver:
```bash
wget https://download.01.org/intel-sgx/latest/linux-latest/distro/ubuntu`lsb_release -rs`-server/sgx_linux_x64_driver_1.41.bin
chmod +x sgx_linux_x64_driver_1.41.bin
sudo ./sgx_linux_x64_driver_1.41.bin
```

### Packages

On some systems you may need to install the `libsgx-enclave-common` package.

On Ubuntu 18.04 or 20.04 you can do this by running:

```bash
wget -qO- https://download.01.org/intel-sgx/sgx_repo/ubuntu/intel-sgx-deb.key | sudo apt-key add
sudo add-apt-repository "deb [arch=amd64] https://download.01.org/intel-sgx/sgx_repo/ubuntu `lsb_release -cs` main"
sudo apt install --no-install-recommends libsgx-enclave-common
```

## Run the Docker image
Run EdgelessDB on an SGX-capable system:
```bash
docker run -t \
  --name my-edb \
  -p3306:3306 \
  -p8080:8080 \
  --device /dev/sgx_enclave --device /dev/sgx_provision \
  ghcr.io/edgelesssys/edgelessdb-sgx-1gb
```

Or try it in simulation mode on any system:
```bash
docker run -t \
  --name my-edb \
  -p3306:3306 \
  -p8080:8080 \
  -e OE_SIMULATION=1 \
  ghcr.io/edgelesssys/edgelessdb-sgx-1gb
```

This exposes two services:
* The MySQL interface served on port 3306
* The HTTP REST API on port 8080

## Storage
If EdgelessDB is run with one of the commands above, all data is stored inside the docker container in the `/data` directory. For a production deployment, consider using one of the [data management approaches of Docker](https://docs.docker.com/storage). E.g., to mount a directory on the host system, add `-v /my/own/datadir:/data` to the command line.

## Remote attestation
If you're on Azure, remote attestation works out of the box.

Otherwise, you must use a *Provisioning Certificate Caching Service (PCCS)*, which caches attestation data from Intel.

### Set up the PCCS
1. Register with [Intel](https://api.portal.trustedservices.intel.com/provisioning-certification) to get a PCCS API key
1. Run the PCCS:
   ```bash
   docker run -e APIKEY=<your-API-key> -p 8081:8081 --name pccs -d ghcr.io/edgelesssys/pccs
   ```
1. Verify that the PCCS is running:
   ```bash
   curl -kv https://localhost:8081/sgx/certification/v3/rootcacrl
   ```
   You should see a 200 status code.

### Configure EdgelessDB to use the PCCS
Add `-e PCCS_ADDR=<your-pccs-address>` to the Docker command line. E.g., if the PCCS runs on the same host, use `-e PCCS_ADDR=172.17.0.1:8081` (the gateway of Docker's default network bridge + the default PCCS port).
