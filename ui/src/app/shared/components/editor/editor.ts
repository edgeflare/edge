import { CommonModule } from '@angular/common';
import { Component, ViewChild, ElementRef, EventEmitter, Output, Input, AfterViewInit } from '@angular/core';
import * as ace from 'ace-builds';

@Component({
  selector: 'e-editor',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './editor.html',
})
export class Editor implements AfterViewInit {
  @ViewChild("editor") private editor!: ElementRef<HTMLElement>;
  @Output() contentChange = new EventEmitter<string>();
  @Input() inputText: string = "";
  @Input() isReadOnly: boolean = false;
  @Input() height: string = "60vh";
  @Input() width: string = "100%";
  @Input() fontSize: string = "16px";
  @Input() theme: string = "xcode";
  @Input() mode: string = "yaml";

  ngAfterViewInit(): void {
    ace.config.set("fontSize", this.fontSize);
    ace.config.set(
      "basePath",
      "https://unpkg.com/ace-builds@1.31.1/src-noconflict"
    );
    const aceEditor = ace.edit(this.editor.nativeElement);
    aceEditor.session.setValue(this.inputText);
    aceEditor.setTheme(`ace/theme/${this.theme}`);
    aceEditor.session.setMode(`ace/mode/${this.mode}`);
    aceEditor.setReadOnly(this.isReadOnly);

    aceEditor.on("change", () => {
      this.contentChange.emit(aceEditor.getValue()); // Emit the content on change
    });
  }
}
