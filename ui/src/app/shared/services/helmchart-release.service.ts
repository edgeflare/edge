import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { BehaviorSubject, Observable, forkJoin, throwError } from 'rxjs';
import { catchError, switchMap } from 'rxjs/operators';
import { environment } from '@env';
import { CattleHelmChart, ChartRelease, ChartSpec } from '@interfaces';
import { HelmCatalogService } from './helmchart-catalog.service';

@Injectable({
  providedIn: 'root'
})
export class HelmChartReleaseService {
  /**
   * helm.sh/helm/v3/pkg/release
   */
  helmChartReleases$: BehaviorSubject<ChartRelease[] | undefined> = new BehaviorSubject<ChartRelease[] | undefined>(undefined);

  /**
   * helmcharts.helm.cattle.io
   */
  cattleHelmCharts$: BehaviorSubject<CattleHelmChart[] | undefined> = new BehaviorSubject<CattleHelmChart[] | undefined>(undefined);
  cattleHelmChart$: BehaviorSubject<CattleHelmChart | undefined> = new BehaviorSubject<CattleHelmChart | undefined>(undefined);


  private http = inject(HttpClient);
  private catalogService = inject(HelmCatalogService);

  /**
   * Loads data necessary for installing a chart, including available versions and namespaces.
   * @param repoName The name of the chart repository.
   * @param chartName The name of the chart.
   * @returns An Observable emitting ChartReleaseData.
   */
  public loadInstallableChartData(repoName: string, chartName: string, chartVersion?: string): Observable<ChartReleaseData> {
    return forkJoin({
      availableVersions: this.catalogService.getChartVersions(repoName, chartName),
      namespaces: this.getNamespaces(),
      chart: this.catalogService.getChartSpec(repoName, chartName, chartVersion)
    }).pipe(catchError(error => throwError(() => error)));
  }

  /**
   * Loads data necessary for upgrading a chart, including available versions, namespaces, and current release information.
   * @param repoName The name of the repository.
   * @param chartName The name of the chart.
   * @param namespace The Kubernetes namespace.
   * @param releaseName The name of the chart release.
   * @returns An Observable emitting ChartReleaseData.
   */
  public loadUpgradableChartData(repoName: string, chartName: string, namespace: string, releaseName: string): Observable<ChartReleaseData> {
    return forkJoin({
      availableVersions: this.catalogService.getChartVersions(repoName, chartName),
      namespaces: this.getNamespaces(),
      release: this.getChartRelease(namespace, releaseName),
    }).pipe(catchError(error => throwError(() => error)));
  }

  /**
   * Fetches the release data of a specific Helm chart.
   * @param namespace The Kubernetes namespace.
   * @param releaseName The name of the chart release.
   * @returns An Observable emitting ChartRelease, Observable<ChartRelease>
   */
  private getChartRelease(namespace: string, releaseName: string): Observable<ChartRelease> {
    const url = `${environment.api}/namespaces/${namespace}/helmcharts/${releaseName}/workloads`;
    return this.http.get<ChartRelease>(url).pipe(
      catchError(error => throwError(() => error))
    );
  }

  /**
   * Installs or upgrades a Helm chart release.
   * @param chart The CattleHelmChart object representing the chart to install or upgrade.
   * @param crdNamespace The Kubernetes namespace where the chart would be installed
   * @returns An Observable of CattleHelmChart, if successful.
   */
  public installOrUpgradeChartRelease(chart: CattleHelmChart, crdNamespace?: string): Observable<CattleHelmChart> {
    crdNamespace = crdNamespace || chart.metadata.namespace;
    return this.catalogService.getRepositoryUrlByName(chart.spec.repo).pipe(
      switchMap(repoUrl => {
        chart.spec.repo = repoUrl;
        const url = `${environment.api}/cattle/namespaces/${crdNamespace}/helmcharts`;
        return this.http.post<CattleHelmChart>(url, chart);
      }),
      catchError(error => throwError(() => error))
    );
  }

  /**
   * Retrieves all Helm chart releases, optionally filtered by namespace.
   * @param namespace The Kubernetes namespace to filter the releases (optional).
   * @returns void.
   */
  public fetchHelmChartReleases(namespace?: string): void {
    const url = namespace ? `${environment.api}/namespaces/${namespace}/helmcharts` : `${environment.api}/helmcharts`;
    this.http.get<ChartRelease[]>(url).pipe(
      catchError(error => throwError(() => error))
    ).subscribe(releases => this.helmChartReleases$.next(releases));
  }

  /**
   * Retrieves all Cattle Helm charts, optionally filtered by namespace.
   * @param namespace The Kubernetes namespace to filter the charts (optional).
   * @returns void.
   */
  public fetchCattleHelmCharts(namespace?: string): void {
    const url = namespace ? `${environment.api}/cattle/namespaces/${namespace}/helmcharts` : `${environment.api}/cattle/helmcharts`;
    this.http.get<CattleHelmChart[]>(url).pipe(
      catchError(error => throwError(() => error))
    ).subscribe(charts => this.cattleHelmCharts$.next(charts));
  }

  /**
   * Retrieves a specific Cattle Helm chart.
   * @param releaseNamespace The Kubernetes namespace of the chart release.
   * @param releaseName The name of the chart release (optional).
   * @returns void.
   */
  public fetchCattleHelmChart(releaseNamespace: string, releaseName?: string): void {
    const url = `${environment.api}/cattle/namespaces/${releaseNamespace}/helmcharts/${releaseName}`;
    this.http.get<CattleHelmChart>(url).pipe(
      catchError(error => throwError(() => error))
    ).subscribe(chart => this.cattleHelmChart$.next(chart));
  }

  /**
   * Retrieves a list of Kubernetes namespaces.
   * @returns An Observable emitting an array of namespace strings.
   */
  public getNamespaces(): Observable<string[]> {
    const url = `${environment.api}/namespaces`;
    return this.http.get<string[]>(url).pipe(catchError(error => throwError(() => error)));
  }

  /**
   * @param namespace
   * @param releaseName
   * @returns Observable<CattleHelmChart>
   */
  public getCattleHelmChart(namespace: string, releaseName: string): Observable<CattleHelmChart> {
    return this.http.get<CattleHelmChart>(`${environment.api}/cattle/namespaces/${namespace}/helmcharts/${releaseName}`);
  }
}

export interface ChartReleaseData {
  availableVersions: string[];
  namespaces: string[];
  release?: ChartRelease;
  chart?: ChartSpec;
}
