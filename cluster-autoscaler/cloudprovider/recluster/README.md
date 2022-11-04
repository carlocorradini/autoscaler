# Cluster Autoscaler for reCluster

The cluster autoscaler for reCluster scales nodes in a reCluster cluster.

## Configuration

It is mandatory to define the cloud configuration file `cloud-config`.  You can see an example of the cloud config file at [examples/cluster-autoscaler-secret.yaml](./examples/cluster-autoscaler-secret.yaml).

The (JSON) configuration file of the reCluster cloud provider supports the following values:

- `token`: API auth token
- `URL`: reCluster API server URL


Configuring the autoscaler such as if it should be monitoring node pools or what the minimum and maximum values. Should be configured through the `reCluster API`.
The autoscaler will pick up any changes and adjust accordingly.

## Development

### Generate `generated.go` from `GraphQL` schema

> **Warning**: Make sure you are inside [`gorecluster`](./gorecluster) directory

```console
go run github.com/Khan/genqlient
```

### Build

> **Warning**: Make sure you are inside [`cluster-autoscaler`](../../../cluster-autoscaler) directory

```console
make build
```
tag the generated docker image and push it to a registry.

### Docker image

> **Warning**: Make sure you are inside [`cluster-autoscaler`](../../../cluster-autoscaler) directory

```console
make container
```
