# edge: pocketbase, for PostgreSQL. its components scale as containers

[![CI](https://github.com/edgeflare/edge/actions/workflows/ci.yml/badge.svg)](https://github.com/edgeflare/edge/actions/workflows/ci.yml)
[![CodeQL](https://github.com/edgeflare/edge/actions/workflows/codeql.yml/badge.svg)](https://github.com/edgeflare/edge/actions/workflows/codeql.yml)
[![Release](https://github.com/edgeflare/edge/actions/workflows/release.yml/badge.svg)](https://github.com/edgeflare/edge/actions/workflows/release.yml)

edge configures and manages:

* [PostgreSQL](https://www.postgresql.org/): The world's most advanced open source database
* [ZITADEL](https://github.com/zitadel/zitadel): Centralized identity provider (OIDC)
* [MinIO](https://github.com/minio/minio) / [SeaweedFS](https://github.com/seaweedfs/seaweedfs): S3-compatible object storage
* [NATS](https://nats.io): Message streaming platform
* [envoy](https://github.com/envoyproxy/envoy): Cloud-native high-performance edge/middle/service proxy
* [edgeflare/pgo](https://github.com/edgeflare/pgo): PostgREST-compatible API and Debezium-compatible CDC

for a unified backend - similar to Firebase, Supabase, Pocketbase etc. And with scaling capabilities.

## Deployment options

- A single binary (embeds official component binaries): planned
- [Docker compose](./docker-compose.yaml) or Kubernetes resources: follow this README
- Via a Kubernetes CRD: [Project](./example/project.yaml)

edge is in very early stage. Interested in experimenting or contributing? See [CONTRIBUTING.md](./CONTRIBUTING.md).

```sh
git clone git@github.com:edgeflare/edge.git && cd edge
```

### [docker-compose.yaml](./docker-compose.yaml)

Adjust the docker-compose.yaml

- Free up port 80 from envoy for zitadel's initial configuration via its management API which requires end-to-end HTTP/2 support.
We still need to get envoy (in docker) to proxy HTTP/2 traffic. On k8s everything works fine.

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

Configure ZITADEL. Adjust the domain in env vars, and in `internal/stack/envoy/config.yaml`

```sh
export ZITADEL_HOSTNAME=iam.192-168-0-121.sslip.io
export ZITADEL_ISSUER=http://$ZITADEL_HOSTNAME
export ZITADEL_API=$ZITADEL_HOSTNAME:80
export ZITADEL_KEY_PATH=__zitadel-machinekey/zitadel-admin-sa.json
export ZITADEL_JWK_URL=http://$ZITADEL_HOSTNAME/oauth/v2/keys
```

```sh
go run ./internal/stack/configure/...
```

The above go code creates, among others, an OIDC client which pgo uses for authN/authZ. Any OIDC compliant Identity Provider (eg , Keycloak, Auth0) can be used; pgo just needs the client credentials.

Once ZITADEL is configured, revert the ports (use 80 for envoy), and `docker compose down && docker compose up -d`

Visit ZITADEL UI (eg at http://iam.192-168-0-121.sslip.io), login (see docker-compose.yaml) and regenerate client-secret for oauth2-proxy client in edge project. Then update `internal/stack/pgo/config.yaml` with the values. Again, `docker compose down && docker compose up -d`

#### `pgo rest`: PostgREST-compatible REST API

Create a table in app-db for REST and pipeline demo. See pgo repo for more examples

```sh
PGUSER=postgres PGPASSWORD=postgrespw PGHOST=localhost PGDATABASE=main PGPORT=5432 psql
```

```sql
CREATE SCHEMA IF NOT EXISTS iam;

CREATE TABLE IF NOT EXISTS iam.users (
  id TEXT DEFAULT gen_random_uuid()::TEXT PRIMARY KEY
);

-- wide-open for demo. use GRANT and RLS for granular ACL
GRANT USAGE ON SCHEMA iam to anon;
GRANT ALL ON iam.users to anon;
```

`docker restart edge_pgo-rest_1` to reload schema cache if it bugs.
Now we can GET, POST, PATCH, DELETE on the users table in iam schema like:

```sh
curl http://api.127-0-0-1.sslip.io/iam/users
```

##### `pgo pipeline`: Debezium-compatible CDC for realtime-event/replication etc

The demo pgo-pipeline container syncs users from auth-db (in projections.users14 table) to app-db (in iam.users)

### Kubernetes
If you already have a live k8s cluster, great just copy-paste-enter.
For development and lightweight prod, [k3s](https://github.com/k3s-io/k3s) seems a great option.
See [example/cluster](./example/cluster) for setup.

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
