import { Component, OnInit, inject } from '@angular/core';
import { FormControl, FormsModule, ReactiveFormsModule } from '@angular/forms';
import { ChartMetadata, ChartRepo } from '@interfaces';
import { debounceTime, distinctUntilChanged } from 'rxjs';
import { HelmCatalogService } from '@services';
import { CommonModule } from '@angular/common';

import { MatCardModule } from '@angular/material/card';
import { MatChipsModule } from '@angular/material/chips';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSelectModule } from '@angular/material/select';
import { MatToolbarModule } from '@angular/material/toolbar';
import { RouterModule } from '@angular/router';
import { Title } from '@angular/platform-browser';

@Component({
  selector: 'e-charts-grid',
  standalone: true,
  imports: [CommonModule, FormsModule, ReactiveFormsModule, MatCardModule, MatChipsModule, MatIconModule, MatInputModule, MatFormFieldModule,
    MatSelectModule, MatProgressSpinnerModule, MatToolbarModule, RouterModule],
  templateUrl: './charts-grid.html',
  styles: ``
})
export class ChartsGrid implements OnInit {
  latestChartEntries: ChartMetadata[] = [];
  filteredChartEntries: ChartMetadata[] = [];
  chartRepos: ChartRepo[] = [];
  selectedRepo!: ChartRepo;
  searchTerm = new FormControl();
  isLoading = true;

  private title = inject(Title);
  private catalogService = inject(HelmCatalogService);

  ngOnInit(): void {
    this.getChartRepos();
  }

  getChartRepos(): void {
    this.catalogService.getChartRepositories().subscribe(repos => {
      this.chartRepos = repos;

      if (this.chartRepos.length > 0) {
        this.selectedRepo = this.chartRepos[0];
        this.getLatestCharts(this.selectedRepo.name);
        this.setTitle();
      }
    });
  }

  onRepoChange(): void {
    this.getLatestCharts(this.selectedRepo.name);
    this.setTitle();
  }

  getLatestCharts(repoName: string): void {
    this.isLoading = true;
    this.filteredChartEntries = [];

    this.catalogService.getLatestChartsFromRepository(repoName).subscribe(charts => {
      this.latestChartEntries = charts;
      this.filteredChartEntries = this.latestChartEntries;

      this.isLoading = false;

      this.searchTerm.valueChanges.pipe(
        debounceTime(300),
        distinctUntilChanged()
      ).subscribe(term => {
        this.filteredChartEntries = this.latestChartEntries.filter(chart =>
          chart.name.toLowerCase().includes(term.toLowerCase()) ||
          chart.annotations?.category.toLowerCase().includes(term.toLowerCase()) ||
          chart.description.toLowerCase().includes(term.toLowerCase()) ||
          chart.keywords.some(keyword => keyword.toLowerCase().includes(term.toLowerCase()))
        );
      });
    });
  }

  private setTitle() {
    this.title.setTitle(`edge - apps catalog - ${this.selectedRepo.name}`);
  }

}
