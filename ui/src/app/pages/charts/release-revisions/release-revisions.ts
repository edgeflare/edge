import { Component, Input, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ChartRelease } from '@interfaces';
import { HelmChartReleaseService } from '@services';
import { ExpandableTable } from '@components/expandable-table';
import { Editor } from '@components/editor';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { YamlPipe } from '@app/shared/pipes';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';

@Component({
  selector: 'e-release-revisions',
  standalone: true,
  imports: [CommonModule, ExpandableTable, Editor, MatProgressSpinnerModule, YamlPipe, MatIconModule, MatButtonModule, MatSnackBarModule],
  templateUrl: './release-revisions.html',
  styles: ``
})
export class ReleaseRevisions implements OnInit {
  @Input() releaseName!: string;
  @Input() releaseNamespace!: string;
  releaseRevisions!: ChartRelease[];
  currentExpandedRow!: ChartRelease;
  isShowDetails = false;
  isShowCustomValues = false;
  columns = ['revision', 'first_deployed', 'last_deployed', 'description', 'chartVersion'];
  cellDefs = ['version', 'info.first_deployed | date:"short"', 'info.last_deployed | date:"short"', 'info.description', 'chart.metadata.version'];
  isLoading = true;

  private helmchartReleaseService = inject(HelmChartReleaseService);
  private snackBar = inject(MatSnackBar);

  handleRowChange(rowData: ChartRelease) {
    this.currentExpandedRow = rowData;
  }

  toggleDetails() {
    this.isShowCustomValues = false;
    this.isShowDetails = !this.isShowDetails;
  }

  toggleShowCustomValues() {
    this.isShowDetails = false;
    this.isShowCustomValues = !this.isShowCustomValues;
  }

  ngOnInit(): void {
    this.helmchartReleaseService.getChartReleaseRevisions(this.releaseNamespace, this.releaseName)
      .subscribe({
        next: (revisions) => {
          this.releaseRevisions = revisions;
          this.isLoading = false;
        },
        error: (error) => {
          this.isLoading = false;
          this.snackBar.open(`Error loading release revisions: ${JSON.stringify(error)}`, 'Dismiss', {
            duration: 5000,
          });
        },
        complete: () => {}
      });
  }
}
