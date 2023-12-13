import { Routes } from "@angular/router";
import { Gateways } from "./gateways";
import { EarlyAccess } from "./early-access/early-access";
import { GwIndex } from "./gw-index/gw-index";

export const GATEWAY_ROUTES: Routes = [
  {
    path: '',
    component: Gateways,
    children: [
      { path: '', component: GwIndex, pathMatch: 'full' },
      { path: 'early-access', component: EarlyAccess },
    ],
    pathMatch: 'prefix'
  },
]
