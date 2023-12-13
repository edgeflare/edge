import { Component, Input } from '@angular/core';
import { ChartRelease, KubernetesResourceMetadata } from '@interfaces';
import { CommonModule } from '@angular/common';
import { ExpandableTable } from '@app/shared/components/expandable-table';
import { Editor } from '@app/shared/components/editor';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { YamlPipe } from '@app/shared/pipes';

@Component({
  selector: 'e-release-resources-table',
  standalone: true,
  imports: [CommonModule, ExpandableTable, Editor, MatProgressSpinnerModule, MatButtonModule, MatIconModule, YamlPipe],
  templateUrl: './release-resources-table.html',
  styles: ``
})
export class ReleaseResourcesTable {
  @Input() helmchartRelease!: ChartRelease;

  currentExpandedRow!: KubernetesResourceMetadata;
  columns = ['kind', 'name', 'namespace', 'apiVersion'];
  cellDefs = ['kind', 'metadata.name', 'metadata.namespace', 'apiVersion'];

  handleRowChange(rowData: KubernetesResourceMetadata) {
    this.currentExpandedRow = rowData;
  }
}
