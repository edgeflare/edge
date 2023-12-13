import { Component, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Title } from '@angular/platform-browser';
import { MatToolbarModule } from '@angular/material/toolbar';
import { RouterOutlet } from '@angular/router';

@Component({
  selector: 'e-clusters',
  standalone: true,
  imports: [CommonModule, MatToolbarModule, RouterOutlet],
  templateUrl: './clusters.html',
  styles: ``
})
export class Clusters implements OnInit {
  private title = inject(Title);

  ngOnInit() {
    this.title.setTitle(`edge - k3s clusters`)
  }
}
