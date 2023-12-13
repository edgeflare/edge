import { Routes } from '@angular/router';
import { Clusters } from './clusters';
import { ClustersTable } from './clusters-table/clusters-table';
import { ClusterInstance } from './cluster-instance/cluster-instance';
import { CreateCluster } from './create-cluster/create-cluster';
import { JoinNode } from './join-node/join-node';
import { DeleteNode } from './delete-node/delete-node';

export const CLUSTER_ROUTES: Routes = [
  {
    path: '',
    component: Clusters,
    children: [
      { path: 'create', component: CreateCluster },
      { path: ':clusterId/join', component: JoinNode },
      { path: ':clusterId/:nodeId/delete', component: DeleteNode },
      { path: ':clusterId', component: ClusterInstance },
      { path: '', component: ClustersTable, pathMatch: 'full' },
    ],
    pathMatch: 'prefix'
  },
]
