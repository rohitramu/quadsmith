import { createClient } from "@connectrpc/connect";
import { AuthService } from "@quadsmith/shared/gen/quadsmith/api/auth_pb";
import { UserService } from "@quadsmith/shared/gen/quadsmith/api/user_pb";
import { createApiTransport, getStoredAccessToken, setStoredAccessToken } from "@quadsmith/shared";

const isDev = import.meta.env.DEV;

export const setAccessToken = setStoredAccessToken;
export const getAccessToken = getStoredAccessToken;

export const transport = createApiTransport({
  baseUrl: isDev ? "http://localhost:8080" : "",
});

export const authClient = createClient(AuthService, transport);
export const userClient = createClient(UserService, transport);
