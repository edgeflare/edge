import { Routes } from "@angular/router";
import { Kubectl } from "./kubectl";

export const KUBECTL_ROUTES: Routes = [
  {
    path: '',
    component: Kubectl,
    pathMatch: 'prefix',
  },
]
