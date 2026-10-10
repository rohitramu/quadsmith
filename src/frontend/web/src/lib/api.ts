import { createConnectTransport } from "@connectrpc/connect-web";
import { createClient } from "@connectrpc/connect";
import { AuthService } from "../gen/quadsmith/auth_pb";
import { UserService } from "../gen/quadsmith/user_pb";

const isDev = import.meta.env.DEV;

let currentAccessToken: string | null = localStorage.getItem("qs_access_token");

export function setAccessToken(token: string | null) {
  currentAccessToken = token;
  if (token) {
    localStorage.setItem("qs_access_token", token);
  } else {
    localStorage.removeItem("qs_access_token");
  }
}

export function getAccessToken(): string | null {
  return currentAccessToken || localStorage.getItem("qs_access_token");
}

export const transport = createConnectTransport({
  baseUrl: isDev ? "http://localhost:8080" : "",
  fetch: (input, init) => {
    return fetch(input, {
      ...init,
      credentials: "include",
    });
  },
  interceptors: [
    (next) => async (req) => {
      const token = getAccessToken();
      if (token) {
        req.header.set("Authorization", `Bearer ${token}`);
      }
      return await next(req);
    },
  ],
});

export const authClient = createClient(AuthService, transport);
export const userClient = createClient(UserService, transport);
