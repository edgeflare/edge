import { Routes } from '@angular/router';
import { Catalog } from './catalog';
import { ChartsGrid } from './charts-grid';

export const CATALOG_ROUTES: Routes = [
  {
    path: '',
    component: Catalog,
    children: [
      { path: '', component: ChartsGrid, pathMatch: 'full' },
    ],
    pathMatch: 'prefix'
  },
]
