import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable, of, throwError } from 'rxjs';
import { catchError, map, switchMap } from 'rxjs/operators';
import { environment } from '@env';
import { ChartMetadata, ChartRepo, ChartRepoIndex, ChartSpec } from '@interfaces';

@Injectable({
  providedIn: 'root'
})
export class HelmCatalogService {
  private chartRepositories: ChartRepo[] | null = null;

  private http = inject(HttpClient);

  /**
   * Fetches the list of Helm chart repositories.
   * @returns An Observable emitting an array of ChartRepo.
   */
  private fetchChartRepositories(): Observable<ChartRepo[]> {
    const url = `${environment.api}/catalog/helm/repos`;
    return this.http.get<ChartRepo[]>(url).pipe(
      catchError(error => throwError(() => error))
    );
  }

  /**
   * Retrieves the list of chart repositories. If the repositories are already loaded, it returns the cached version.
   * @returns An Observable emitting an array of ChartRepo.
   */
  getChartRepositories(): Observable<ChartRepo[]> {
    if (this.chartRepositories) {
      return of(this.chartRepositories);
    } else {
      return this.fetchChartRepositories().pipe(
        map(repos => {
          this.chartRepositories = repos;
          return repos;
        })
      );
    }
  }

  /**
   * Retrieves the URL of a chart repository by its name.
   * @param repoName The name of the repository.
   * @returns An Observable emitting the repository URL.
   */
  getRepositoryUrlByName(repoName: string): Observable<string> {
    return this.getChartRepositories().pipe(
      map(repos => {
        const repository = repos.find(r => r.name === repoName);
        if (!repository) {
          throw new Error(`Repository with name '${repoName}' not found.`);
        }
        return repository.url;
      }),
      catchError(error => throwError(() => error))
    );
  }

  getRepositoryNameByUrl(repoUrl: string): Observable<string> {
    return this.getChartRepositories().pipe(
      map(repos => {
        const repo = repos.find(r => r.url === repoUrl);
        if (!repo) {
          throw new Error(`Repository with name '${repoUrl}' not found.`);
        }
        return repo.name;
      }),
      catchError(err => throwError(() => err))
    );
  }

  /**
   * Retrieves the latest chart metadata from a specific repository.
   * @param repoName The name of the repository.
   * @returns An Observable emitting an array of the latest ChartMetadata.
   */
  getLatestChartsFromRepository(repoName: string): Observable<ChartMetadata[]> {
    const url = `${environment.api}/catalog/helm/repos/${repoName}/charts`;
    return this.http.get<ChartRepoIndex>(url).pipe(
      map(repoIndex => this.extractLatestCharts(repoIndex.entries)),
      catchError(error => throwError(() => error))
    );
  }

  /**
   * Extracts the latest charts from chart repository entries.
   * @param entries The chart entries from a repository.
   * @returns An array of ChartMetadata representing the latest charts.
   */
  private extractLatestCharts(entries: Record<string, ChartMetadata[]>): ChartMetadata[] {
    return Object.values(entries).map(charts => charts[0]);
  }

  /**
   * Retrieves all available versions of a specific chart from a repository.
   * @param repoName The name of the repository.
   * @param chartName The name of the chart.
   * @returns An Observable emitting an array of chart versions.
   */
  getChartVersions(repoName: string, chartName: string): Observable<string[]> {
    const url = `${environment.api}/catalog/helm/repos/${repoName}/charts`;
    return this.http.get<ChartRepoIndex>(url).pipe(
      map(repoIndex => {
        const chartVersions = repoIndex.entries[chartName];
        return chartVersions ? chartVersions.map(chart => chart.version) : [];
      }),
      catchError(error => throwError(() => error))
    );
  }

  /**
   * Retrieves the latest chart specification from a repository.
   * @param repoName The name of the repository.
   * @param chartName The name of the chart.
   * @returns An Observable emitting the latest ChartSpec.
   */
  getChartSpec(repoName: string, chartName: string, chartVersion?: string): Observable<ChartSpec> {
    return this.getChartVersions(repoName, chartName).pipe(
      map(versions => versions[0]),
      switchMap(latestVersion => {
        // Use the supplied chartVersion if available, otherwise use latestVersion
        const versionToUse = chartVersion ?? latestVersion;
        return this.getChartFromRepository(repoName, chartName, versionToUse);
      }),
      catchError(error => throwError(() => error))
    );
  }

  /**
   * Retrieves a specific chart specification from a repository.
   * @param repoName The name of the repository.
   * @param chartName The name of the chart.
   * @param version The version of the chart.
   * @returns An Observable emitting the ChartSpec.
   */
  getChartFromRepository(repoName: string, chartName: string, version: string): Observable<ChartSpec> {
    const url = `${environment.api}/catalog/helm/repos/${repoName}/charts/${chartName}/${version}`;
    return this.http.get<ChartSpec>(url).pipe(
      catchError(error => throwError(() => error))
    );
  }
}
