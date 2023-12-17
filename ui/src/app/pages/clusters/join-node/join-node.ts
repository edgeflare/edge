import { Component, Input, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { RouterModule } from '@angular/router';
import { Editor } from '@components/editor';
import { K3sService } from '@services';
import { Subject, takeUntil } from 'rxjs';
import { K3sCluster } from '@app/shared/interfaces/cluster';

@Component({
  selector: 'e-join-node',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, MatFormFieldModule, MatInputModule, MatButtonModule, MatSelectModule,
    MatSnackBarModule, MatSlideToggleModule, MatProgressSpinnerModule, Editor, RouterModule],
  templateUrl: './join-node.html',
  styles: ``
})
export class JoinNode implements OnInit {
  @Input() clusterId!: string;
  cluster!: K3sCluster;
  clusters!: K3sCluster[];
  k3sJoinForm!: FormGroup;
  k3sVersions: string[] = [];
  isLoading = false;
  logs = '';

  private fb = inject(FormBuilder);
  private k3sService = inject(K3sService);
  private snackBar = inject(MatSnackBar);
  // private router = inject(Router);
  private destroy$ = new Subject<void>();

  ngOnInit(): void {
    this.k3sService.clusters$.pipe(
      takeUntil(this.destroy$)
    ).subscribe({
      next: (clusters) => {
        this.clusters = clusters;
        const cluster = this.clusters.find(cluster => cluster.id === this.clusterId);

        if (cluster !== undefined) {
          this.cluster = cluster;
          this.initForm();
        }

        this.isLoading = false;
      },
      error: (error) => {
        this.snackBar.open(`Error fetching clusters ${error.message}`, 'Dismiss', { duration: 5000 });
        this.isLoading = false;
      },
      complete: () => {}
    });

    this.k3sService.getClusters();
  }

  private initForm() {
    this.k3sJoinForm = this.fb.group({
      ssh: this.fb.group({
        host: ['', [Validators.required, /* custom validators */]],
        user: ['', Validators.required],
        password: [''],
        keyfile: [''],
        keypassphrase: [''],
        port: [22]
      }),
      server: [this.cluster.apiserver, Validators.required],
      token: [''],
      master: [false],
    });
  }

  onSubmit() {
    this.isLoading = true;
    this.logs = '';
    this.snackBar.open(`Joining node to k3s cluster...`, 'Dismiss', { duration: 5000 });

    this.k3sService.joinNode(this.k3sJoinForm.value, this.clusterId).subscribe({
      next: (response) => {
        this.isLoading = false;
        const lines = response.split('\n');
        const nodeId_jsonLine = lines.find((line: string) => line.startsWith('{"joined":'));
        if (nodeId_jsonLine) {
          const nodeId_json = JSON.parse(nodeId_jsonLine);
          this.snackBar.open(`Joined node ${nodeId_json.joined}`, 'Dismiss', { duration: 5000 });
        }
        this.logs = lines.join('\n');
      },
      error: (error) => {
        this.isLoading = false;
        console.error(error);
        this.snackBar.open(`Error joining node: ${error.message}`, 'Dismiss', { duration: 5000 });
      }
    });
  }

}
