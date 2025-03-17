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

- determine a root domain (hostname) eg `example.org`. if such a globally routable domain isn't available,
use something like https://sslip.io, which returns embedded IP address in domain name. that's what this demo setup does

> when containers dependent on zitadel (it being the centralized IdP) fail, try restarting it once zitadel is healthy

```sh
EDGE_DOMAIN_ROOT=192-168-0-10.sslip.io              # resolves to 192.168.0.121 (gateway/envoy IP). use LAN or accesible IP/hostname
ZITADEL_EXTERNALDOMAIN=iam.192-168-0-10.sslip.io
MINIO_BROWSER_REDIRECT_URL=http://minio.192-168-0-10.sslip.io
```

similarly adjust `internal/stack/envoy/config.yaml` and `internal/stack/pgo/config.yaml`

- ensure zitadel container can write admin service account key which edge uses to configure zitadel

```sh
mkdir -p __zitadel
chmod -R a+rw __zitadel
```

- ensure ./tls.key ./tls.crt exists. Use something like

```sh
openssl req -x509 -newkey rsa:4096 -keyout tls.key -out tls.crt -days 365 -nodes \
  -subj "/CN=iam.example.local" \
  -addext "subjectAltName=DNS:*.example.local,DNS:*.192-168-0-10.sslip.io"

# for envoy container to access keypair
chmod 666 tls.crt
chmod 666 tls.key
```

This is to configure envoy for end-to-end HTTP/2 required by zitadel management API. zitadel API bugs with self-signed certificates.
For publicly trusted certificates, enable TLS by updating env vars in ZITADEL.

```sh
docker compose up -d
```

Check zitadel health with `curl http://iam.192-168-0-10.sslip.io/debug/healthz` or `docker exec -it edge_edge_1 /edge healthz`

#### Use the centralized IdP for authorization in Postgres via `pgo rest` (PostgREST API) as well as minio-s3, NATS etc

edge so far creates the clients. a bit works needed to for configuring consumers of client secrets.
For now, isit ZITADEL UI (eg at http://iam.192-168-0-10.sslip.io), login (see docker-compose.yaml) and regenerate client-secrets for oauth2-proxy and minio clients in edge project. Then

- update `internal/stack/pgo/config.yaml` with the values
- update relevant env vars in minio container

And `docker compose down && docker compose up -d`

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
curl http://api.192-168-0-10.sslip.io/iam/users
```

##### `pgo pipeline`: Debezium-compatible CDC for realtime-event/replication etc

The demo pgo-pipeline container syncs users from auth-db (in projections.users14 table) to app-db (in iam.users)

#### minio-s3
ensure minio MINIO_IDENTITY_OPENID_CLIENT_ID and MINIO_IDENTITY_OPENID_CLIENT_SECRET are set withc appropriate values. console ui is at http://minio.192-168-0-10.sslip.io.

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
