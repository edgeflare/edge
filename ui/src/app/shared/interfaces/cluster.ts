export interface K3sCluster {
  id: string;
  status: string;
  version: string;
  is_ha: boolean;
  apiserver: string;
  created_at: string;
  updated_at: string;
}

export interface K3sNode {
  id: string;
  cluster_id: string;
  role: string;
  status: string;
  ip: string;
  created_at: string;
  updated_at: string;
}

export interface K3sInstallRequest {
  cluster: boolean;
  ssh: SSHClient;
  tls_san: string;
  k3s_args: string;
  version: string;
}

export interface K3sUninstallRequest {
  ssh: SSHClient;
  agent: boolean;
}

export interface K3sJoinRequest {
  ssh: SSHClient;
  server: string;
  token: string;
  master: boolean;
}

export interface SSHClient {
  host: string;
  user: string;
  password: string;
  keyfile: string;
  port: number;
}
