import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: 'catalog',
    loadChildren: () => import('./pages/catalog').then(r => r.CATALOG_ROUTES)
  },
  {
    path: 'gateways',
    loadChildren: () => import('./pages/gateways').then(r => r.GATEWAY_ROUTES)
  },
  {
    path: 'apps',
    loadChildren: () => import('./pages/charts').then(r => r.CHARTS_ROUTES)
  },
  {
    path: 'clusters',
    loadChildren: () => import('./pages/clusters').then(r => r.CLUSTER_ROUTES)
  },
  {
    path: 'kubectl',
    loadChildren: () => import('./pages/kubectl').then(r => r.KUBECTL_ROUTES)
  },
  {
    path: 'xterm',
    loadChildren: () => import('./pages/xterm').then(r => r.XTERM_ROUTES)
  },
  {
    path: 'privacy-policy',
    loadComponent: () => import('./pages/privacy-policy/privacy-policy').then(r => r.PrivacyPolicy)
  },
  { path: '**', redirectTo: '/catalog'}
];
