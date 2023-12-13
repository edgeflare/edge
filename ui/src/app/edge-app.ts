import { Component, OnDestroy, OnInit, ViewChild, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { NavigationEnd, Router, RouterModule, RouterOutlet } from '@angular/router';
import { AppService } from '@core/services/app.service';
import { Subscription, filter } from 'rxjs';

import { MatSidenav, MatSidenavModule } from '@angular/material/sidenav';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatIconModule } from '@angular/material/icon';
import { MatListModule } from '@angular/material/list';
import { RouteInfo } from '@interfaces';
import { Title } from '@angular/platform-browser';

@Component({
  selector: 'e-root',
  standalone: true,
  imports: [CommonModule, RouterOutlet, RouterModule, MatSidenavModule, MatToolbarModule, MatIconModule, MatListModule],
  templateUrl: './edge-app.html',
  styleUrl: './edge-app.scss',
  providers: [
    AppService,
  ]
})
export class AppComponent implements OnInit, OnDestroy {
  private appService = inject(AppService);
  private router = inject(Router);
  private title = inject(Title);

  @ViewChild('drawer') drawer!: MatSidenav;
  isHandset$ = this.appService.isHandset$;

  private routeSubscription: Subscription = new Subscription();

  ngOnInit() {
    this.title.setTitle('edge - k3s and charts manager');
    this.appService.loadConfig().subscribe((config) => {
      console.log(config);
    });

    this.routeSubscription = this.appService.isHandset$.subscribe(isHandset => {
      if (isHandset) {
        this.routeSubscription.add(
          this.router.events.pipe(
            filter(event => event instanceof NavigationEnd)
          ).subscribe(() => {
            this.drawer.close();
          })
        );
      }
    });
  }

  ngOnDestroy(): void {
    this.routeSubscription.unsubscribe();
  }

  routes: RouteInfo[] = [
    { name: 'Clusters', path: '/clusters', icon: 'lan' },
    { name: 'Catalog', path: '/catalog', icon: 'format_list_bulleted_add' },
    { name: 'Apps', path: '/apps', icon: 'apps' },
    { name: 'Gateways', path: '/gateways', icon: 'public' },
  ];
}
