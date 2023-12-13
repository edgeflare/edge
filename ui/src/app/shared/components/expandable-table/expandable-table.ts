import { AfterViewInit, ChangeDetectorRef, Component, EventEmitter, Input, Output, TemplateRef, ViewChild } from '@angular/core';
import { animate, state, style, transition, trigger } from '@angular/animations';
import { MatTableDataSource, MatTableModule } from '@angular/material/table';
import { MatPaginator, MatPaginatorModule } from '@angular/material/paginator';
import { MatSort, MatSortModule } from '@angular/material/sort';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { TablePipe } from './table.pipe';
import {
  CommonModule, CurrencyPipe, DatePipe, DecimalPipe, JsonPipe, KeyValuePipe, LowerCasePipe,
  PercentPipe, SlicePipe, TitleCasePipe, UpperCasePipe
} from '@angular/common';
import { TimeAgoPipe } from '@app/shared/pipes';

interface TemplateContext<T> {
  $implicit: T;
}

@Component({
  selector: 'e-expandable-table',
  standalone: true,
  imports: [CommonModule, MatTableModule, MatIconModule, MatProgressSpinnerModule, MatSortModule, MatPaginatorModule, TablePipe],
  templateUrl: './expandable-table.html',
  styleUrl: './expandable-table.scss',
  animations: [
    trigger('detailExpand', [
      state('collapsed', style({ height: '0px', minHeight: '0' })),
      state('expanded', style({ height: '*' })),
      transition('expanded <=> collapsed', animate('225ms cubic-bezier(0.4, 0.0, 0.2, 1)')),
    ]),
  ],
  providers: [
    TablePipe, DatePipe, SlicePipe, KeyValuePipe, CurrencyPipe, DecimalPipe,
    PercentPipe, LowerCasePipe, UpperCasePipe, TitleCasePipe, JsonPipe, TimeAgoPipe
  ]
})
export class ExpandableTable<T> implements AfterViewInit {
  constructor(private cdr: ChangeDetectorRef) { }

  // undefined for mat-spinner till data is loaded
  @Input() set data(value: T[] | undefined) {
    this.dataSource = new MatTableDataSource(value);
    if (this.sort && this.paginator) {
      this.dataSource.sort = this.sort;
      this.dataSource.paginator = this.paginator;
    }
  }

  dataSource!: MatTableDataSource<T>;
  expandedRows: T[] = [];
  @ViewChild(MatSort) sort!: MatSort;
  @ViewChild(MatPaginator) paginator!: MatPaginator;
  @Input() disablePaginator = false;
  @Input() pageSizeOptions = [10, 25, 50];
  @Input() columns!: string[];
  @Input() cellDefs!: string[];
  // @Input() expandedDetailContent!: TemplateRef<T>;
  @Input() expandedDetailContent!: TemplateRef<TemplateContext<T>>;
  @Output() currentRowChanged: EventEmitter<T | any> = new EventEmitter<T | any>(); // eslint-disable-line @typescript-eslint/no-explicit-any

  toggleRow(element: T, event?: MouseEvent) {
    if (event) {
      event.stopPropagation();
    }

    if (this.isExpanded(element)) {
      this.expandedRows = this.expandedRows.filter((row) => row !== element);
      this.currentRowChanged.emit(null);
    } else {
      this.expandedRows.push(element);
      this.currentRowChanged.emit(element);
    }
  }

  isExpanded(element: T): boolean {
    return this.expandedRows.includes(element);
  }

  ngAfterViewInit() {
    this.dataSource.sort = this.sort;
    this.dataSource.paginator = this.paginator;
    this.cdr.detectChanges();
  }

  getContext(element: T): TemplateContext<T> {
    return { $implicit: element };
  }

}
