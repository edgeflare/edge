# DIY nocode backend around PostgreSQL on Kubernetes

[![CI](https://github.com/edgeflare/edge/actions/workflows/ci.yml/badge.svg)](https://github.com/edgeflare/edge/actions/workflows/ci.yml)
[![CodeQL](https://github.com/edgeflare/edge/actions/workflows/codeql.yml/badge.svg)](https://github.com/edgeflare/edge/actions/workflows/codeql.yml)
[![Release](https://github.com/edgeflare/edge/actions/workflows/release.yml/badge.svg)](https://github.com/edgeflare/edge/actions/workflows/release.yml)

This is a work-in-progress kubebuilder-based operator that integrates and manages lifecycle of

- [PostgreSQL database](https://www.postgresql.org)
- [ZITADEL Identity Provider](https://github.com/zitadel/zitadel)
- [PostgREST API](https://github.com/PostgREST/postgrest)
- [SeaweedFS S3](https://github.com/seaweedfs/seaweedfs) (not integrated yet)
- [edgeflare/pgo](https://github.com/edgeflare/pgo) (experimental) for realtime events etc
- whatever extras you need... well it's on Kubernetes

for a [Firebase](https://firebase.google.com)/[Supabase](https://supabase.com/)-like nocode backend.
For now, you'd be better of installing the components as regular k8s resources as in this README.md.
If you wanna experiment and possibly contribute, please see [CONTRIBUTING.md](./CONTRIBUTING.md).

The stack can be run as
- [docker-compose.yaml](./example/docker-compose.yaml) (incomplete... no plan yet to improve)
- Kubernetes resources: This README instructions
- A [Project](./example/project.yaml) CRD: simplest, robust but not ready yet

## Build your own nocode backend stack

```sh
git clone git@github.com:edgeflare/edge.git && cd edge
```

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

Before installing PostgREST (for REST API), we gotta to create required roles. First set the libpq env vars for `psql`

```sh
export PGPASSWORD=$(kubectl -n default get secrets example-pguser-postgres -o jsonpath={.data.PGPASSWORD} | base64 -d)
export PGDATABASE=main
export PGUSER=postgres
export PGPORT=5432
export PGSSLMODE=require
export PGHOST=$(kubectl -n envoy-gateway-system get gateway eg-default -o jsonpath='{.status.addresses[0].value}')  # something like 192.168.0.17 
```

And exec into `psql` console below SQL

```sql
CREATE ROLE anon;       -- for anonymous / public access
CREATE ROLE authn;      -- for authenticated users. RLS or authz-proxy for granular authorization
CREATE ROLE postgrest LOGIN PASSWORD 'postgrestpw';
GRANT anon TO authn;
GRANT authn TO postgrest;
```

Now install PostgREST

```sh
kubectl apply -f example/k8s/03-postgrest.yaml
```

This uses iam.example.local and api.example.local domains. Ensure they point to the Gateway IP (could be same as $PGHOST) eg by adding an entry to `/etc/hosts` like

```sh
192.168.0.17 api.example.local iam.example.local
```

## Use the centralized IdP for authorization in Postgres via PostgREST API

Any OIDC compliant Identity Provider (eg ZITADEL, Keycloak, Auth0) can be used.

```sh
export CONN_STRING="host=$PGHOST port=$PGPORT user=$PGUSER password=$PGPASSWORD dbname=$PGDATABASE sslmode=require"

kubectl get secrets zitadel-admin-sa -o jsonpath='{.data.zitadel-admin-sa\.json}' | base64 -d > __zitadel-admin-sa.json

export ZITADEL_ISSUER=http://iam.example.local # Why HTTPS? See https://discord.com/channels/927474939156643850/1343884049726312509
export ZITADEL_API=iam.example.local:80
export ZITADEL_KEY_PATH=__zitadel-admin-sa.json
export ZITADEL_JWK_URL=http://iam.example.local/oauth/v2/keys
export ZITADEL_ADMIN_PW=$(kubectl get secrets example-zitadel-firstinstance -o jsonpath='{.data.ZITADEL_FIRSTINSTANCE_ORG_HUMAN_PASSWORD}' | base64 -d)
```

Create OIDC clients etc, and refresh issuer JWK in database (for PostgREST). Look up the code and [https://github.com/PostgREST/postgrest/issues/1130](https://github.com/PostgREST/postgrest/issues/1130) to see what it does and why.

```sh
go run ./hack/cmdext/...
```

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
