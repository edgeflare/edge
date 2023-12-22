import { Component, OnDestroy, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatInputModule } from '@angular/material/input';
import { FormBuilder, FormGroup, FormsModule, ReactiveFormsModule, Validators } from '@angular/forms';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatSelectModule, MatSelectChange } from '@angular/material/select';
import { Editor } from '@components/editor';
import { AppService } from '@app/core/services';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { MatToolbarModule } from '@angular/material/toolbar';
import { KubectlService } from '@app/shared/services/kubectl.service';
import { GroupVersion } from '@app/shared/interfaces';
import { Subscription } from 'rxjs';
import { YamlPipe } from '@app/shared/pipes';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { ExpandableTable } from '@app/shared/components/expandable-table';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatButtonToggleModule } from '@angular/material/button-toggle';
import * as yaml from 'js-yaml';
// import { ChangeDetectorRef } from '@angular/core';

@Component({
  selector: 'e-kubectl',
  standalone: true,
  imports: [CommonModule, MatInputModule, FormsModule, ReactiveFormsModule, MatFormFieldModule, MatSelectModule, Editor, MatButtonToggleModule,
    MatSnackBarModule, MatToolbarModule, MatIconModule, YamlPipe, MatButtonModule, ExpandableTable, MatProgressSpinnerModule,
  ],
  templateUrl: './kubectl.html',
  styles: ``
})
export class Kubectl implements OnInit, OnDestroy {
  private appService = inject(AppService);
  private fb = inject(FormBuilder);
  private snackBar = inject(MatSnackBar);
  private kubectlService = inject(KubectlService);
  // private cdr = inject(ChangeDetectorRef);

  isHandset$ = this.appService.isHandset$;
  kubectlForm!: FormGroup;
  namespaces: string[] = [];
  selectedResource!: GroupVersion;
  groupVersions!: GroupVersion[];
  manifests: [] = [];
  resourceDefinition!: object | string;
  isNamespaceDisabled: boolean = false;
  verbs: string[] = [];
  editorMode = 'json';
  private subscriptions = new Subscription();

  isLoading = true;
  isShowDetails = false;
  currentExpandedRow?: object;
  columns = ['apiVersion', 'kind', 'name', 'namespace', 'age'];
  cellDefs = ['apiVersion', 'kind', 'metadata.name', 'metadata.namespace', 'metadata.creationTimestamp | timeago'];

  constructor() {
    this.initForm();
  }

  ngOnInit(): void {
    this.subscriptions.add(this.loadInitialData());
    this.setupVerbListener();
  }

  ngOnDestroy(): void {
    this.subscriptions.unsubscribe();
  }

  private initForm() {
    this.kubectlForm = this.fb.group({
      groupVersion: ['v1'],
      resource: ['pods'],
      namespace: ['default'],
      resourceName: [''],
      verb: [''],
      resourceDefinition: [''],
    });
  }

  toggleEditorMode(mode: string) {
    this.editorMode = mode;
  }

  private loadInitialData(): void {
    this.subscriptions.add(
      this.kubectlService.loadInitialData().subscribe({
        next: data => this.handleDataLoad(data),
        error: error => this.handleError(error)
      })
    );
  }

  private handleDataLoad(data: { groupVersions: GroupVersion[]; namespaces: string[] }) {
    this.groupVersions = data.groupVersions;
    this.namespaces = data.namespaces;
    this.updateFormWithInitialData();
    this.isLoading = false;
  }

  private updateFormWithInitialData() {
    this.kubectlForm.patchValue({
      groupVersion: this.groupVersions[0]?.groupVersion,
      resource: this.groupVersions[0]?.resources[0]?.name,
      namespace: this.namespaces[0] || 'default',
      verb: this.groupVersions[0]?.resources[0]?.verbs[0] || ''
    });
  }

  private setupVerbListener() {
    this.kubectlForm.get('verb')?.valueChanges.subscribe(verb => {
      this.kubectlForm.get('resourceName')?.setValidators(verb === 'list' ? null : Validators.required);
      this.kubectlForm.get('resourceName')?.updateValueAndValidity();
    });
  }

  onGroupVersionChange(event: MatSelectChange) {
    this.manifests = []

    const selectedValue = event.value;
    this.selectedResource = selectedValue;

    // Find the selected API resource
    const selectedApiResource = this.groupVersions
      .find(gv => gv.groupVersion === selectedValue.groupVersion)
      ?.resources.find(res => res.name === selectedValue.resource.name);

    // Update verbs and exclude 'watch' and 'deletecollection'
    this.verbs = selectedApiResource?.verbs.filter(verb => verb !== 'watch' && verb !== 'deletecollection' && verb !== 'patch') || [];
    this.isNamespaceDisabled = !selectedApiResource?.namespaced;

    // Disable namespace if resource is not namespaced
    if (this.isNamespaceDisabled) {
      this.kubectlForm.get('namespace')?.disable();
    } else {
      this.kubectlForm.get('namespace')?.enable();
    }

    // Update form values
    this.kubectlForm.patchValue({
      groupVersion: selectedValue.groupVersion,
      resource: selectedValue.resource.name,
      verb: null,
      namespace: this.isNamespaceDisabled ? 'N/A' : this.kubectlForm.get('namespace')?.value || 'default'
    });
  }

  // Handle content changes
  handleCustomValuesChange(content: string): void {
    try {
      this.resourceDefinition = this.editorMode === 'json' ? JSON.parse(content) : yaml.load(content);
    } catch (e) {
      this.snackBar.open(`Error parsing ${this.editorMode} ${JSON.stringify(e)}`, 'Dismiss', { duration: 5000 });
    }
  }

  handleAction() {
    switch (this.kubectlForm.value.verb) {
      case 'get':
        this.getResource();
        break;
      case 'list':
        this.listResources();
        break;
      case 'delete':
        this.deleteResource();
        break;
      case 'create':
        this.applyResource();
        break;
      case 'update':
        this.applyResource();
        break;
      default:
        this.snackBar.open('Invalid action', 'Dismiss', { duration: 5000 });
        break;
    }
  }

  handleRowChange(rowData: object) {
    this.currentExpandedRow = rowData;
  }

  toggleDetails() {
    this.isShowDetails = !this.isShowDetails;
  }

  private getResource() {
    this.kubectlService.getResources(
      this.kubectlForm.value.namespace || 'all',
      this.kubectlForm.value.resource.resource.name,
      this.kubectlForm.value.resourceName,
    ).subscribe({
      next: (manifest) => {
        // this.resourceDefinition = manifest;
        this.handleCustomValuesChange(JSON.stringify(manifest));
      },
      error: (error) => this.handleError(error)
    });
  }

  private listResources() {
    this.kubectlService.getResources(
      this.kubectlForm.value.namespace || 'all',
      this.kubectlForm.value.resource.resource.name,
      '',
    ).subscribe({
      next: (manifests) => {
        this.manifests = manifests;
      },
      error: (error) => this.handleError(error)
    });
  }

  private applyResource() {
    this.kubectlService.applyResource(
      this.kubectlForm.value.namespace || 'all',
      this.kubectlForm.value.resource.resource.name,
      this.kubectlForm.value.resourceName,
      this.resourceDefinition,
    ).subscribe({
      next: (manifests) => {
        this.manifests = manifests;
      },
      error: (error) => this.handleError(error)
    });
  }

  private deleteResource() {
    this.kubectlService.deleteResource(
      this.kubectlForm.value.namespace || 'all',
      this.kubectlForm.value.resource.resource.name,
      this.kubectlForm.value.resourceName,
    ).subscribe({
      next: (res) => {
        this.snackBar.open(`${JSON.stringify(res)}`, 'Dismiss', { duration: 5000 });
      },
      error: (error) => this.handleError(error)
    });
  }

  private handleError(error: object) {
    this.snackBar.open(`Error loading data: ${JSON.stringify(error)}`, 'Dismiss', { duration: 5000 });
  }

}
