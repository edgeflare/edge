# manage k3s clusters and helm-charts anywhere

## What can `edge` do for you?
- manage [k3s](https://k3s.io), aka lightweight-[kubernetes](https://kubernetes.io), clusters on local or remote computers
- manage [helm](https://helm.sh) packaged containerized apps
- expose applications to the Internet using [traefik](https://traefik.io), [cert-manager](https://cert-manager.io) and [letsencrypt](https://letsencrypt.org)
- authenticate users to applications using [dex](https://dexidp.io)

## How to use `edge`?
`edge` is a Go binary with an embedded web UI built with [Angular](https://angular.io/). It runs on Linux, Windows, macOS, and as container.

### Install `edge`

Get from [Releases](https://github.com/edgeflare/edge/releases) page. Or

```shell
curl -sfLO https://raw.githubusercontent.com/edgeflare/edge/master/install.sh
chmod +x install.sh && ./install.sh
```

### Install k3s using `edge`

```shell
edge k3s install --host 10.164.0.11 --user admin
# or use flag aliases
edge k i -H 127.0.0.1 -u admin
```

### Join k3s agent or server node to a cluster

```shell
edge k join --server 10.164.0.11 -H 10.164.0.12 -u admin
edge k j -s 10.164.0.11 -H 10.164.0.12 -u admin --master # server in HA mode
```

#### few other example commands

```shell
edge k copy-kubeconfig -H 10.164.0.11 -u admin # alias: cpk
# Kubeconfig saved to /Users/<user>/.kube/10.164.0.11.config

edge k ls
# ID              Status          Version         Is HA           APIserver       CreatedAt 
# b5fb728e341e    Running         v1.28.4+k3s2    false           10.164.0.11     2023-12-13T00:09:36Z

edge k nodes --clusterid b5fb728e341e
# Node ID         IP              Role            Status          CreatedAt 
# 33e37c119a90    10.164.0.11     server          Running         2023-12-13T00:09:36Z
# 443b2b12f320    10.164.0.12     agent           Running         2023-12-13T00:09:36Z
```

### Uninstall / destroy k3s

```shell
edge k destroy -H 10.164.0.11 -u admin # alias: uninstall
edge k d -H 10.164.0.12 -u admin -a # if agent node
```

> **Sometimes scripts require sudo privileges without being prompted for a password. To verify or enable passwordless sudo access for a user on a remote SSH host, first, SSH into the host using ssh yourusername@remotehost. Then, use sudo visudo to edit the sudoers file and add or confirm the line `<yourusername> ALL=(ALL) NOPASSWD: ALL` is present and correctly formatted. Most cloud VMs nowadays have this settup.**


### WebUI

```shell
edge server # alias: s
# Web UI available at http://localhost:8080
# See edge s --help for more options
```

#### Explore readonly WebUI at [demo.edgeflare.io](https://demo.edgeflare.io)

##### manage helm-charts

|                                            |                                              |
|--------------------------------------------|----------------------------------------------|
| ![helm-catalog](docs/img/helm-catalog.png) | ![helm-install](docs/img/helm-install.png)   |
| ![helm-list](docs/img/helm-list.png)       | ![helm-release](docs/img/helm-release.png)   |

##### install k3s cluster and join nodes

<p align="center">
  <img src="docs/img/demo.gif" alt="demo">
</p>



## How to contribute to `edge`?

We welcome contributions to edge! If you're interested in helping improve this tool, please refer to our [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on how to get started.