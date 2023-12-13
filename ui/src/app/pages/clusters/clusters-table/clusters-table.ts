import { Component, OnDestroy, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { K3sCluster } from '@app/shared/interfaces/cluster';
import { Subject, takeUntil } from 'rxjs';
import { K3sService } from '@services';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { ExpandableTable } from '@components/expandable-table';
import { RouterModule } from '@angular/router';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';

@Component({
  selector: 'e-clusters-table',
  standalone: true,
  imports: [CommonModule, MatSnackBarModule, ExpandableTable, RouterModule, MatIconModule, MatButtonModule, MatProgressSpinnerModule],
  templateUrl: './clusters-table.html',
  styles: ``
})
export class ClustersTable implements OnInit, OnDestroy {
  clusters: K3sCluster[] | undefined = undefined;
  isLoading = true;
  private destroy$ = new Subject<void>();
  isShowDetails = false;

  columns = ['id', 'version', 'status', 'is_ha', 'apiserver', 'age'];
  cellDefs = ['id', 'version', 'status', 'is_ha', 'apiserver', 'created_at | timeago'];
  currentExpandedRow?: K3sCluster;

  private k3sService = inject(K3sService);
  private snackBar = inject(MatSnackBar);

  ngOnInit() {
    this.k3sService.clusters$.pipe(
      takeUntil(this.destroy$)
    ).subscribe({
      next: (clusters) => {
        this.clusters = clusters;
        this.isLoading = false;
      },
      error: (error) => { // error: (error)
        this.snackBar.open(`Error loading clusters: ${error}`, 'Dismiss', {
          duration: 5000,
        });
        this.isLoading = false;
      },
      complete: () => {}
    });

    this.k3sService.getClusters();
  }

  ngOnDestroy() {
    this.destroy$.next();
    this.destroy$.complete();
  }

  handleRowChange(rowData: K3sCluster) {
    this.currentExpandedRow = rowData;
  }

  toggleDetails() {
    this.isShowDetails = !this.isShowDetails;
  }
}
