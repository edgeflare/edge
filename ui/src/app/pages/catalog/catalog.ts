import { Component, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule, RouterOutlet } from '@angular/router';
import { Title } from '@angular/platform-browser';

@Component({
  selector: 'e-catalog',
  standalone: true,
  imports: [CommonModule, RouterOutlet, RouterModule],
  templateUrl: './catalog.html',
  styles: ``
})
export class Catalog implements OnInit {
  private title = inject(Title);

  ngOnInit(): void {
    this.title.setTitle('edge - apps catalog');
  }
}
