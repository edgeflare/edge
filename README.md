# edge: pocketbase, for PostgreSQL. its components scale as containers

[![CI](https://github.com/edgeflare/edge/actions/workflows/ci.yml/badge.svg)](https://github.com/edgeflare/edge/actions/workflows/ci.yml)
[![CodeQL](https://github.com/edgeflare/edge/actions/workflows/codeql.yml/badge.svg)](https://github.com/edgeflare/edge/actions/workflows/codeql.yml)
[![Release](https://github.com/edgeflare/edge/actions/workflows/release.yml/badge.svg)](https://github.com/edgeflare/edge/actions/workflows/release.yml)

Edge configures and manages: 
* [ZITADEL](https://github.com/zitadel/zitadel) - Centralized identity provider (OIDC)
* [SeaweedFS](https://github.com/seaweedfs/seaweedfs) - S3-compatible object storage
* [edgeflare/pgo](https://github.com/edgeflare/pgo) - PostgREST-compatible API and Debezium-compatible CDC
* [NATS](https://nats.io) - Message streaming platform
* [envoyproxy](https://github.com/envoyproxy/envoy) - Cloud-native high-performance edge/middle/service proxy
## How it works

Edge launches and configures these components to work together as a unified backend with PostgreSQL - similar to Supabase or Pocketbase. And with scaling capabilities.

## Deployment options

Edge can run as:
- A single binary (embeds official component binaries)
- [Docker compose](./docker-compose.yaml)
- Kubernetes resources (follow this README)
- Via a Kubernetes CRD named [Project](./example/project.yaml)

This project is in the ideation stage. Edge configures/manages the four underlying tools to create a cohesive system.

Interested in experimenting or contributing? See [CONTRIBUTING.md](./CONTRIBUTING.md).

```sh
git clone git@github.com:edgeflare/edge.git && cd edge
```

This uses iam.example.local and api.example.local domains. Ensure they point to the Gateway IP (envoyproxy) eg by adding an entry to `/etc/hosts` like

```sh
127.0.0.1 api.example.local iam.example.local
```

### [docker-compose.yaml](./docker-compose.yaml)

Adjust the docker-compose.yaml

- Free up port 80 from envoy for zitadel's initial configuration via management API which requires end-to-end HTTP/2 support.
envoyproxy config in docker doesn't support (our xds-server incomplete) HTTP/2 yet; on [k8s](https://raw.githubusercontent.com/edgeflare/pgo/refs/heads/main/k8s.yaml) everything works fine.

```yaml
  envoy:
    ports:
    - 9901:9901
    # - 80:10080 # or use eg 10080:10080
```

- Expose ZITADEL on port 80, by uncommenting

```yaml
    ports:
    - 80:8080
```

```sh
docker compose up -d
```

#### Use the centralized IdP for authorization in Postgres via `pgo rest` (PostgREST API)

Any OIDC compliant Identity Provider (eg ZITADEL, Keycloak, Auth0) can be used.

```sh
export ZITADEL_ISSUER=http://iam.example.local
export ZITADEL_API=iam.example.local:80
export ZITADEL_KEY_PATH=__zitadel-machinekey/zitadel-admin-sa.json
export ZITADEL_JWK_URL=http://iam.example.local/oauth/v2/keys
```

Configure components eg create OIDC clients in ZITADEL etc

```sh
go run ./internal/util/configure/...
```

Once done, revert the ports (use 80 for envoy), and `docker compose restart`

#### pgo rest

Visit http://iam.example.local, login and regenerate client-secret for oauth2-proxy client in edge project. Then adjust `internal/util/pgo/config.yaml`

> `pgo rest` container fails because of proxy issues. It can be run locally

```sh
go install github.com/edgeflare/pgo@latest # or download from release page
```
##### PostgREST-compatible REST API

```sh
pgo rest --config internal/util/pgo/config.yaml --rest.pg_conn_string "host=localhost port=5432 user=pgo password=pgopw dbname=main sslmode=prefer"
```

###### realtime/replication eg sync users from auth-db to app-db

Create table in sink-db. See pgo repo for more examples

```sh
PGUSER=postgres PGPASSWORD=postgrespw PGHOST=localhost PGDATABASE=main PGPORT=5432 psql
```

```sql
CREATE SCHEMA IF NOT EXISTS iam;

CREATE TABLE IF NOT EXISTS iam.users (
  id TEXT DEFAULT gen_random_uuid()::TEXT PRIMARY KEY
);
```

Start pipeline

```sh
pgo pipeline --config internal/util/pgo/config.yaml
```

### Kubernetes
If you already have a live k8s cluster, great just copy-paste-enter.
For development and lightweight prod, [k3s](https://github.com/k3s-io/k3s) seems a great option.
See [example/cluster](./example/cluster) for cluster setup.

```sh
kubectl apply -f example/k8s/00-secrets.yaml

# Database: PostgreSQL
helm upgrade --install example-postgres oci://registry-1.docker.io/bitnamicharts/postgresql -f example/k8s/01-postgres.values.yaml
kubectl apply -f example/k8s/01-postgres.tcproute.yaml
kubectl wait --for=condition=Ready pod -l app.kubernetes.io/instance=example-postgres --timeout=-1s

# AuthN / AuthZ: ZITADEL
helm upgrade --install example-zitadel oci://registry-1.docker.io/edgeflare/zitadel -f example/k8s/02-zitadel.values.yaml
kubectl apply -f example/k8s/02-zitadel.httproute.yaml
```

```sh
kubectl get secrets zitadel-admin-sa -o jsonpath='{.data.zitadel-admin-sa\.json}' | base64 -d > __zitadel-machinekey/zitadel-admin-sa.json

export ZITADEL_ADMIN_PW=$(kubectl get secrets example-zitadel-firstinstance -o jsonpath='{.data.ZITADEL_FIRSTINSTANCE_ORG_HUMAN_PASSWORD}' | base64 -d)
```

Configure zitadel like in docker-compose. Then apply something like `https://raw.githubusercontent.com/edgeflare/pgo/refs/heads/main/k8s.yaml`


## Cleanup

```sh
kubectl delete -f example/k8s/00-secrets.yaml -f example/k8s/01-postgres.tcproute.yaml -f example/k8s/02-zitadel.httproute.yaml -f example/k8s/03-postgrest.yaml

helm uninstall example-zitadel
helm uninstall example-postgres

kubectl delete cm zitadel-config-yaml
kubectl delete secret zitadel-admin-sa
kubectl delete jobs.batch example-zitadel-init example-zitadel-setup

kubectl delete $(kubectl get pvc -l app.kubernetes.io/instance=example-postgres -o name)
```
