set xds_cluster socket_address to `address: 0.0.0.0` in `internal/util/envoy/bootstrap.yaml`

```sh
envoy --config-path internal/util/envoy/bootstrap.yaml --base-id 1
```

```sh
go run ./internal/util/envoy
```

```sh
curl -H 'Host: iam.example.local' localhost:10080 -L
```
