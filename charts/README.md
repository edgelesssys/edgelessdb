# EdgelessDB helm charts

## Documentation

See the [docs](https://docs.edgeless.systems/edgelessdb/#/) for a comprehensive documentation of EdgelessDB.

## Add Repository (stable)

```bash
helm repo add edgeless https://helm.edgeless.systems/stable
helm repo update
```

## Install Packages (stable)

* If you are deploying on a cluster with nodes that support SGX1+FLC (e.g. AKS or minikube + Azure Standard_DC*s)

    * If you are deploying with MarbleRun

        ```bash
        helm install edgelessdb edgeless/edgelessdb --set edb.launchMarble=true --create-namespace  --namespace edgelessdb
        ```

    * Otherwise

        ```bash
        helm install edgelessdb edgeless/edgelessdb --create-namespace  --namespace edgelessdb
        ```

* Otherwise

    ```bash
    helm install edgelessdb edgeless/edgelessdb --create-namespace --namespace edgelessdb --set edgelessdb.resources=null --set edgelessdb.simulation=true --set tolerations=null
    ```

## Configuration

The following table lists the configurable parameters of the marblerun-coordinator chart and
their default values.

| Parameter                     | Type    | Description    | Default                              |
|:------------------------------|:---------------|:---------------|:-------------------------------------|
| `edb.debug`                   |bool   | Set to `true` enable debug logging to the terminal. The manifest must allow this because logs may leak data | `false` |
| `edb.heapSize`                |int    | Heap size of the database enclave in GB. Currently Edgeless Systems provides images for 1GB and 4GB heap sizes | `1` |
| `edb.launchMarble`            |bool   | Set to start EdgelessDB in MarbleRun mode | `false` |
| `edb.logDir`                  |string | Like EDG_EDB_DEBUG, but log to files. Set this, e.g., to /log and mount a storage interface to that path | `""` |
| `edb.marbleType`              |string | Label needed when EdgelessDB is deployed as a Marble with MarbleRun | `"EdgelessDB"` |
| `edb.replicas`                |int    | Number of replicas for EdgelessDB | `1` |
| `edb.resources`               |object | Resource requirements for EdgelessDB | `{limits:[{"sgx.intel.com/epc": "10Mi"},{"sgx.intel.com/provision":1},{"sgx.intel.com/enclave":1}]}` |
| `edb.restApiHost`             |string | The network address of the HTTP REST API | `"0.0.0.0"` |
| `edb.restApiPort`             |int    | Port of the HTTP REST API | `8080` |
| `edb.simulation`              |bool   | Needs be set to `true` when running on systems without SGX1+FLC capabilities | `false` |
| `edb.sqlApiHost`              |string | The network address of the MySQL interface | `"0.0.0.0"` |
| `edb.sqlApiPort`              |int    | Port of the MySQL interface | `3306` |
| `global.image`                |object | EdgelessDB image configuration | `{"pullPolicy":"IfNotPresent","version":" v0.3.1","repository":"ghcr.io/edgelesssys"}` |
| `global.podAnnotations`       |object | Additional annotations to add to all pods | `{}`|
| `global.podLabels`            |object | Additional labels to add to all pods | `{}` |
| `nodeSelector`                |object | NodeSelector section, See the [K8S documentation](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#nodeselector) for more information | `{"beta.kubernetes.io/os": "linux"}` |
| `storage`                     |object | EdgelessDB persistent storage spec | `{spec:{accessModes:[ReadWriteOnce],resources:{requests:{storage:"1Gi"}}}}` |
| `tolerations`                 |object | Tolerations section, See the [K8S documentation](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/) for more information | `{key:"sgx.intel.com/epc",operator:"Exists",effect:"NoSchedule"}` |

## Add new version (maintainers)

```bash
cd <edb-repo>
helm package charts
mv edgelessdb-x.x.x.tgz <helm-repo>/stable
cd <helm-repo>
helm repo index stable --url https://helm.edgeless.systems/stable
```
