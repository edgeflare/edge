import { Pipe, PipeTransform } from '@angular/core';
import { ipNetToCidr } from '@shared/utils';
import { IPNet } from '@shared/interfaces';

@Pipe({
  name: 'ipnettocidr', // check naming convention
  standalone: true,
})
export class ipNetToCidrPipe implements PipeTransform {
  transform(ipNet: IPNet): string {
    return ipNetToCidr(ipNet);
  }
}
