import { Component, Inject, EventEmitter, Output } from '@angular/core';
import { MatDialogRef, MAT_DIALOG_DATA, MatDialogModule } from '@angular/material/dialog';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { HttpClient, HttpClientModule } from '@angular/common/http';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatButtonModule } from '@angular/material/button';
import { MatInputModule } from '@angular/material/input';
import { ConfirmDeleteDialogData } from '@interfaces';

@Component({
  selector: 'e-confirm-delete-dialog',
  standalone: true,
  imports: [CommonModule, HttpClientModule, MatSnackBarModule, MatDialogModule, FormsModule,
    MatFormFieldModule, MatButtonModule, MatInputModule],
  templateUrl: './confirm-delete-dialog.html',
})
export class ConfirmDeleteDialog {
  @Output() deletionComplete = new EventEmitter<void>();
  inputName: string = '';
  itemName: string;
  deleteUrl: string;
  itemType!: string;

  constructor(
    public dialogRef: MatDialogRef<ConfirmDeleteDialog>,
    @Inject(MAT_DIALOG_DATA) public data: ConfirmDeleteDialogData,
    private http: HttpClient,
    private snackBar: MatSnackBar
  ) {
    this.itemName = data.itemName;
    this.deleteUrl = data.deleteUrl;
    this.itemType = data.itemType;
  }

  confirmDelete() {
    this.http.delete(this.deleteUrl).subscribe({
      next: () => {
        this.snackBar.open(`${this.itemName} ${this.itemType} deleted successfully`, 'Close', { duration: 3000 });
        this.deletionComplete.emit();
        this.dialogRef.close();
      },
      error: () => {
        this.snackBar.open(`Error deleting ${this.itemName} ${this.itemType}`, 'Close', { duration: 3000 });
      }
    });
  }
}
