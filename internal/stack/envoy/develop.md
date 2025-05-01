TLS certicates

generate self-signed certificates for development purposes

```sh
openssl req -x509 -newkey rsa:4096 -keyout tls.key -out tls.crt -days 365 -nodes \
  -subj "/CN=ca.example.local" \
  -addext "subjectAltName=DNS:*.example.local,DNS:*.${EDGE_DOMAIN_ROOT}"
```

```sh
# debian-derived distros
sudo cp tls.crt /usr/local/share/ca-certificates/
sudo update-ca-certificates

# rpm-based distros
sudo cp tls.crt /etc/pki/ca-trust/source/anchors/
sudo update-ca-trust extract

# macOS
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ca.crt
security verify-cert -c tls.crt

# # remove cert
# sudo security find-certificate -c "iam.example.local" /Library/Keychains/System.keychain
# sudo security delete-certificate -c "iam.example.local" /Library/Keychains/System.keychain
```

set xds_cluster socket_address to `address: 0.0.0.0` in `internal/stack/envoy/bootstrap.yaml`

```sh
envoy --config-path internal/stack/envoy/bootstrap.yaml --base-id 1
```

```sh
curl -H 'Host: iam.example.local' localhost:10080 -L
```

expose privileged port on podman
```sh
podman machine ssh
sudo sh -c 'echo "net.ipv4.ip_unprivileged_port_start=80" >> /etc/sysctl.conf'
sudo sysctl -p
exit
```