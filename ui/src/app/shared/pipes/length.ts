import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
  name: 'length',
  standalone: true
})
export class LengthPipe implements PipeTransform {

  transform(value: any | null): number {
    if (!value || typeof value !== 'object') {
      return value?.length || 0;
    }
    return Object.keys(value).length;
  }

}
