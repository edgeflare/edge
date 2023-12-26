import { Injectable } from '@angular/core';
import { Observable, Subject } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class WebsocketService {
  webSocket!: WebSocket;
  private messageSubject: Subject<string>;
  public messages: Observable<string>;

  constructor() {
    this.messageSubject = new Subject<string>();
    this.messages = this.messageSubject.asObservable();
  }

  public connect(url: string): void {
    this.webSocket = new WebSocket(url);

    this.webSocket.onopen = event => {
      console.log('WebSocket connection established', event);
    };

    this.webSocket.onmessage = event => {
      this.messageSubject.next(event.data as string);
    };

    this.webSocket.onerror = event => {
      console.error('WebSocket error', event);
      this.messageSubject.error(event);
    };

    this.webSocket.onclose = event => {
      console.log('WebSocket connection closed', event);
      this.messageSubject.complete();
    };
  }

  public sendMessage(message: string): void {
    if (this.webSocket && this.webSocket.readyState === WebSocket.OPEN) {
      this.webSocket.send(message);
    } else {
      console.error('WebSocket is not connected.');
    }
  }

  public disconnect(): void {
    if (this.webSocket) {
      this.webSocket.close();
    }
  }
}


