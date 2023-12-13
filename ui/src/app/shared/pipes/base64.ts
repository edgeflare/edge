import { Pipe, PipeTransform } from '@angular/core';
import { decodeBase64 } from '@shared/utils';

@Pipe({
  name: 'decodeBase64',
  standalone: true
})
export class DecodeBase64Pipe implements PipeTransform {
  transform(value: string): string {
    return decodeBase64(value);
  }
}
