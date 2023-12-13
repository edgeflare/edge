import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
  name: 'timeago',
  standalone: true,
})
export class TimeAgoPipe implements PipeTransform {

  transform(value: Date | string | number): string {
    if (value) {
      const now = new Date();
      const date = new Date(value);
      const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);

      // if (seconds < 60) {
      //   return 'Just now';
      // }
      const intervals: { [key: string]: number } = {
        'y': 31536000,
        'mo': 2592000,
        'w': 604800,
        'd': 86400,
        'h': 3600,
        'm': 60,
        's': 1
      };
      let counter;
      for (const i in intervals) {
        counter = Math.floor(seconds / intervals[i]);
        if (counter > 0)
          if (counter === 1) {
            return `${counter}${i}`; // singular (1 day ago)
          } else {
            return `${counter}${i}`; // plural (2 days ago)
          }
        // return `${counter}${i}`
      }
    }
    return '';
  }
}
