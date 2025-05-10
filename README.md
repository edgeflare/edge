# edge: PostgreSQL backend in a binary, whose components scale as containers

edge configures and manages:

| Component         | Technology / Tool       | Description |
|-------------------|-----------------------|-------------|
| Database          | [PostgreSQL](https://www.postgresql.org) + [pgvector](https://github.com/pgvector/pgvector)  | The world's most advanced open source database. Vector search using pgvector |
| (IAM) AuthN/AuthZ | Any OIDC compliant IdP eg [Keycloak](https://www.keycloak.org), Auth0, [ZITADEL](https://github.com/zitadel/zitadel), [dexidp/dex](https://github.com/dexidp/dex) (default) + [Postgres RLS](https://www.postgresql.org/docs/current/ddl-rowsecurity.html) | Comprehensive authN and authZ through OIDC claims, PostgreSQL Row-Level Security and envoy filters eg ext-authz |
| Object Storage    | Any S3 compliant storage eg AWS S3, Cloudflare R2, [MinIO](https://github.com/minio/minio), Ceph RGW, [SeaweedFS](https://github.com/seaweedfs/seaweedfs) (default)                 | Offers high-performance, Kubernetes-native object storage. |
| REST API / Events | [edgeflare/pgo](https://github.com/edgeflare/pgo) | PostgREST-compatible REST API, Debezium-compatible CDC |
| API Gateway       | [Istio](https://istio.io)/[Envoy](https://www.envoyproxy.io), [cert-manager](https://cert-manager.io) and optionally [Cloudflare](https://cloudflare.com)         | Manages, secures, and monitors traffic between microservices as well as from and to the Internet |

to build a unified backend - similar to Firebase, Supabase etc. And with scaling capabilities. The stack runs on Linux, Docker and [Kubernetes](https://kubernetes.io) allowing it to start on a RaspberryPi-like device and scale to a multi-region Kubernetes cluster.

edge allows (is purposefully designed) to mix-match existing, external (incl proprietary) components from anywhere - eg GCP Cloud SQL, Auth0 IdP, AWS S3, etc. it simply ensures all these are configred to function as a single unit.

> **We use [PostgREST](https://docs.postgrest.org) where reliability is important; writing [edgeflare/pgo](https://github.com/edgeflare/pgo) in Go to be able to 1. embed in a go binary and 2. run in serverless env.**

## Deployment options

- Native: components binaries eg ZITADEL shuold be available on system. If not, edge will try downloading/installing from official releases
- [Docker compose](./docker-compose.yaml) or Kubernetes resources: follow this README
- Via a Kubernetes CRD: [Project](./example/project.yaml)

edge is in very early stage. Interested in experimenting or contributing? See [CONTRIBUTING.md](./CONTRIBUTING.md).

```sh
git clone git@github.com:edgeflare/edge.git && cd edge
```

### [docker-compose.yaml](./docker-compose.yaml)

1. determine a root domain (hostname) eg `example.org`. if such an FQDN isn't available, maybe edit `/etc/hosts` or utilize something like https://sslip.io resolver, which returns embedded IP address in domain name. that's what this demo setup does

> when containers dependent on zitadel (it being the centralized IdP) fail, try restarting them once zitadel is healthy

```sh
export EDGE_DOMAIN_ROOT=192-168-0-121.sslip.io              # resolves to 192.168.0.121 (gateway/envoy IP). use LAN or accesible IP/hostname
```

2. generate `envoy/config.yaml` and `pgo/config.yaml`

```sh
sed  "s/EDGE_DOMAIN_ROOT/${EDGE_DOMAIN_ROOT}/g" internal/stack/envoy/routes.template.yaml > envoy-routes.yaml
sed  "s/EDGE_DOMAIN_ROOT/${EDGE_DOMAIN_ROOT}/g" internal/stack/pgo/config.template.yaml > pgo-config.yaml
```

3. ensure zitadel container can write admin service account key which edge uses to configure zitadel

```sh
mkdir -p __zitadel
chmod -R a+rw __zitadel
```

4. ensure ./tls.key ./tls.crt exist. Use something like

```sh
openssl req -x509 -newkey rsa:4096 -keyout tls.key -out tls.crt -days 365 -nodes \
  -subj "/CN=iam.example.local" \
  -addext "subjectAltName=DNS:*.example.local,DNS:*.${EDGE_DOMAIN_ROOT}"

# for envoy container to access keypair
chmod 666 tls.crt
chmod 666 tls.key
```

Clients connecting to HTTPS endpoints secured with self-signed certificates must trust the certifactes or CA. See [internal/stack/envoy/develop.md](internal/stack/envoy/develop.md) if tinkering locally.

5. start containers
```sh
# docker compose build
docker compose up -d
```

Check zitadel health with `curl https://iam.${EDGE_DOMAIN_ROOT}/debug/healthz -k` or `docker exec -it edge_edge_1 /edge healthz`

#### Use the centralized IdP for authorization in Postgres via `pgo rest` (PostgREST API) as well as minio-s3, NATS etc

edge so far creates the OIDC clients on ZITADEL. a bit works needed to for configuring consumers of client secrets.
The idea is to use `edge` to serve config for each component, much like envoy control plane which is already embeded in edge for envoy to pull config dynamically.

For now, visit ZITADEL UI at http://iam.${EDGE_DOMAIN_ROOT}, login (see docker-compose.yaml) and regenerate client-secrets for oauth2-proxy and minio clients in edge project. Then

- update `pgo-config.yaml` with the values
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
curl https://api.${EDGE_DOMAIN_ROOT}/iam/users -k
```

##### `pgo pipeline`: Debezium-compatible CDC for realtime-event/replication etc

The demo pgo-pipeline container syncs users from auth-db (in projections.users14 table) to app-db (in iam.users)

#### minio-s3
ensure minio MINIO_IDENTITY_OPENID_CLIENT_ID and MINIO_IDENTITY_OPENID_CLIENT_SECRET are set withc appropriate values. console ui is at http://minio.${EDGE_DOMAIN_ROOT}.

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
