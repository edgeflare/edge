export interface KubernetesResource {
  apiVersion: string;
  kind: string;
  metadata: KubernetesResourceMetadata;
  spec: object;
}

export interface KubernetesWorkload {
  name: string;
  kind: string;
  status: KubernetesWorkloadStatus;
}

export interface KubernetesWorkloadStatus {
  replicas: number;
  readyReplicas: number;
  updatedReplicas: number;
  availableReplicas: number;
}

export interface KubernetesResourceMetadata {
  name: string;
  namespace: string;
  labels?: object;
  annotations?: object;
}
