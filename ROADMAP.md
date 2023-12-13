# ROADMAP for Edge

## Introduction
`edge` simplifies managing clusters on both local and remote machines, handling TLS certificates, authenticating users, and much more, with a focus on edge computing solutions.

## Current State of the Project
- **Manage k3s Kubernetes Clusters**: Install, join, and uninstall clusters.
- **Manage Helm-Charts**: Basic management of containerized apps.
- **Basic and JWT Authentication**: Implemented in Golang backend, pending in Angular frontend.
- **Kubernetes Cluster Authentication**: Via KUBECONFIG or in-cluster ServiceAccount.
- **SSH Credential Management**: Supply for each install, join, uninstall operation.

## Roadmap

### Short-term Goals (Next 6 Months)
1. **Frontend Authentication**: Implement basic and JWT authentication in the Angular frontend.
2. **Enhanced Cluster Authentication**: Introduce options for supplying credentials via the frontend.
3. **Credential Type Expansion**: Develop a system for managing different credential types, starting with SSH credentials.
4. **Helm Chart Management Enhancements**:
   - Implement tracking of helm chart release history.
   - Develop a robust rollback mechanism.
   - Introduce features to show diffs in `values.yaml` for various releases.
5. **Initial Cloud Integration**: Begin work on storing cloud credentials for cloud-controller-manager integration.

### Mid-term Goals (6-12 Months)

1. **Exposing Applications**: Focus on robust solutions for application exposure:
   - Improve ingress capabilities.
   - Enhance TLS certificate management.
   - Develop more sophisticated authentication mechanisms.
2. **Wireguard VPN Integration**: Implement management of an Internet-facing server/gateway/supernode peer using Wireguard VPN, for applications on small devices.

3. **Cloud-Controller-Manager Integration**: Complete the integration with cloud-controller-managers to provision resources on cloud platforms.


## Contribution
We welcome contributions from the community. Whether you're a developer, a documentation expert, or an enthusiast in edge computing, your input is valuable. Please refer to our CONTRIBUTING.md for guidelines on how to contribute.

## Discussions and Feedback
We encourage discussions and feedback on our plans. Join us on [edgeflare.slack.com](https://edgeflare.slack.com), or directly on the GitHub issues page for open discussions.

## Acknowledgments
A special thanks to our contributors and supporting organizations who have played a significant role in the development of `edge`.

## Revision History
- 2023-12-12: Initial draft
