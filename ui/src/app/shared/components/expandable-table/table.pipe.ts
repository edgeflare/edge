import { CurrencyPipe, DatePipe, DecimalPipe, JsonPipe, KeyValuePipe, LowerCasePipe, PercentPipe, SlicePipe, TitleCasePipe, UpperCasePipe } from '@angular/common';
import { Pipe, PipeTransform, Injector } from '@angular/core';
import { TimeAgoPipe } from '@app/shared/pipes';

@Pipe({
  name: 'table',
  standalone: true,
})
export class TablePipe implements PipeTransform {
  constructor(private injector: Injector) { }

  transform(value: any, path: string): any {
    if (!value || !path) {
      return value;
    }

    const [pathPart, pipeString] = path.split('|').map(part => part.trim());
    let result = this.resolvePath(value, pathPart);

    if (pipeString) {
      result = this.applyPipe(result, pipeString);
    }

    return result;
  }

  private resolvePath(value: any, path: string): any {
    return path.split('.').reduce((currentObject, part) => {
      if (!currentObject) return null;

      const arrayMatch = part.match(/^(.+)\[(\d+)\]$/);
      if (arrayMatch) {
        const [_, arrayName, index] = arrayMatch;
        currentObject = currentObject[arrayName];
        if (!Array.isArray(currentObject)) {
          // console.warn(`The specified property ${arrayName} is not an array.`);
          return null;
        }
        return currentObject[parseInt(index, 10)];
      }

      return currentObject[part];
    }, value);
  }

  private applyPipe(value: any, pipeString: string): any {
    const [pipeName, ...args] = pipeString.split(':').map(arg => arg.trim());
    const processedArgs = args.map(arg => arg.replace(/^"(.*)"$/, '$1'));
    const pipeToken = this.getPipeToken(pipeName);

    if (!pipeToken) {
      return value;
    }

    const pipe = this.injector.get<PipeTransform>(pipeToken);
    try {
      return pipe.transform(value, ...processedArgs);
    } catch (error) {
      return value;
    }
  }

  private getPipeToken(pipeName: string): any {
    const pipes: { [key: string]: any } = {
      'date': DatePipe,
      'keyvalue': KeyValuePipe,
      'slice': SlicePipe,
      'json': JsonPipe,
      'uppercase': UpperCasePipe,
      'lowercase': LowerCasePipe,
      'titlecase': TitleCasePipe,
      'currency': CurrencyPipe,
      'number': DecimalPipe,
      'percent': PercentPipe,
      'timeago': TimeAgoPipe,
    };

    return pipes[pipeName];
  }
}
