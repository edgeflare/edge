import { Routes } from "@angular/router";
import { Xterm } from "./xterm";

export const XTERM_ROUTES: Routes = [
  {
    path: '',
    component: Xterm,
    pathMatch: 'prefix',
  },
]
