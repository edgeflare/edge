import { Pipe, PipeTransform } from '@angular/core';
import { jsonToYaml } from '@shared/utils';

@Pipe({
  name: 'yaml',
  standalone: true
})
export class YamlPipe implements PipeTransform {
  transform(value: any): string {
    return jsonToYaml(value);
  }

}
