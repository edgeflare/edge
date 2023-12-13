import { Pipe, PipeTransform } from '@angular/core';
import { convertMarkdownToHtml } from '@shared/utils';

@Pipe({
  name: 'markdownToHtml',
  standalone: true
})
export class MarkdownToHtmlPipe implements PipeTransform {
  transform(markdown: string): string {
    return convertMarkdownToHtml(markdown);
  }
}
