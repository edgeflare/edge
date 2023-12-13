export enum PeerType {
  // Gateway = "gateway", // reserved
  OriginServer = "origin_server",
  Client = "client"
}

export interface IPNet {
  IP: string;
  Mask: string;
}

export interface Network {
  id?: string;
  user_id?: string;
  name: string;
  address_range: IPNet | string | any;
  created_at: Date | string;
  updated_at: Date | string;
  domains: string[];
}

export interface Peer {
  id?: string;
  name: string;
  type: PeerType;
  address?: IPNet | string | any;
  user_id?: string;
  network_id?: string;
  public_key?: string;
  allowed_ips: IPNet[] | string[] | any[];
  endpoint?: string;
  latest_handshake?: Date;
  created_at?: Date;
  updated_at?: Date;
  is_gateway?: boolean;
  description?: string;
}

export interface Domain {
  domain: string;
  active: boolean;
}
