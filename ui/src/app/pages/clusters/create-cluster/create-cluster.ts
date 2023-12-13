import { Component, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatSelectModule } from '@angular/material/select';
import { K3sService } from '@services';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { Router, RouterModule } from '@angular/router';
import { Editor } from '@components/editor';

@Component({
  selector: 'e-create-cluster',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, MatFormFieldModule, MatInputModule, MatButtonModule, MatSelectModule,
    MatSnackBarModule, MatSlideToggleModule, MatProgressSpinnerModule, RouterModule, Editor],
  templateUrl: './create-cluster.html',
  styles: ``
})
export class CreateCluster implements OnInit {
  k3sForm!: FormGroup;
  k3sVersions: string[] = [];
  isLoading = false;
  clusterId = '';
  logs = '';

  private fb = inject(FormBuilder);
  private k3sService = inject(K3sService);
  private snackBar = inject(MatSnackBar);
  private router = inject(Router);

  ngOnInit(): void {
    this.k3sService.getK3sVersions().subscribe({
      next: (versions) => {
        this.k3sVersions = versions;
        this.initForm();
      },
      error: (error) => {
        this.snackBar.open(`Error loading k3s versions: ${error}`, 'Dismiss', {
          duration: 5000,
        });
      },
      complete: () => { }
    });
  }

  onSubmit() {
    this.isLoading = true;
    this.logs = '';
    this.snackBar.open(`Creating cluster...`, 'Dismiss', { duration: 5000 });

    this.k3sService.createCluster(this.k3sForm.value).subscribe({
      next: (response) => {
        this.isLoading = false;
        // Split the response by newline and find the JSON part
        const lines = response.split('\n');
        const cluster_id_jsonLine = lines.find((line: string) => line.startsWith('{"cluster_id":'));
        if (cluster_id_jsonLine) {
          const cluster_id_json = JSON.parse(cluster_id_jsonLine);
          this.snackBar.open(`Created cluster ${cluster_id_json.cluster_id}`, 'Dismiss', { duration: 5000 });
          this.clusterId = cluster_id_json.cluster_id;
        }
        this.logs = lines.join('\n'); // Show all lines including the JSON as logs
      },
      error: (error) => {
        this.isLoading = false;
        console.error(error);
        this.snackBar.open(`Error creating cluster: ${error.message}`, 'Dismiss', { duration: 5000 });
      }
    });
  }

  private initForm() {
    this.k3sForm = this.fb.group({
      cluster: [false, Validators.required],
      ssh: this.fb.group({
        host: ['', [Validators.required, /* custom validators */]],
        user: ['', Validators.required],
        password: [''],
        keyfile: [''],
        port: [22]
      }),
      tls_san: [''],
      k3s_args: [''],
      version: [this.k3sVersions[0] || ''], // Sets the latest version as default
      dl_kubeconfig: [true],
    });
  }

}
