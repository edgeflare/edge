import * as marked from 'marked';

export function convertMarkdownToHtml(markdown: string): string {
  return marked.parse(markdown);
}
