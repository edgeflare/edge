import { Component, Input, OnDestroy, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { K3sNode } from '@interfaces';
import { Subject, takeUntil } from 'rxjs';
import { K3sService } from '@services';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { ExpandableTable } from '@components/expandable-table';
import { Editor } from '@components/editor';
import { YamlPipe } from '@shared/pipes';
import { RouterModule } from '@angular/router';

@Component({
  selector: 'e-cluster-instance',
  standalone: true,
  imports: [CommonModule, MatProgressSpinnerModule, MatIconModule, MatButtonModule, ExpandableTable, Editor, YamlPipe, RouterModule],
  templateUrl: './cluster-instance.html',
  providers: [YamlPipe],
})
export class ClusterInstance implements OnInit, OnDestroy {
  @Input() clusterId!: string;
  nodes: K3sNode[] | undefined = undefined;
  isLoading = true;
  private destroy$ = new Subject<void>();
  isShowDetails = false;

  columns = ['id', 'ip', 'role', 'status', 'age'];
  cellDefs = ['id', 'ip', 'role', 'status', 'created_at | timeago'];
  currentExpandedRow?: K3sNode;

  private k3sService = inject(K3sService);
  private snackBar = inject(MatSnackBar);

  ngOnInit() {
    this.k3sService.nodes$.pipe(
      takeUntil(this.destroy$)
    ).subscribe({
      next: (nodes) => {
        this.nodes = nodes;
        this.isLoading = false;
      },
      error: (error) => { // error: (error)
        this.snackBar.open(`Error loading nodes: ${error}`, 'Dismiss', {
          duration: 5000,
        });
        this.isLoading = false;
      },
      complete: () => {}
    });

    this.k3sService.getNodes(this.clusterId);
  }

  ngOnDestroy() {
    this.destroy$.next();
    this.destroy$.complete();
  }

  handleRowChange(rowData: K3sNode) {
    this.currentExpandedRow = rowData;
  }

  toggleDetails() {
    this.isShowDetails = !this.isShowDetails;
  }
}
