import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { GroupVersion } from '../interfaces';
import { Observable, catchError, forkJoin, throwError } from 'rxjs';
import { environment } from '@env';

@Injectable({
  providedIn: 'root'
})
export class KubectlService {
  private http = inject(HttpClient);

  // Fetches the list of namespaces
  getNamespaces(): Observable<string[]> {
    return this.http.get<string[]>(`${environment.api}/namespaces`).pipe(
      catchError(this.handleError)
    );
  }

  // Fetches the list of API resources
  getApiResources(): Observable<GroupVersion[]> {
    return this.http.get<GroupVersion[]>(`${environment.api}/api-resources`).pipe(
      catchError(this.handleError)
    );
  }

  // Fetches resources based on namespace, resource type and name
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  getResources(namespace: string, resource: string, resourceName: string): Observable<any> {
    const url = resourceName ?
      `${environment.api}/namespaces/${namespace}/${resource}/${resourceName}` :
      `${environment.api}/namespaces/${namespace}/${resource}`;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    return this.http.get<any>(url).pipe(
      catchError(this.handleError)
    );
  }

  // Applies a resource manifest to the server
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  applyResource(namespace: string, resource: string, resourceName: string, manifest: any): Observable<any> {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    return this.http.post<any>(`${environment.api}/namespaces/${namespace}/${resource}/${resourceName}`, manifest).pipe(
      catchError(this.handleError)
    );
  }

  // Deletes a resource from the server
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  deleteResource(namespace: string, resource: string, resourceName: string): Observable<any> {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    return this.http.delete<any>(`${environment.api}/namespaces/${namespace}/${resource}/${resourceName}`).pipe(
      catchError(this.handleError)
    );
  }

  // Loads initial data required by the service
  loadInitialData(): Observable<{ groupVersions: GroupVersion[]; namespaces: string[] }> {
    return forkJoin({
      groupVersions: this.getApiResources(),
      namespaces: this.getNamespaces()
    });
  }

  private handleError(error: HttpErrorResponse) {
    return throwError(() => new Error(error.error || 'Server Error'));
  }

}
