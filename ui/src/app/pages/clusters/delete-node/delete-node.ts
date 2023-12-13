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
import { Router, RouterModule } from '@angular/router';
import { Editor } from '@components/editor';
import { K3sService } from '@services';
import { Subject, takeUntil } from 'rxjs';
import { K3sNode } from '@interfaces';

@Component({
  selector: 'e-delete-node',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, MatFormFieldModule, MatInputModule, MatButtonModule, MatSelectModule,
    MatSnackBarModule, MatSlideToggleModule, MatProgressSpinnerModule, RouterModule, Editor],
  templateUrl: './delete-node.html',
  styles: ``
})
export class DeleteNode implements OnInit {
  @Input() nodeId!: string;
  @Input() clusterId!: string;
  k3sUninstallForm!: FormGroup;
  nodes!: K3sNode[];
  node!: K3sNode;
  isLoading = false;
  logs = '';

  private fb = inject(FormBuilder);
  private k3sService = inject(K3sService);
  private snackBar = inject(MatSnackBar);
  private router = inject(Router);
  private destroy$ = new Subject<void>();

  ngOnInit(): void {
    this.k3sService.nodes$.pipe(
      takeUntil(this.destroy$)
    ).subscribe({
      next: (nodes) => {
        this.nodes = nodes;
        // Find the node that matches this.nodeId
        const node = this.nodes.find(node => node.id === this.nodeId);
        if (node !== undefined) {
          this.node = node;
          this.initForm();
        }

        this.isLoading = false;
      },
      error: (error) => {
        this.snackBar.open(`Error fetching nodes ${error.message}`, 'Dismiss', { duration: 5000 });
        this.isLoading = false;
      },
      complete: () => {}
    });

    this.k3sService.getNodes(this.clusterId);
  }

  onSubmit() {
    this.isLoading = true;
    this.logs = '';
    this.snackBar.open(`Uninstalling k3s...`, 'Dismiss', { duration: 5000 });

    console.log(this.k3sUninstallForm.value);

    this.k3sService.uninstallK3s(this.k3sUninstallForm.value, this.clusterId, this.nodeId).subscribe({
      next: (response) => {
        this.isLoading = false;
        // Split the response by newline and find the JSON part
        const lines = response.split('\n');
        const nodeId_jsonLine = lines.find((line: string) => line.startsWith('{"uninstalled":'));
        if (nodeId_jsonLine) {
          const nodeId_json = JSON.parse(nodeId_jsonLine);
          this.snackBar.open(`Uninstalled K3s on ${nodeId_json.uninstalled}`, 'Dismiss', { duration: 5000 });

          this.router.navigate(['/clusters']);
        }
        this.logs = lines.join('\n'); // Show all lines including the JSON as logs
      },
      error: (error) => {
        this.isLoading = false;
        console.error(error);
        this.snackBar.open(`Error uninstalling k3s: ${error.message}`, 'Dismiss', { duration: 5000 });
      }
    });
  }

  private initForm() {
    this.k3sUninstallForm = this.fb.group({
      ssh: this.fb.group({
        host: [this.node.ip, [Validators.required, /* custom validators */]],
        user: ['', Validators.required],
        password: [''],
        keyfile: [''],
        port: [22]
      }),
      agent: [this.node.role === 'agent'],
    });
  }
}
