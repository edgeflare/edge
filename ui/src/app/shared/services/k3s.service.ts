import { Injectable, inject } from '@angular/core';
import { BehaviorSubject, Observable, catchError, throwError } from 'rxjs';
import { K3sCluster, K3sJoinRequest, K3sNode, K3sUninstallRequest } from '@interfaces';
import { HttpClient } from '@angular/common/http';
import { environment } from '@env';

@Injectable({
  providedIn: 'root'
})
export class K3sService {
  clusters$: BehaviorSubject<K3sCluster[]> = new BehaviorSubject<K3sCluster[]>([]);
  nodes$: BehaviorSubject<K3sNode[]> = new BehaviorSubject<K3sNode[]>([]);

  private http = inject(HttpClient);

  /**
   * Fetches and updates the list of clusters.
   * @returns void.
   */
  getClusters(): void {
    this.http.get<K3sCluster[]>(`${environment.api}/clusters`).pipe(
      catchError(error => throwError(() => error))
    ).subscribe(clusters => this.clusters$.next(clusters));
  }

  /**
   * Fetches and updates the list of nodes.
   * @returns void.
   */
  getNodes(clusterId: string): void {
    this.http.get<K3sNode[]>(`${environment.api}/clusters/${clusterId}/nodes`).pipe(
      catchError(error => throwError(() => error))
    ).subscribe(nodes => this.nodes$.next(nodes));
  }

  getK3sVersions(): Observable<string[]> {
    return this.http.get<string[]>(`${environment.api}/clusters/versions`).pipe(
      catchError(error => throwError(() => error))
    );
  }

  createCluster(cluster: K3sCluster): Observable<any> {
    return this.http.post(`${environment.api}/clusters/install`, cluster, { responseType: 'text' }).pipe(
      catchError(error => throwError(() => error))
    );
  }

  uninstallK3s(req: K3sUninstallRequest, clusterId: string, nodeId: string): Observable<any> {
    return this.http.post(`${environment.api}/clusters/${clusterId}/nodes/${nodeId}`, req, { responseType: 'text' }).pipe(
      catchError(error => throwError(() => error))
    );
  }

  joinNode(req: K3sJoinRequest, clusterId: string): Observable<any> {
    return this.http.post(`${environment.api}/clusters/${clusterId}/join`, req, { responseType: 'text' }).pipe(
      catchError(error => throwError(() => error))
    );
  }

}
