import { Component, ElementRef, ViewChild, OnInit, AfterViewInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import { Subscription } from 'rxjs';
import { WebsocketService } from '@services';
import { AttachAddon } from 'xterm-addon-attach';

@Component({
  selector: 'e-xterm',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './xterm.html',
  styleUrl: `xterm.scss`,
})
export class Xterm implements OnInit, AfterViewInit, OnDestroy {
  private terminal!: Terminal;
  private fitAddon!: FitAddon;
  private webSocketSubscription!: Subscription;

  @ViewChild('terminalContainer') terminalContainer!: ElementRef;

  constructor(private websocketService: WebsocketService) { }

  ngOnInit(): void {
    this.terminal = new Terminal({
      cursorBlink: true,
      letterSpacing: 0,
      convertEol: true,
    });
    this.fitAddon = new FitAddon();
    this.terminal.loadAddon(this.fitAddon);

    this.websocketService.connect('ws://localhost:8080/api/ws');

    this.webSocketSubscription = this.websocketService.messages.subscribe({
      next: (message: string) => {
        this.terminal.write(message);
      },
      error: (error: any) => {
        console.error('WebSocket error:', error);
      },
      complete: () => {
        console.log('WebSocket connection closed');
      }
    }
      // message => {
      //   this.terminal.write(message);
      // },
      // error => console.error('WebSocket error:', error),
      // () => console.log('WebSocket connection closed')
    );

    this.terminal.focus();
  }

  ngAfterViewInit(): void {
    this.terminal.open(this.terminalContainer.nativeElement);
    this.fitAddon.fit();
    window.addEventListener('resize', () => this.fitAddon.fit());



    let inputBuffer = '';

    this.terminal.onKey(e => {
      const printable = !e.domEvent.altKey && !e.domEvent.ctrlKey && !e.domEvent.metaKey;

      // Handle Enter
      if (e.domEvent.keyCode === 13) {
        this.terminal.write('\r\n');
        this.websocketService.sendMessage(inputBuffer);
        inputBuffer = ''; // Clear buffer after sending
      } else if (e.domEvent.keyCode === 8) {
        // Handle Backspace
        if (this.terminal.buffer.active.cursorX > 2) {
          this.terminal.write('\b \b');
          if (inputBuffer.length > 0) {
            inputBuffer = inputBuffer.substr(0, inputBuffer.length - 1);
          }
        }
      } else if (printable) {
        inputBuffer += e.key;
        this.terminal.write(e.key);
      }

    });


  }

  ngOnDestroy(): void {
    this.webSocketSubscription.unsubscribe();
    this.websocketService.disconnect();
  }
}


// Add more conditions here for other keystrokes like Ctrl+C etc.

// ngAfterViewInit(): void {
//   this.terminal.open(this.terminalContainer.nativeElement);
//   this.fitAddon.fit();
//   window.addEventListener('resize', () => this.fitAddon.fit());

//   const attachAddon = new AttachAddon(this.websocketService.webSocket, { bidirectional: true });
//   this.terminal.loadAddon(attachAddon);
// }


// const attachAddon = new AttachAddon(this.websocketService.webSocket, { bidirectional: true });
// this.terminal.loadAddon(attachAddon);

// this.terminal.onData(data => {
//   this.websocketService.sendMessage(data);
// });

// this.terminal.onData(data => {
//   // Handle user input data - send to WebSocket or process locally
//   this.websocketService.sendMessage(data);
// });

// import { Component, ElementRef, ViewChild } from '@angular/core';
// import { CommonModule } from '@angular/common';
// import { Terminal } from 'xterm';
// import { FitAddon } from 'xterm-addon-fit';

// @Component({
//   selector: 'e-xterm',
//   standalone: true,
//   imports: [CommonModule],
//   templateUrl: './xterm.html',
//   styles: ``
// })
// export class Xterm {
//   private terminal!: Terminal;
//   private fitAddon!: FitAddon;

//   @ViewChild('terminalContainer') terminalContainer!: ElementRef;

//   constructor() { }

//   ngOnInit(): void {
//     this.terminal = new Terminal({ cursorBlink: true });
//     this.fitAddon = new FitAddon();
//     this.terminal.loadAddon(this.fitAddon);
//   }

//   ngAfterViewInit(): void {
//     this.terminal.open(this.terminalContainer.nativeElement);
//     this.fitAddon.fit();
//     window.addEventListener('resize', () => this.fitAddon.fit());
//   }
// }
