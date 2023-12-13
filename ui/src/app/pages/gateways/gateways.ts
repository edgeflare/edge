import { Component, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Network } from '@interfaces';
import { MatToolbarModule } from '@angular/material/toolbar';
import { RouterOutlet } from '@angular/router';
import { Title } from '@angular/platform-browser';

@Component({
  selector: 'e-gateways',
  standalone: true,
  imports: [CommonModule, MatToolbarModule, RouterOutlet],
  templateUrl: './gateways.html',
  styles: ``,
  providers: [],
})
export class Gateways implements OnInit {
  networks: Network[] | undefined = undefined;

  private title = inject(Title);

  ngOnInit() {
    this.title.setTitle(`edge - gateways`)
  }
}
