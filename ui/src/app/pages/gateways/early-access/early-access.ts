import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { HttpClient, HttpClientModule } from '@angular/common/http';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { environment } from '@env';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';

@Component({
  selector: 'e-early-access',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, MatFormFieldModule, MatInputModule, MatButtonModule, MatSnackBarModule, HttpClientModule],
  templateUrl: './early-access.html',
  styles: ``
})
export class EarlyAccess {

  private fb = inject(FormBuilder);
  private http = inject(HttpClient);
  private snackBar = inject(MatSnackBar);

  contactForm = this.fb.group({
    contact_name: ['', Validators.required],
    contact_email: ['', [Validators.required, Validators.email]],
  });

  formSubmitted = false;

  onSubmit() {
    this.http.post(`${environment.api}/gateways/early_access`, this.contactForm.value).subscribe({
      next: data => {
        console.log('Response from the server: ', data);
        this.snackBar.open('Form submitted successfully', 'Close', {
          duration: 3000,
        });
        this.contactForm.reset(); // reset the form after successful submission
        this.contactForm.clearValidators();
        this.formSubmitted = true;
      },
      error: error => {
        this.snackBar.open(`Error submitting form: ${error.message}`, 'Close', {
          duration: 3000,
        });
      }
    });
  }

}
