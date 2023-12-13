import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ChartRelease, KubernetesWorkload } from '@interfaces';
import { ExpandableTable } from '@app/shared/components/expandable-table';
import { Editor } from '@app/shared/components/editor';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { YamlPipe } from '@app/shared/pipes';

@Component({
  selector: 'e-release-workloads-table',
  standalone: true,
  imports: [CommonModule, ExpandableTable, Editor, MatProgressSpinnerModule, YamlPipe],
  templateUrl: './release-workloads-table.html',
  styles: ``
})
export class ReleaseWorkloadsTable {
  @Input() helmchartRelease!: ChartRelease;

  currentExpandedRow!: KubernetesWorkload;
  columns = ['kind', 'name', 'phase', 'age', 'replicas', 'ready', 'available'];
  cellDefs = ['kind', 'name', 'status.phase', 'status.startTime | timeago', 'status.replicas', 'status.readyReplicas', 'status.availableReplicas'];

  handleRowChange(rowData: KubernetesWorkload) {
    this.currentExpandedRow = rowData;
  }
}
