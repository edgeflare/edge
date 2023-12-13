import { Component, OnDestroy, OnInit } from '@angular/core';
import { CattleHelmChart } from '@interfaces';
import { Subject, takeUntil } from 'rxjs';
// import { ErrorHandlerService } from '@services';
import { ConfirmDeleteDialog } from '@components/confirm-delete-dialog';
import { environment as env } from '@env';
import { MatDialog } from '@angular/material/dialog';
import { Router, RouterModule } from '@angular/router';
import { HelmCatalogService } from '@services';
import { CommonModule } from '@angular/common';
import { HelmChartReleaseService } from '@app/shared/services/helmchart-release.service';
import { Editor } from '@app/shared/components/editor';
import { ExpandableTable } from '@app/shared/components/expandable-table';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { YamlPipe } from '@app/shared/pipes';
import { MatToolbarModule } from '@angular/material/toolbar';

@Component({
  selector: 'e-releases-table',
  standalone: true,
  imports: [CommonModule, Editor, ExpandableTable, MatButtonModule, MatIconModule, ConfirmDeleteDialog, MatProgressSpinnerModule,
    YamlPipe, RouterModule, MatToolbarModule],
  templateUrl: './cattle-releases-table.html',
  styles: ``
})
export class CattleReleasesTable implements OnInit, OnDestroy {
  cattleHelmCharts: CattleHelmChart[] | undefined = undefined;
  isLoading = true;
  private destroy$ = new Subject<void>();
  isShowDetails = false;
  columns = ['name', 'namespace', 'chart', 'version', 'age'];
  cellDefs = ['metadata.name', 'spec.targetNamespace', 'spec.chart | slice:-28', 'spec.version', 'metadata.creationTimestamp | timeago'];
  currentExpandedRow?: CattleHelmChart;

  constructor(
    private chartReleaseService: HelmChartReleaseService,
    // private errorHandler: ErrorHandlerService,
    private dialog: MatDialog,
    private router: Router,
    private catalogService: HelmCatalogService, // improve
  ) { }

  ngOnInit() {
    this.chartReleaseService.cattleHelmCharts$.pipe(
      takeUntil(this.destroy$)
    ).subscribe({
      next: (cattleHelmCharts) => {
        this.cattleHelmCharts = cattleHelmCharts;
        if (cattleHelmCharts && cattleHelmCharts.length > 0) {
          this.isLoading = false;
        } else if (cattleHelmCharts?.length === 0) {
          // Handle the empty array case here
          // Maybe show a message saying "No data available"
          this.isLoading = false;
        }
      },
      error: (error) => {
        // this.errorHandler.handleError(error);
        console.log(error)
        this.isLoading = false;
      }
    });

    this.chartReleaseService.fetchCattleHelmCharts();
  }

  ngOnDestroy() {
    this.destroy$.next();
    this.destroy$.complete();
  }

  handleRowChange(rowData: CattleHelmChart) {
    this.currentExpandedRow = rowData;
  }

  toggleDetails() {
    this.isShowDetails = !this.isShowDetails;
  }

  openConfirmDeleteDialog(chart: CattleHelmChart) {
    const dialogRef = this.dialog.open(ConfirmDeleteDialog, {
      // width: '300px',
      data: {
        itemName: chart.metadata.name,
        itemType: 'release',
        deleteUrl: `${env.api}/cattle/namespaces/${chart.metadata.namespace}/helmcharts/${chart.metadata.name}`
      }
    });

    dialogRef.componentInstance.deletionComplete.subscribe(() => {
      this.onDeletionComplete();
    });
  }

  onDeletionComplete() {
    setTimeout(() => {
      this.chartReleaseService.fetchCattleHelmCharts();
    }, 5000); // Wait for 5 seconds before refreshing the data
  }

  navigateToInstance(rowData: CattleHelmChart, upgrade = false) {
    this.catalogService.getRepositoryNameByUrl(rowData.spec.repo).subscribe({
      next: (repoName) => {
        const basePath = ['apps', rowData.metadata.namespace || 'kube-system', rowData.metadata.name];
        const path = upgrade ? [...basePath, 'upgrade'] : basePath;

        this.router.navigate(path, {
          queryParams: {
            chart: rowData.spec.chart,
            repo: repoName,
            version: rowData.spec.version
          }
        });
      },
      error: (error) => {
        console.log(error)
      }
    });

  }
}
