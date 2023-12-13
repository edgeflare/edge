import { Component, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterOutlet } from '@angular/router';
import { CattleReleasesTable } from './cattle-releases-table';
import { ReleaseInstance } from './release-instance';
import { MatToolbarModule } from '@angular/material/toolbar';
import { Title } from '@angular/platform-browser';

@Component({
  selector: 'e-charts',
  standalone: true,
  imports: [CommonModule, RouterOutlet, ReleaseInstance, CattleReleasesTable, MatToolbarModule],
  templateUrl: './charts.html',
  styles: ``
})
export class Charts implements OnInit {
  private title = inject(Title);

  ngOnInit(): void {
    this.title.setTitle('edge - installed apps');
  }
}
