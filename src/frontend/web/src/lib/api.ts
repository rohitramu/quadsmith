import { createConnectTransport } from "@connectrpc/connect-web";

const isDev = import.meta.env.DEV;

export const transport = createConnectTransport({
  // Use relative path in production since it's served by the Go backend itself
  baseUrl: isDev ? "http://localhost:8080" : "", 
});
