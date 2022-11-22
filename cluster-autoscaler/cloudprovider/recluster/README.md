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

```sh
go run github.com/Khan/genqlient
```

### Build binaries

> **Warning**: Make sure you are inside [`cluster-autoscaler`](../../../cluster-autoscaler) directory

#### amd64

```sh
make build-arch-amd64
```

#### arm64

```sh
make build-arch-arm64
```

### Docker images

> **Warning**: Make sure you are inside [`cluster-autoscaler`](../../../cluster-autoscaler) directory

> **Warning**: Enable Docker BuildKit:
> - For all future commands in the current shell: `export DOCKER_BUILDKIT=1`
> - Just for one command: `DOCKER_BUILDKIT=1 docker build ...`

#### adm64

```sh
docker buildx build \
  --platform linux/amd64 \
  -f Dockerfile.amd64 \
  --tag recluster/cluster-autoscaler:latest \
  --output "type=docker,dest=cluster-autoscaler.amd64.tar.gz" \
  .
```

#### arm64

```sh
docker buildx build \
  --platform linux/arm64 \
  -f Dockerfile.arm64 \
  --tag recluster/cluster-autoscaler:latest \
  --output "type=docker,dest=cluster-autoscaler.arm64.tar.gz" \
  .
```
