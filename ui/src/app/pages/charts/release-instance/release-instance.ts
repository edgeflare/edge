import { Component, Input, OnDestroy, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { CattleHelmChart, ChartFile, ChartRelease } from '@app/shared/interfaces';
import { AppService } from '@app/core/services';
import { Subject, of, switchMap, takeUntil } from 'rxjs';
import { FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { HelmChartReleaseService } from '@services';
import { ActivatedRoute, Router, RouterModule } from '@angular/router';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { MatDialog, MatDialogModule } from '@angular/material/dialog';
import { ConfirmDeleteDialog } from '@components/confirm-delete-dialog';
import { environment as env } from '@env';
import { decodeBase64, jsonToYaml } from '@utils';
import { DecodeBase64Pipe, MarkdownToHtmlPipe, YamlPipe } from '@app/shared/pipes';
import { Editor } from '@components/editor';
import { ExpandableTable } from '@components/expandable-table';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatSelectChange, MatSelectModule } from '@angular/material/select';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatTabsModule } from '@angular/material/tabs';
import { ReleaseResourcesTable } from './release-resources-table';
import { ReleaseWorkloadsTable } from './release-workloads-table';
import { ChartReleaseData } from '@app/shared/services/helmchart-release.service';
import { Title } from '@angular/platform-browser';

@Component({
  selector: 'e-release-instance',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, RouterModule, MatSnackBarModule, MatDialogModule, MatIconModule, MatInputModule,
    MatButtonModule, MatFormFieldModule, MatSelectModule, MatProgressSpinnerModule, ConfirmDeleteDialog, YamlPipe, DecodeBase64Pipe,
    MatToolbarModule, MatTabsModule, MarkdownToHtmlPipe, Editor, ExpandableTable, ReleaseResourcesTable, ReleaseWorkloadsTable],
  templateUrl: './release-instance.html',
  styles: ``
})
export class ReleaseInstance implements OnInit, OnDestroy {
  private destroy$ = new Subject<void>();
  private appService = inject(AppService);
  private helmChartsService = inject(HelmChartReleaseService);
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private snackBar = inject(MatSnackBar);
  private title = inject(Title);
  private dialog = inject(MatDialog);
  isHandset$ = this.appService.isHandset$;
  @Input() releaseName!: string;
  @Input() releaseNamespace!: string;
  cattleHelmChart!: CattleHelmChart;
  repoName!: string;
  chartName!: string;
  chartVersion!: string;
  chartFormGroup!: FormGroup;
  namespaces: string[] = [];
  chartVersions!: string[];
  mode: 'install' | 'upgrade' | 'view' | 'reinstall' = 'install';
  helmchartRelease: ChartRelease | any = {};
  chartCustomValues!: string | any;
  isLoading = true;

  ngOnInit(): void {
    this.initializeMode();
    this.initializeForm();
    this.title.setTitle(`edge - ${this.releaseName === undefined ? this.chartName : this.releaseName }`)
    this.loadDataBasedOnMode();
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  private initializeMode(): void {
    const path = this.route.snapshot.url[this.route.snapshot.url.length - 1].path;
    this.mode = path === 'upgrade' ? 'upgrade' :
      (path === 'install' ? 'install' :
        (path === 'reinstall' ? 'reinstall' : 'view'));


    if (this.mode !== 'install') {
      this.releaseNamespace = this.route.snapshot.params['releaseNamespace'];
      this.releaseName = this.route.snapshot.params['releaseName'];
    }

    this.route.queryParams.subscribe(params => {
      this.repoName = params['repo'];
      this.chartName = params['chart'];
      this.chartVersion = params['version'];
    });
  }

  kubernetesNameValidator(control: FormControl): { [key: string]: any } | null {
    const value = control.value;
    const kubernetesRegex = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/;

    if (value && value.length <= 253 && kubernetesRegex.test(value)) {
      return null; // valid
    }

    return { 'invalidKubernetesName': true }; // invalid
  }

  private initializeForm(): void {
    this.chartFormGroup = new FormGroup({
      chartReleaseName: new FormControl(this.releaseName, [Validators.required, this.kubernetesNameValidator]),
      chartReleaseNamespace: new FormControl(this.releaseNamespace || 'default', [Validators.required]),
      chartVersion: new FormControl('', [Validators.required])
    });

    // Disable form controls if not in 'install' mode
    if (this.mode !== 'install') {
      this.chartFormGroup.get('chartReleaseName')?.disable();
      this.chartFormGroup.get('chartReleaseNamespace')?.disable();
    }

    // If in 'view' mode, also disable chartVersion control
    if (this.mode === 'view') {
      this.chartFormGroup.get('chartVersion')?.disable();
    }
  }

  private loadDataBasedOnMode(): void {
    this.isLoading = true;

    if (this.mode === 'install') {
      this.loadInstallableChartData();
    } else {
      this.helmChartsService.getCattleHelmChart(this.releaseNamespace, this.releaseName)
        .pipe(
          switchMap(cattleHelmChart => {
            this.cattleHelmChart = cattleHelmChart;
            this.chartCustomValues = cattleHelmChart.spec.valuesContent;
            if (cattleHelmChart.installer_job_completed === true) {
              return this.helmChartsService.loadUpgradableChartData(this.repoName, this.chartName, this.releaseNamespace, this.releaseName);
            } else {
              this.loadInstallableChartData();
              return of(null); // or return throwError() to handle it as an error
            }
          }),
          takeUntil(this.destroy$)
        ).subscribe({
          next: (helmchartRelease) => {
            if (helmchartRelease) {
              this.processChartReleaseData(helmchartRelease);
            } else {
              this.isLoading = false;
              this.snackBar.open(`${this.mode === 'view' ? 'install' : this.mode} not yet complete`, 'Close', { duration: 3000 });
            }
          }
        });
    }
  }

  private loadInstallableChartData(): void {
    this.helmChartsService.loadInstallableChartData(this.repoName, this.chartName, this.chartVersion)
      .pipe(takeUntil(this.destroy$)
      ).subscribe({
        next: (helmchartRelease) => {
          this.processChartReleaseData(helmchartRelease);
        },
        error: (error) => {
          console.log(error);
          this.isLoading = false;
        }
      });
  }

  private processChartReleaseData(data: ChartReleaseData): void {
    this.namespaces = data.namespaces;
    this.chartVersions = data.availableVersions;
    this.chartFormGroup.get('chartVersion')?.setValue(this.chartVersion || this.chartVersions[0]);

    // Handle data based on the mode
    if (this.mode === 'install' || this.mode === 'reinstall') {
      this.helmchartRelease.chart = data.chart;
      this.chartCustomValues = '';
    } else {
      this.helmchartRelease = data.release;
      this.chartCustomValues = this.helmchartRelease?.config || '';
    }

    this.isLoading = false;
  }

  installOrUpgrade(): void {
    if (this.chartFormGroup.valid) {
      const chart: CattleHelmChart = {
        apiVersion: 'helm.cattle.io/v1',
        kind: 'HelmChart',
        metadata: {
          name: this.mode === 'install' ? this.chartFormGroup.value.chartReleaseName : this.releaseName,
          namespace: this.mode === 'install' ? this.chartFormGroup.value.chartReleaseNamespace : this.releaseNamespace
        },
        spec: {
          chart: this.chartName,
          repo: this.repoName, // Service replaces the repoName with the repoURL
          targetNamespace: this.mode === 'install' ? this.chartFormGroup.value.chartReleaseNamespace : this.releaseNamespace || this.releaseNamespace,
          version: this.chartFormGroup.value.chartVersion,
          valuesContent: this.chartCustomValues
        }
      };

      this.helmChartsService.installOrUpgradeChartRelease(chart).subscribe({
        next: () => { // next: (response)
          this.snackBar.open(`${chart.spec.chart} ${this.mode}${this.mode === 'upgrade' ? 'd' : 'ed'} successfully`, 'Close', { duration: 3000 });
          this.isLoading = true;
          setTimeout(() => {
            this.router.navigate(
              ['/apps', chart.metadata.namespace, chart.metadata.name],
              { queryParams: { chart: chart.spec.chart, repo: this.repoName, version: chart.spec.version } }
            );
          }, 5000); // wait 5 seconds before redirecting or, better, start polling for release status
        },
        error: (error) => {
          this.snackBar.open(`Error installing ${chart.spec.chart}: ${error.message}`, 'Close', { duration: 3000 });
        }
      });
    }
  }

  handleCustomValuesChange(content: string): void {
    this.chartCustomValues = content;
  }

  editRelease(): void {
    this.router.navigate(['/apps', this.releaseNamespace, this.releaseName, 'upgrade'],
      { queryParams: { chart: this.chartName, repo: this.repoName, version: this.chartVersion } });
  }

  cancelEdit(): void {
    this.router.navigate(['/apps', this.releaseNamespace, this.releaseName],
      { queryParams: { chart: this.chartName, repo: this.repoName, version: this.chartVersion } });
  }

  // heavy-handed approach to refresh the window
  reloadhWindow(): void {
    window.location.reload();
  }

  reinstallFailedRelease() {
    this.router.navigate(['/apps', this.releaseNamespace, this.releaseName, 'reinstall'],
      { queryParams: { chart: this.chartName, repo: this.repoName, version: this.chartVersion } });
  }

  openConfirmDeleteDialog() {
    const dialogRef = this.dialog.open(ConfirmDeleteDialog, {
      data: {
        itemName: this.releaseName,
        itemType: 'release',
        deleteUrl: `${env.api}/cattle/namespaces/${this.cattleHelmChart.metadata.namespace}/helmcharts/${this.cattleHelmChart.metadata.name}`
      }
    });

    dialogRef.componentInstance.deletionComplete.subscribe(() => {
      this.onDeletionComplete();
    });
  }

  onDeletionComplete() {
    setTimeout(() => {
      this.router.navigate(['/apps']);
    }, 5000); // Wait for 5 seconds before refreshing the data
  }

  get readme(): string {
    const readme = this.helmchartRelease?.chart?.files?.find((file: ChartFile) => file.name.toLowerCase() === 'readme.md');
    return readme ? readme.data : '';
  }

  get values(): string {
    if (this.mode === 'install') {
      const values = this.helmchartRelease?.chart?.files?.find((file: ChartFile) => file.name.toLowerCase() === 'values.yaml');
      return values ? decodeBase64(values.data) : '';
    } else {
      return jsonToYaml(this.helmchartRelease?.chart?.values);
    }
  }

  onChartVersionChange(event: MatSelectChange) {
    const selectedValue = event.value;

    if (selectedValue === this.chartVersion) {
      return;
    }

    this.chartVersion = selectedValue;
    this.router.navigate([], {
      queryParams: {
        repo: this.repoName,
        chart: this.chartName,
        version: this.chartVersion
      },
      queryParamsHandling: 'merge' // merge with the current query params
    });

    this.loadDataBasedOnMode();
  }
}
