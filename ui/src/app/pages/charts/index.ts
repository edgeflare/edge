import { Routes } from '@angular/router';
import { CattleReleasesTable } from './cattle-releases-table';
import { Charts } from './charts';
import { ReleaseInstance } from './release-instance';

export const CHARTS_ROUTES: Routes = [
  {
    path: '',
    component: Charts,
    children: [
      { path: 'install', component: ReleaseInstance },
      { path: ':releaseNamespace/:releaseName', component: ReleaseInstance },
      { path: ':releaseNamespace/:releaseName/upgrade', component: ReleaseInstance },
      { path: ':releaseNamespace/:releaseName/reinstall', component: ReleaseInstance },
      { path: '', component: CattleReleasesTable, pathMatch: 'full' },
    ],
    pathMatch: 'prefix'
  },
]
