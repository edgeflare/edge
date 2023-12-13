import { KubernetesResource, KubernetesResourceMetadata, KubernetesWorkload } from ".";

export interface ChartRelease {
  name: string;
  namespace: string;
  version: string;
  info: ChartInfo;
  config: object | string; // custom values as json
  chart: ChartSpec;
  manifest: KubernetesResource[];
  workloads: KubernetesWorkload[];
}

export interface ChartInfo {
  first_deployed: string;
  last_deployed: string;
  status: string;
  description: string;
  notes: string;
  deleted: Date | string;
}

export interface ChartSpec {
  metadata: ChartMetadata;
  lock: ChartLock;
  templates: ChartTemplate[];
  values: object; // default values as json
  schema: object | string;
  files: ChartFile[];
}

export interface ChartLock {
  digest: string;
  generated: string;
  dependencies: ChartDependency[];
}

export interface ChartMetadata {
  annotations?: {
    category: string;
    images?: string;
    licenses?: string;
  };
  apiVersion: string;
  appVersion: string;
  created: string;
  dependencies?: ChartDependency[];
  description: string;
  digest: string;
  home: string;
  icon: string;
  keywords: string[];
  maintainers: ChartMaintainer[];
  name: string;
  sources: string[];
  urls: string[];
  version: string;
}

export interface ChartDependency {
  condition?: string;
  name: string;
  repository: string;
  version: string;
  tags?: string[];
}

export interface ChartMaintainer {
  name: string;
  url: string;
}

export interface ChartFile {
  name: string;
  data: string;
}

export interface ChartTemplate {
  name: string;
  data: string;
}

export interface ChartRepo {
  name: string;
  url: string;
}

export interface ChartRepoIndex {
  apiVersion: string;
  entries: {
    [key: string]: ChartMetadata[];
  };
}

export interface CattleHelmChart {
  apiVersion: string;
  kind: string;
  metadata: KubernetesResourceMetadata;
  spec: CattleHelmChartSpec;
  installer_job_completed?: boolean;
  installer_job_logs?: string;
}

export interface CattleHelmChartSpec {
  chart: string;
  repo: string;
  targetNamespace: string;
  version: string;
  valuesContent: string;
  // set: [object]
}
