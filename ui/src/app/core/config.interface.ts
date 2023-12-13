export interface Config {
  auth: {
    type: string;
    jwt?: JwtAuthConfig;
  };
}

interface JwtAuthConfig {
  isssuer: string;
  clientId: string;
}
