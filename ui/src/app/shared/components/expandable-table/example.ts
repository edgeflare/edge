import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ExpandableTable } from '@components/expandable-table';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { Editor } from '@components/editor';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { Network } from '@interfaces';

@Component({
  selector: 'e-gateways',
  standalone: true,
  imports: [CommonModule, ExpandableTable, MatProgressSpinnerModule, Editor, MatIconModule, MatButtonModule],
  templateUrl: './example.html',
  styles: ``,
  providers: [],
})
export class Gateways implements OnInit {
  networks: Network[] | undefined = undefined;
  columns = ['name', 'nestedJson', 'domains', 'created_at', 'updated_at', 'jsonPipe'];
  cellDefs = ['name | uppercase', 'address_range.IP', 'domains | jq[0]', 'created_at | date:"short"', 'updated_at | date:"short"', 'address_range | json'];
  isShowDetails = false;
  currentExpandedRow?: Network;

  handleRowChange(rowData: Network) {
    this.currentExpandedRow = rowData;
  }

  toggleDetails() {
    this.isShowDetails = !this.isShowDetails;
  }

  ngOnInit() {
    setTimeout(() => {
      this.networks = JSON.parse(this.networksStr)
      this.networks?.map((network) => {
        network.created_at = new Date(network.created_at),
          network.updated_at = new Date(network.updated_at)
      })
    }, 5000) // simulate data fetching over network by waiting 5 seconds
  }

  networksStr = `[
    {
      "id": "c37c7625-4720-41fe-b9c4-b77a8fa9f7a0",
      "user_id": "123e4567-e89b-12d3-a456-426614174001",
      "name": "agile-eagle-net",
      "address_range": {
        "IP": "10.0.0.0",
        "Mask": "////AA=="
      },
      "domains": [
        "b77a8fa9f7a0.edgeflare.dev",
        "*.b77a8fa9f7a0.edgeflare.dev"
      ],
      "created_at": "2023-11-24T17:20:46.655+06:00",
      "updated_at": "2023-11-24T17:20:46.655+06:00"
    },
    {
      "id": "9b6c9576-ac24-4cee-8406-89a34414a2df",
      "user_id": "123e4567-e89b-12d3-a456-426614174001",
      "name": "jolly-jay-net",
      "address_range": {
        "IP": "10.0.0.0",
        "Mask": "////AA=="
      },
      "domains": [
        "89a34414a2df.edgeflare.dev",
        "*.89a34414a2df.edgeflare.dev"
      ],
      "created_at": "2023-11-24T17:20:49.475+06:00",
      "updated_at": "2023-11-24T17:20:49.475+06:00"
    },
    {
      "id": "4325d333-0211-4acb-87cc-6beaa9e1d188",
      "user_id": "123e4567-e89b-12d3-a456-426614174001",
      "name": "daring-canary-net",
      "address_range": {
        "IP": "10.0.0.0",
        "Mask": "////AA=="
      },
      "domains": [
        "6beaa9e1d188.edgeflare.dev",
        "*.6beaa9e1d188.edgeflare.dev"
      ],
      "created_at": "2023-11-24T19:37:34.28+06:00",
      "updated_at": "2023-11-24T19:37:34.28+06:00"
    }
  ]`
}
